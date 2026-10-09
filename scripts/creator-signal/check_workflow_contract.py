#!/usr/bin/env python3
"""Fail closed when Creator Signal Forgejo workflow contracts drift."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = ROOT / ".github" / "workflows"
POLICY = json.loads(
    (ROOT / "creator-signal" / "forgejo-release-policy.json").read_text(encoding="utf-8")
)
SHA_USE = re.compile(r"^\s*-?\s*uses:\s*[^\s@]+@[0-9a-f]{40}(?:\s+#.*)?$", re.MULTILINE)


def require(condition: bool, message: str, errors: list[str]) -> None:
    if not condition:
        errors.append(message)


def main() -> int:
    errors: list[str] = []
    expected = {
        "automation-validation.yml",
        "upstream-sync.yml",
        "forgejo-qualification.yml",
        "forgejo-publish-variant.yml",
        "forgejo-release.yml",
        "forgejo-release-verification.yml",
    }
    actual = {path.name for path in WORKFLOWS.glob("*.yml")}
    require(actual == expected, f"workflow inventory drift: {sorted(actual)}", errors)
    sources = {
        name: (WORKFLOWS / name).read_text(encoding="utf-8")
        for name in expected
        if (WORKFLOWS / name).exists()
    }
    combined = "\n".join(sources.values())
    for name, source in sources.items():
        for line in re.findall(r"^\s*-?\s*uses:.*$", source, re.MULTILINE):
            reference = line.split("uses:", 1)[1].strip()
            require(
                reference.startswith("./.github/workflows/") or bool(SHA_USE.fullmatch(line)),
                f"{name}: mutable action reference: {line.strip()}",
                errors,
            )
        require(
            "permissions:\n  contents: read" in source,
            f"{name}: default permissions are not read-only",
            errors,
        )
        require("timeout-minutes:" in source, f"{name}: bounded timeout missing", errors)
        require("pull_request_target:" not in source, f"{name}: pull_request_target is prohibited", errors)

    configured = POLICY.get("downstreamReleases", {}).get("v16.0.3-cs.1", {})
    require(
        configured.get("baseTag") == "v16.0.3"
        and configured.get("baseSourceSha") == "eccddb2d17c93b42b2c8995725e03e549ac9ec0c"
        and configured.get("sourceCommitSha") == "ea71be6eb248b928ee5d446ed441bf78d8dd42ee"
        and configured.get("sourceTreeSha") == "a10677d2b09db7484cd48af7a3ea21eb0b322d1a"
        and configured.get("sourcePatchSha256")
        == "2a1452f9cb0d69b63abc3327301928401d395a9af7a74e3e9b95920154ee8c4f",
        "downstream source/base/tree/patch identity drifted",
        errors,
    )
    require(
        configured.get("platforms") == ["linux/amd64", "linux/arm64"],
        "downstream platform policy must require amd64 and arm64",
        errors,
    )
    require(
        set(configured.get("baseImages", {})) == {"xx", "golang", "alpine"}
        and all(
            re.fullmatch(r"sha256:[0-9a-f]{64}", item.get("digest", ""))
            for item in configured.get("baseImages", {}).values()
        ),
        "all three downstream base-image indexes must be digest pinned",
        errors,
    )

    sync = sources.get("upstream-sync.yml", "")
    control = (ROOT / "scripts" / "creator-signal" / "release_control.py").read_text(encoding="utf-8")
    require("cron: '43 4 * * *'" in sync, "non-top-of-hour schedule missing", errors)
    require("--mode dry-run" in sync and "--mode apply" in sync, "sync dry-run/apply boundary missing", errors)
    require("reserved-unpublished" in control and "immutable-downstream" in control, "sync does not distinguish downstream tags", errors)
    require("sourcePatchSha256" in control and "baseImages" in control, "release control does not bind downstream patch/base inputs", errors)
    require("publish-downstream-tag" in control, "downstream source tag publication control missing", errors)

    validation = sources.get("automation-validation.yml", "")
    runtime = POLICY.get("downstreamReleases", {}).get("v16.0.3-cs.2", {})
    require(runtime.get("sourceFormat") == "creator-signal.forgejo-secret-pair-source/v1"
            and runtime.get("baseSourceSha") == configured.get("baseSourceSha")
            and runtime.get("sourceCommitChain", [])[-1:] == [runtime.get("sourceCommitSha")]
            and {name: item.get("digest") for name, item in runtime.get("baseImages", {}).items()}
                == {name: item.get("digest") for name, item in configured.get("baseImages", {}).items()}
            and {name: item.get("reference") for name, item in runtime.get("baseImages", {}).items()}
                == {"xx": "data.forgejo.org/oci/xx", "golang": "docker.io/library/golang:1.26-alpine3.23", "alpine": "docker.io/library/alpine:3.23"}
            and runtime.get("platforms") == configured.get("platforms"),
            "atomic pair runtime source closure drifted", errors)
    qualification = sources.get("forgejo-qualification.yml", "")
    publish = sources.get("forgejo-publish-variant.yml", "")
    release = sources.get("forgejo-release.yml", "")
    verify = sources.get("forgejo-release-verification.yml", "")

    require(
        f"source_ref: {runtime.get('sourceCommitSha')}" in validation
        and f"source_patch_sha256: {runtime.get('sourcePatchSha256')}" in validation
        and "release_version: 16.0.3-cs.2" in validation,
        "automation validation is not bound to the committed downstream source",
        errors,
    )
    for token in ("linux/amd64", "linux/arm64", "ubuntu-24.04-arm", "Dockerfile", "Dockerfile.rootless"):
        require(token in qualification, f"qualification missing {token}", errors)
    require("setup-qemu" not in qualification.lower(), "native qualification must not use QEMU", errors)
    require(qualification.count("ubuntu-24.04-arm") == 2, "both ARM64 variants need real hosted ARM64", errors)
    require("sha256sum --check --strict" in qualification and "@sha256:" in qualification, "qualification source/base identity is not fail closed", errors)
    require("git describe --tags --exact-match HEAD" in qualification, "qualification does not provide exact local semantic-tag context", errors)
    require(qualification.index("Run isolated native startup") < qualification.index("Enforce fixed HIGH and CRITICAL"), "startup must precede native scan", errors)
    require("severity: CRITICAL,HIGH" in qualification and 'exit-code: "1"' in qualification, "strict native vulnerability gate missing", errors)
    require("format: spdx-json" in qualification, "native SPDX SBOM missing", errors)
    require("creator-signal.forgejo-native-qualification-provenance/v1" in qualification, "native build provenance missing", errors)
    for token in ("verify-source-checkout", "make swagger-check", "make swagger-validate", "./models/secret ./services/secrets ./services/actions", "./models/forgejo_migrations", "test-sqlite#SecretPair", "qualify_secret_pair.py", "secret-pair-tests-"):
        require(token in qualification, f"atomic pair native qualification missing {token}", errors)
    for token in ("addgroup -S -g 1000 forgejo", "adduser -D -u 1000 -G forgejo", "chown -R 1000:1000 /go/src/forgejo.org", "exec su -s /bin/sh forgejo -c", 'test "$(id -u)" = 1000', 'test "$(id -g)" = 1000', 'GOMODCACHE="$HOME/go/pkg/mod"', 'GOCACHE="$HOME/.cache/go-build"'):
        require(token in qualification, f"atomic compiler unprivileged execution missing {token}", errors)
    # Temporary cs.2 Source materialization is deliberately not a qualification cell.
    capture = qualification.split("      - name: Capture bounded grpc dependency graph without qualification", 1)
    require(len(capture) == 2, "fixed dependency capture missing", errors)
    if len(capture) == 2:
        capture = capture[1].split("      - name: Run isolated native startup", 1)[0]
        for token in ('test "$SOURCE_SHA" = 2d1f5bea847d1c9d2336e1f22e5b90ada5f8ada9',
                      "go get google.golang.org/grpc@v1.83.2", "go mod tidy",
                      "GOSUMDB=sum.golang.org GOTOOLCHAIN=local", "unset GOPRIVATE GONOPROXY GONOSUMDB GOFLAGS",
                      "creator-signal.forgejo-dependency-capture/v1", "qualified:false",
                      "DependencyCaptureRetainedUnqualified", 'cmp runtime-source/go.mod "$capture_dir/go.mod"',
                      'cmp runtime-source/go.sum "$capture_dir/go.sum"',
                      'if test -f "$HOME/dependency-capture/go.mod"; then exit 0; fi'):
            require(token in capture, f"dependency custody-only gate missing {token}", errors)
        require(capture.index("test \"$result\" = 0") < capture.index("docker cp"), "dependency copy precedes stopped-success proof", errors)
        require(capture.index("go mod tidy") < capture.index("exit 0") < capture.index("make swagger-check"), "capture could claim tests without running them", errors)
        require(capture.index("DependencyCaptureRetainedUnqualified") < capture.index("cmp runtime-source/go.mod"), "capture comparison precedes retained packet", errors)
        require(capture.rstrip().endswith("exit 1"), "capture must fail even if comparison matches", errors)
        require("NativeTestsCompleted" not in capture, "capture cannot emit native test acceptance", errors)

    require("docker/setup-qemu-action@" in publish, "multi-architecture publication builder missing", errors)
    require("platforms: linux/amd64,linux/arm64" in publish, "publication is not exactly amd64 plus arm64", errors)
    require("Apply bounded Alpine security refresh" not in publish, "publication still mutates source after checkout", errors)
    require("git describe --tags --exact-match HEAD" in publish, "publication does not preserve exact semantic-tag build context", errors)
    require("grep -Fc '@sha256:'" in publish and "RUN apk upgrade --no-cache" in publish, "publication does not validate committed source inputs", errors)
    require("provenance: mode=max" in publish and "sbom: true" in publish, "registry provenance/SBOM missing", errors)

    for token in (
        "default: v16.0.3-cs.1",
        "publish-source-tag:",
        "publish-downstream-tag",
        "ubuntu-24.04-arm",
        "verify-published-platforms:",
        "published-scan-rootful",
        "published-scan-rootless",
        "platform.architecture == $arch",
        "cosign sign --yes",
        "gh attestation verify",
        "release-record.json",
        "gh release create",
    ):
        require(token in release, f"release workflow missing {token}", errors)
    require("needs: [plan, qualify, publish-source-tag]" in release, "publication is not gated on source-tag materialization", errors)
    require("--rootful-manifest evidence/manifest-rootful.json" in release, "release record lacks platform child digests", errors)
    require("--clobber" not in release, "immutable Release assets may not be overwritten", errors)
    require("latest-rootless" in release and "prerelease != 'true'" in release, "stable aliases are not protected from downstream prereleases", errors)

    for token in ("ubuntu-24.04", "ubuntu-24.04-arm", "platformDigests", "cosign verify", "gh attestation verify", "severity: CRITICAL,HIGH"):
        require(token in verify, f"independent verification missing {token}", errors)
    require("gh release download" in verify and "docker pull --platform" in verify, "independent exact-platform download missing", errors)

    for forbidden in (
        "docker.io/gitea/gitea",
        "gitea/gitea",
        "s3-sync-action",
        "DOCKERHUB_",
        "AWS_S3_BUCKET",
    ):
        require(forbidden not in combined, f"forbidden legacy destination present: {forbidden}", errors)
    if errors:
        print("\n".join(f"ERROR: {item}" for item in errors), file=sys.stderr)
        return 1
    print("Forgejo workflow contracts passed for explicit downstream source and native ARM64.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
