#!/usr/bin/env python3
"""Fail-closed controls for the Creator Signal Forgejo mirror and releases."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Any


ROOT = Path(__file__).resolve().parents[2]
POLICY_PATH = ROOT / "creator-signal" / "forgejo-release-policy.json"
RELEASE_BRANCH = re.compile(r"^v\d+(?:\.\d+)?/forgejo$")
DIGEST = re.compile(r"^sha256:[0-9a-f]{64}$")


class ControlError(RuntimeError):
    """A governed mirror or release invariant failed."""


def run(*args: str, check: bool = True) -> str:
    result = subprocess.run(args, text=True, capture_output=True, check=False)
    if check and result.returncode:
        raise ControlError(
            f"{' '.join(args)} failed: {(result.stderr or result.stdout).strip()}"
        )
    return result.stdout.strip()


def run_bytes(*args: str, check: bool = True) -> bytes:
    result = subprocess.run(args, capture_output=True, check=False)
    if check and result.returncode:
        error = (result.stderr or result.stdout).decode("utf-8", errors="replace")
        raise ControlError(f"{' '.join(args)} failed: {error.strip()}")
    return result.stdout


def output(name: str, value: str) -> None:
    target = os.environ.get("GITHUB_OUTPUT")
    if target:
        with open(target, "a", encoding="utf-8") as stream:
            stream.write(f"{name}={value}\n")


def load_policy(path: Path = POLICY_PATH) -> dict[str, Any]:
    policy = json.loads(path.read_text(encoding="utf-8"))
    required = {
        "upstreamUrl",
        "automationBranch",
        "semanticTagPattern",
        "downstreamTagPattern",
        "legacyReleasePlatforms",
        "downstreamReleases",
        "variants",
        "nativePlatforms",
    }
    missing = sorted(required.difference(policy))
    if missing:
        raise ControlError(f"release policy is missing: {', '.join(missing)}")
    return policy


def refs(remote: str) -> tuple[dict[str, str], dict[str, str]]:
    heads: dict[str, str] = {}
    tags: dict[str, str] = {}
    peeled: dict[str, str] = {}
    for line in run("git", "ls-remote", remote).splitlines():
        sha, ref = line.split("\t", 1)
        if ref.startswith("refs/heads/"):
            heads[ref.removeprefix("refs/heads/")] = sha
        elif ref.startswith("refs/tags/"):
            name = ref.removeprefix("refs/tags/")
            if name.endswith("^{}"):
                peeled[name[:-3]] = sha
            else:
                tags[name] = sha
    tags.update(peeled)
    return heads, tags


def exact_commit_exists(sha: str) -> bool:
    if not re.fullmatch(r"[0-9a-f]{40}", sha):
        return False
    result = subprocess.run(
        ["git", "cat-file", "-e", f"{sha}^{{commit}}"], capture_output=True, check=False
    )
    return result.returncode == 0


def expected_platforms(tag: str, policy: dict[str, Any]) -> set[str]:
    downstream = policy["downstreamReleases"].get(tag)
    if downstream:
        return set(downstream["platforms"])
    legacy = policy["legacyReleasePlatforms"].get(tag)
    if legacy:
        return set(legacy)
    return {"linux/amd64", "linux/arm64"}


def source_patch_identity(
    base_sha: str, source_sha: str, changed_paths: list[str]
) -> tuple[str, list[str]]:
    actual_paths = run(
        "git", "diff", "--name-only", base_sha, source_sha, "--"
    ).splitlines()
    patch = run_bytes(
        "git",
        "diff",
        "--no-ext-diff",
        "--binary",
        "--abbrev=9",
        base_sha,
        source_sha,
        "--",
        *changed_paths,
    )
    return hashlib.sha256(patch).hexdigest(), actual_paths


def validate_downstream_source(
    tag: str, configured: dict[str, Any], upstream_tags: dict[str, str], origin_tags: dict[str, str]
) -> dict[str, Any]:
    base_tag = configured["baseTag"]
    base_sha = configured["baseSourceSha"]
    if upstream_tags.get(base_tag) != base_sha or origin_tags.get(base_tag) != base_sha:
        raise ControlError(
            f"downstream base mismatch for {base_tag}: "
            f"upstream={upstream_tags.get(base_tag, 'missing')} "
            f"GitHub={origin_tags.get(base_tag, 'missing')} policy={base_sha}"
        )
    if tag in upstream_tags:
        raise ControlError(f"Creator Signal downstream tag collides with upstream: {tag}")
    source_sha = configured["sourceCommitSha"]
    if not exact_commit_exists(source_sha):
        raise ControlError(f"committed downstream source is unavailable: {source_sha}")
    parents = run("git", "rev-list", "--parents", "-n", "1", source_sha).split()
    if parents != [source_sha, base_sha]:
        raise ControlError(f"downstream source must have sole upstream parent {base_sha}")
    source_tree = run("git", "rev-parse", f"{source_sha}^{{tree}}")
    if source_tree != configured["sourceTreeSha"]:
        raise ControlError(
            f"downstream source tree mismatch: actual={source_tree} policy={configured['sourceTreeSha']}"
        )
    subject = run("git", "show", "-s", "--format=%s", source_sha)
    if subject != configured["sourceCommitSubject"]:
        raise ControlError("downstream source subject differs from policy")
    patch_digest, actual_paths = source_patch_identity(
        base_sha, source_sha, configured["changedPaths"]
    )
    if actual_paths != configured["changedPaths"]:
        raise ControlError(
            f"downstream changed paths differ: actual={actual_paths} policy={configured['changedPaths']}"
        )
    if patch_digest != configured["sourcePatchSha256"]:
        raise ControlError(
            f"downstream patch digest mismatch: actual={patch_digest} "
            f"policy={configured['sourcePatchSha256']}"
        )
    for path in configured["changedPaths"]:
        source = run("git", "show", f"{source_sha}:{path}")
        if source.count(configured["securityRefresh"]) != 1:
            raise ControlError(f"{path} does not contain the exact bounded security refresh once")
        for image in configured["baseImages"].values():
            token = f"{image['reference']}@{image['digest']}"
            if token not in source:
                raise ControlError(f"{path} is missing pinned base input {token}")
    existing = origin_tags.get(tag)
    if existing and existing != source_sha:
        raise ControlError(
            f"immutable downstream tag mismatch: GitHub={existing} policy={source_sha}"
        )
    return {
        "baseTag": base_tag,
        "baseSourceSha": base_sha,
        "sourceSha": source_sha,
        "sourceTreeSha": source_tree,
        "sourcePatchSha256": patch_digest,
        "tagExists": bool(existing),
    }


def sync(args: argparse.Namespace) -> None:
    if args.mode not in {"dry-run", "apply"}:
        raise ControlError("mode must be dry-run or apply")
    policy = load_policy()
    upstream = policy["upstreamUrl"]
    run("git", "remote", "remove", "creator-signal-upstream", check=False)
    run("git", "remote", "add", "creator-signal-upstream", upstream)
    run(
        "git",
        "fetch",
        "--no-tags",
        "creator-signal-upstream",
        "+refs/heads/*:refs/remotes/creator-signal-upstream/*",
    )
    upstream_heads, upstream_tags = refs("creator-signal-upstream")
    origin_heads, origin_tags = refs("origin")
    selected_heads = {
        name: sha
        for name, sha in upstream_heads.items()
        if name == "forgejo" or RELEASE_BRANCH.fullmatch(name)
    }
    semantic = re.compile(policy["semanticTagPattern"])
    selected_tags = {name: sha for name, sha in upstream_tags.items() if semantic.fullmatch(name)}
    mismatches = [
        {"tag": name, "origin": origin_tags[name], "upstream": sha}
        for name, sha in selected_tags.items()
        if name in origin_tags and origin_tags[name] != sha
    ]
    if mismatches:
        raise ControlError(f"immutable upstream tag mismatch: {json.dumps(mismatches, sort_keys=True)}")
    branch_updates = []
    for name, sha in sorted(selected_heads.items()):
        current = origin_heads.get(name)
        if current == sha:
            continue
        if current:
            ancestor = subprocess.run(
                ["git", "merge-base", "--is-ancestor", current, sha], check=False
            ).returncode == 0
            if not ancestor:
                raise ControlError(f"non-fast-forward upstream branch refused: {name} {current} -> {sha}")
        branch_updates.append({"name": name, "from": current, "to": sha})
    new_tags = [
        {"name": name, "sha": sha}
        for name, sha in sorted(selected_tags.items())
        if name not in origin_tags
    ]
    downstream_tags = []
    for name, configured in sorted(policy["downstreamReleases"].items()):
        if name in upstream_tags:
            raise ControlError(f"reserved downstream tag unexpectedly exists upstream: {name}")
        actual = origin_tags.get(name)
        expected = configured["sourceCommitSha"]
        if not actual:
            action = "reserved-unpublished"
        elif actual == expected:
            action = "immutable-downstream"
        else:
            raise ControlError(
                f"immutable downstream tag mismatch: {name} GitHub={actual} expected={expected}"
            )
        downstream_tags.append(
            {"name": name, "action": action, "github": actual or "missing", "expected": expected}
        )
    report = {
        "schema": "creator-signal.forgejo-upstream-sync/v2",
        "mode": args.mode,
        "upstream": upstream,
        "branchUpdates": branch_updates,
        "newTags": new_tags,
        "downstreamTags": downstream_tags,
        "tagMismatches": [],
    }
    Path(args.report).write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    if args.mode == "apply":
        for item in branch_updates:
            run("git", "push", "origin", f"{item['to']}:refs/heads/{item['name']}")
        for item in new_tags:
            run("git", "push", "origin", f"{item['sha']}:refs/tags/{item['name']}")


def plan_tag(args: argparse.Namespace) -> dict[str, Any]:
    policy = load_policy()
    if not re.fullmatch(policy["semanticTagPattern"], args.tag):
        raise ControlError("tag is outside the governed semantic-version policy")
    _, upstream_tags = refs(policy["upstreamUrl"])
    _, origin_tags = refs("origin")
    downstream = policy["downstreamReleases"].get(args.tag)
    if downstream:
        identity = validate_downstream_source(args.tag, downstream, upstream_tags, origin_tags)
        source_ref = identity["sourceSha"]
        is_downstream = True
    else:
        upstream_sha = upstream_tags.get(args.tag)
        origin_sha = origin_tags.get(args.tag)
        if not upstream_sha or upstream_sha != origin_sha:
            raise ControlError(
                f"tag source mismatch: mirror={origin_sha or 'missing'} upstream={upstream_sha or 'missing'}"
            )
        identity = {
            "baseTag": "",
            "baseSourceSha": "",
            "sourceSha": run("git", "rev-parse", f"{args.tag}^{{commit}}"),
            "sourceTreeSha": run("git", "rev-parse", f"{args.tag}^{{tree}}"),
            "sourcePatchSha256": "",
            "tagExists": True,
        }
        source_ref = args.tag
        is_downstream = False
    release = subprocess.run(
        ["gh", "release", "view", args.tag, "--repo", args.repository, "--json", "isDraft,url"],
        text=True,
        capture_output=True,
        check=False,
    )
    platforms = expected_platforms(args.tag, policy)
    plan = {
        "schema": "creator-signal.forgejo-release-plan/v2",
        "upstream": policy["upstreamUrl"],
        "tag": args.tag,
        "version": args.tag[1:],
        "sourceRef": source_ref,
        "sourceSha": identity["sourceSha"],
        "sourceTreeSha": identity["sourceTreeSha"],
        "sourcePatchSha256": identity["sourcePatchSha256"],
        "baseTag": identity["baseTag"],
        "baseSourceSha": identity["baseSourceSha"],
        "downstream": is_downstream,
        "tagExists": identity["tagExists"],
        "prerelease": bool("-" in args.tag),
        "releaseExists": release.returncode == 0,
        "platforms": sorted(platforms),
    }
    report_path = getattr(args, "report", None)
    if report_path:
        Path(report_path).write_text(json.dumps(plan, indent=2) + "\n", encoding="utf-8")
    values = {
        "source_ref": source_ref,
        "source_sha": identity["sourceSha"],
        "source_tree_sha": identity["sourceTreeSha"],
        "source_patch_sha256": identity["sourcePatchSha256"],
        "base_tag": identity["baseTag"],
        "base_source_sha": identity["baseSourceSha"],
        "version": args.tag[1:],
        "downstream": str(is_downstream).lower(),
        "tag_exists": str(identity["tagExists"]).lower(),
        "prerelease": str(plan["prerelease"]).lower(),
        "release_exists": str(plan["releaseExists"]).lower(),
        "arm64_enabled": str("linux/arm64" in platforms).lower(),
    }
    for name, value in values.items():
        output(name, value)
    return plan


def publish_downstream_tag(args: argparse.Namespace) -> None:
    policy = load_policy()
    configured = policy["downstreamReleases"].get(args.tag)
    if not configured:
        raise ControlError("only a configured Creator Signal downstream tag may be published")
    plan = plan_tag(args)
    if plan["tagExists"]:
        return
    source_sha = configured["sourceCommitSha"]
    run("git", "push", "origin", f"{source_sha}:refs/tags/{args.tag}")
    _, origin_tags = refs("origin")
    if origin_tags.get(args.tag) != source_sha:
        raise ControlError("published downstream tag does not resolve to the governed source commit")


def platform_digests(data: dict[str, Any]) -> dict[str, str]:
    values = {
        f"{item.get('platform', {}).get('os')}/{item.get('platform', {}).get('architecture')}": item.get("digest", "")
        for item in data.get("manifests", [])
        if item.get("platform", {}).get("os") != "unknown"
    }
    for value in values.values():
        if not DIGEST.fullmatch(value):
            raise ControlError(f"invalid platform digest: {value}")
    return values


def verify_platforms(args: argparse.Namespace) -> None:
    policy = load_policy()
    data = json.loads(Path(args.file).read_text(encoding="utf-8"))
    actual = set(platform_digests(data))
    expected = expected_platforms(args.tag, policy)
    if actual != expected:
        raise ControlError(
            f"manifest platforms differ: expected={sorted(expected)} actual={sorted(actual)}"
        )


def write_record(args: argparse.Namespace) -> None:
    policy = load_policy()
    for value in (args.rootful_digest, args.rootless_digest):
        if not DIGEST.fullmatch(value):
            raise ControlError(f"invalid digest: {value}")
    expected = expected_platforms(args.tag, policy)
    manifests = {
        "rootful": json.loads(Path(args.rootful_manifest).read_text(encoding="utf-8")),
        "rootless": json.loads(Path(args.rootless_manifest).read_text(encoding="utf-8")),
    }
    platform_maps = {name: platform_digests(data) for name, data in manifests.items()}
    for name, values in platform_maps.items():
        if set(values) != expected:
            raise ControlError(f"{name} platform map differs from release policy")
    evidence = {
        path.name: "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest()
        for path in sorted(Path(args.evidence).glob("*"))
        if path.is_file()
    }
    configured = policy["downstreamReleases"].get(args.tag)
    source_identity: dict[str, Any] = {
        "sourceSha": args.source_sha,
        "sourceTreeSha": run("git", "rev-parse", f"{args.source_sha}^{{tree}}"),
    }
    if configured:
        source_identity.update(
            {
                "upstreamBaseTag": configured["baseTag"],
                "upstreamBaseSha": configured["baseSourceSha"],
                "sourcePatchSha256": configured["sourcePatchSha256"],
                "changedPaths": configured["changedPaths"],
                "baseImages": configured["baseImages"],
            }
        )
    record = {
        "schema": "creator-signal.forgejo-container-release/v2",
        "upstream": policy["upstreamUrl"],
        "tag": args.tag,
        "sourceIdentity": source_identity,
        "workflowUrl": args.workflow_url,
        "images": {
            "rootful": {
                "reference": f"ghcr.io/creator-signal/forgejo:{args.tag[1:]}",
                "digest": args.rootful_digest,
                "platforms": sorted(expected),
                "platformDigests": platform_maps["rootful"],
            },
            "rootless": {
                "reference": f"ghcr.io/creator-signal/forgejo:{args.tag[1:]}-rootless",
                "digest": args.rootless_digest,
                "platforms": sorted(expected),
                "platformDigests": platform_maps["rootless"],
            },
        },
        "evidence": evidence,
    }
    Path(args.output).write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")


def verify_record(args: argparse.Namespace) -> None:
    policy = load_policy()
    record = json.loads(Path(args.file).read_text(encoding="utf-8"))
    schema = record.get("schema")
    if schema not in {
        "creator-signal.forgejo-container-release/v1",
        "creator-signal.forgejo-container-release/v2",
    }:
        raise ControlError("release record schema mismatch")
    source_sha = (
        record.get("sourceSha")
        if schema.endswith("/v1")
        else record.get("sourceIdentity", {}).get("sourceSha")
    )
    if record.get("tag") != args.tag or source_sha != args.source_sha:
        raise ControlError("release record source identity mismatch")
    expected = expected_platforms(args.tag, policy)
    for variant in ("rootful", "rootless"):
        image = record.get("images", {}).get(variant, {})
        if not DIGEST.fullmatch(image.get("digest", "")) or set(image.get("platforms", [])) != expected:
            raise ControlError(f"invalid {variant} release record")
        if schema.endswith("/v2") and set(image.get("platformDigests", {})) != expected:
            raise ControlError(f"invalid {variant} platform digest inventory")
    configured = policy["downstreamReleases"].get(args.tag)
    if configured:
        identity = record.get("sourceIdentity", {})
        checks = {
            "sourceTreeSha": configured["sourceTreeSha"],
            "upstreamBaseTag": configured["baseTag"],
            "upstreamBaseSha": configured["baseSourceSha"],
            "sourcePatchSha256": configured["sourcePatchSha256"],
            "baseImages": configured["baseImages"],
        }
        for key, expected_value in checks.items():
            if identity.get(key) != expected_value:
                raise ControlError(f"release record downstream identity mismatch: {key}")


def parser() -> argparse.ArgumentParser:
    root = argparse.ArgumentParser()
    commands = root.add_subparsers(dest="command", required=True)
    command = commands.add_parser("sync")
    command.add_argument("--mode", required=True)
    command.add_argument("--repository", required=True)
    command.add_argument("--report", required=True)
    command.set_defaults(handler=sync)
    command = commands.add_parser("plan-tag")
    command.add_argument("--tag", required=True)
    command.add_argument("--repository", required=True)
    command.add_argument("--report", required=True)
    command.set_defaults(handler=plan_tag)
    command = commands.add_parser("publish-downstream-tag")
    command.add_argument("--tag", required=True)
    command.add_argument("--repository", required=True)
    command.set_defaults(handler=publish_downstream_tag)
    command = commands.add_parser("verify-platforms")
    command.add_argument("--file", required=True)
    command.add_argument("--tag", required=True)
    command.set_defaults(handler=verify_platforms)
    command = commands.add_parser("write-record")
    for name in (
        "tag",
        "source-sha",
        "rootful-digest",
        "rootless-digest",
        "rootful-manifest",
        "rootless-manifest",
        "workflow-url",
        "evidence",
        "output",
    ):
        command.add_argument(f"--{name}", required=True)
    command.set_defaults(handler=write_record)
    command = commands.add_parser("verify-record")
    for name in ("file", "tag", "source-sha"):
        command.add_argument(f"--{name}", required=True)
    command.set_defaults(handler=verify_record)
    return root


def main() -> int:
    try:
        args = parser().parse_args()
        args.handler(args)
        return 0
    except Exception as error:  # fail closed at the CLI boundary
        print(str(error), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
