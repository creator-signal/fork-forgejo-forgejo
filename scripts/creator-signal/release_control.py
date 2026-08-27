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

UPSTREAM = "https://codeberg.org/forgejo/forgejo.git"
SEMVER = re.compile(r"^v(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$")
RELEASE_BRANCH = re.compile(r"^v\d+(?:\.\d+)?/forgejo$")
DIGEST = re.compile(r"^sha256:[0-9a-f]{64}$")
PLATFORMS = {"linux/amd64"}


def run(*args: str, check: bool = True) -> str:
    result = subprocess.run(args, text=True, capture_output=True, check=False)
    if check and result.returncode:
        raise RuntimeError(f"{' '.join(args)} failed: {(result.stderr or result.stdout).strip()}")
    return result.stdout.strip()


def output(name: str, value: str) -> None:
    target = os.environ.get("GITHUB_OUTPUT")
    if target:
        with open(target, "a", encoding="utf-8") as stream:
            stream.write(f"{name}={value}\n")


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
    for name, sha in peeled.items():
        tags[name] = sha
    return heads, tags


def sync(args: argparse.Namespace) -> None:
    if args.mode not in {"dry-run", "apply"}:
        raise ValueError("mode must be dry-run or apply")
    run("git", "remote", "remove", "creator-signal-upstream", check=False)
    run("git", "remote", "add", "creator-signal-upstream", UPSTREAM)
    run("git", "fetch", "--no-tags", "creator-signal-upstream", "+refs/heads/*:refs/remotes/creator-signal-upstream/*")
    upstream_heads, upstream_tags = refs("creator-signal-upstream")
    origin_heads, origin_tags = refs("origin")
    selected_heads = {
        name: sha for name, sha in upstream_heads.items()
        if name == "forgejo" or RELEASE_BRANCH.fullmatch(name)
    }
    selected_tags = {name: sha for name, sha in upstream_tags.items() if SEMVER.fullmatch(name)}
    mismatches = [
        {"tag": name, "origin": origin_tags[name], "upstream": sha}
        for name, sha in selected_tags.items()
        if name in origin_tags and origin_tags[name] != sha
    ]
    if mismatches:
        raise RuntimeError(f"immutable tag mismatch: {json.dumps(mismatches, sort_keys=True)}")

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
                raise RuntimeError(f"non-fast-forward upstream branch refused: {name} {current} -> {sha}")
        branch_updates.append({"name": name, "from": current, "to": sha})
    new_tags = [
        {"name": name, "sha": sha}
        for name, sha in sorted(selected_tags.items()) if name not in origin_tags
    ]
    report = {
        "schema": "creator-signal.forgejo-upstream-sync/v1",
        "mode": args.mode,
        "upstream": UPSTREAM,
        "branchUpdates": branch_updates,
        "newTags": new_tags,
        "tagMismatches": [],
    }
    Path(args.report).write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    if args.mode == "apply":
        for item in branch_updates:
            run("git", "push", "origin", f"{item['to']}:refs/heads/{item['name']}")
        for item in new_tags:
            run("git", "push", "origin", f"{item['sha']}:refs/tags/{item['name']}")


def plan_tag(args: argparse.Namespace) -> None:
    match = SEMVER.fullmatch(args.tag)
    if not match:
        raise ValueError("tag must be a stable or prerelease semantic version")
    source_sha = run("git", "rev-parse", f"{args.tag}^{{commit}}")
    _, upstream_tags = refs(UPSTREAM)
    upstream_sha = upstream_tags.get(args.tag)
    if not upstream_sha or upstream_sha != source_sha:
        raise RuntimeError(f"tag source mismatch: mirror={source_sha} upstream={upstream_sha}")
    release = subprocess.run(
        ["gh", "release", "view", args.tag, "--repo", args.repository, "--json", "isDraft,url"],
        text=True, capture_output=True, check=False,
    )
    release_exists = release.returncode == 0
    plan = {
        "schema": "creator-signal.forgejo-release-plan/v1",
        "upstream": UPSTREAM,
        "tag": args.tag,
        "version": args.tag[1:],
        "sourceSha": source_sha,
        "prerelease": bool(match.group(4)),
        "releaseExists": release_exists,
    }
    Path(args.report).write_text(json.dumps(plan, indent=2) + "\n", encoding="utf-8")
    for key, value in {
        "source_sha": source_sha,
        "version": args.tag[1:],
        "prerelease": str(plan["prerelease"]).lower(),
        "release_exists": str(release_exists).lower(),
    }.items():
        output(key, value)


def verify_platforms(args: argparse.Namespace) -> None:
    data = json.loads(Path(args.file).read_text(encoding="utf-8"))
    actual = {
        f"{item.get('platform', {}).get('os')}/{item.get('platform', {}).get('architecture')}"
        for item in data.get("manifests", [])
        if item.get("platform", {}).get("os") != "unknown"
    }
    if actual != PLATFORMS:
        raise RuntimeError(f"manifest platforms differ: expected={sorted(PLATFORMS)} actual={sorted(actual)}")


def write_record(args: argparse.Namespace) -> None:
    for value in (args.rootful_digest, args.rootless_digest):
        if not DIGEST.fullmatch(value):
            raise ValueError(f"invalid digest: {value}")
    if not re.fullmatch(r"[0-9a-f]{40}", args.source_sha):
        raise ValueError("source SHA must be 40 lowercase hex characters")
    evidence = {}
    for path in sorted(Path(args.evidence).glob("*")):
        if path.is_file():
            evidence[path.name] = "sha256:" + hashlib.sha256(path.read_bytes()).hexdigest()
    record = {
        "schema": "creator-signal.forgejo-container-release/v1",
        "upstream": UPSTREAM,
        "tag": args.tag,
        "sourceSha": args.source_sha,
        "workflowUrl": args.workflow_url,
        "images": {
            "rootful": {"reference": f"ghcr.io/creator-signal/forgejo:{args.tag[1:]}", "digest": args.rootful_digest, "platforms": sorted(PLATFORMS)},
            "rootless": {"reference": f"ghcr.io/creator-signal/forgejo:{args.tag[1:]}-rootless", "digest": args.rootless_digest, "platforms": sorted(PLATFORMS)},
        },
        "evidence": evidence,
    }
    Path(args.output).write_text(json.dumps(record, indent=2) + "\n", encoding="utf-8")


def verify_record(args: argparse.Namespace) -> None:
    record = json.loads(Path(args.file).read_text(encoding="utf-8"))
    if record.get("schema") != "creator-signal.forgejo-container-release/v1":
        raise ValueError("release record schema mismatch")
    if record.get("tag") != args.tag or record.get("sourceSha") != args.source_sha:
        raise ValueError("release record source identity mismatch")
    for variant in ("rootful", "rootless"):
        image = record.get("images", {}).get(variant, {})
        if not DIGEST.fullmatch(image.get("digest", "")) or set(image.get("platforms", [])) != PLATFORMS:
            raise ValueError(f"invalid {variant} release record")


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
    command = commands.add_parser("verify-platforms")
    command.add_argument("--file", required=True)
    command.set_defaults(handler=verify_platforms)
    command = commands.add_parser("write-record")
    for name in ("tag", "source-sha", "rootful-digest", "rootless-digest", "workflow-url", "evidence", "output"):
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
