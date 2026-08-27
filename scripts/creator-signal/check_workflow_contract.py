#!/usr/bin/env python3
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
WORKFLOWS = ROOT / ".github" / "workflows"
SHA_USE = re.compile(r"^\s*-?\s*uses:\s*[^\s@]+@[0-9a-f]{40}(?:\s+#.*)?$", re.MULTILINE)


def require(condition: bool, message: str, errors: list[str]) -> None:
    if not condition:
        errors.append(message)


def main() -> int:
    errors: list[str] = []
    expected = {
        "automation-validation.yml", "upstream-sync.yml", "forgejo-qualification.yml",
        "forgejo-publish-variant.yml", "forgejo-release.yml", "forgejo-release-verification.yml",
    }
    actual = {path.name for path in WORKFLOWS.glob("*.yml")}
    require(actual == expected, f"workflow inventory drift: {sorted(actual)}", errors)
    sources = {name: (WORKFLOWS / name).read_text(encoding="utf-8") for name in expected if (WORKFLOWS / name).exists()}
    combined = "\n".join(sources.values())
    for name, source in sources.items():
        for line in re.findall(r"^\s*-?\s*uses:.*$", source, re.MULTILINE):
            reference = line.split("uses:", 1)[1].strip()
            require(reference.startswith("./.github/workflows/") or bool(SHA_USE.fullmatch(line)), f"{name}: mutable action reference: {line.strip()}", errors)
        require("permissions:\n  contents: read" in source, f"{name}: default permissions are not read-only", errors)
        require("timeout-minutes:" in source, f"{name}: bounded timeout missing", errors)
    require("https://codeberg.org/forgejo/forgejo.git" in (ROOT / "scripts/creator-signal/release_control.py").read_text(), "authoritative upstream missing", errors)
    require("cron: '43 4 * * *'" in sources.get("upstream-sync.yml", ""), "non-top-of-hour schedule missing", errors)
    require("--mode dry-run" in sources.get("upstream-sync.yml", "") and "--mode apply" in sources.get("upstream-sync.yml", ""), "sync dry-run/apply boundary missing", errors)
    release = sources.get("forgejo-release.yml", "")
    qualify = sources.get("forgejo-qualification.yml", "")
    publish = sources.get("forgejo-publish-variant.yml", "")
    verify = sources.get("forgejo-release-verification.yml", "")
    for token in ("linux/amd64", "Dockerfile.rootless", "Dockerfile"):
        require(token in qualify, f"qualification missing {token}", errors)
    require("linux/arm64" not in combined and "ubuntu-24.04-arm" not in combined, "unsupported ARM64 release path present", errors)
    require(qualify.index("Run isolated native startup") < qualify.index("Enforce HIGH and CRITICAL"), "startup must precede scan", errors)
    require("severity: CRITICAL,HIGH" in qualify and 'exit-code: "1"' in qualify, "strict vulnerability gate missing", errors)
    require("format: spdx-json" in qualify, "SPDX SBOM missing", errors)
    require("source_ref: v16.0.3" in sources.get("automation-validation.yml", ""), "initial qualification must use the exact release tag", errors)
    require("GOPROXY=https://proxy.golang.org|direct" in qualify, "qualification Go proxy failover missing", errors)
    require("GOPROXY=https://proxy.golang.org|direct" in publish, "publication Go proxy failover missing", errors)
    require("platforms: linux/amd64" in publish, "publication must be AMD64-only", errors)
    require("/api/v1/version" in qualify and "startswith($version + \"+\")" in qualify and "jq -e '.status == \"pass\"'" in qualify, "native API version/readiness smoke controls missing", errors)
    require("/api/v1/version" in release and "/api/v1/version" in verify, "published-image version verification missing", errors)
    require("needs: [plan, qualify]" in release, "publication is not gated on qualification", errors)
    require("provenance: mode=max" in publish and "sbom: true" in publish, "registry-native provenance/SBOM missing", errors)
    require("verify-platforms" in release and "Pull back and smoke exact" in release, "manifest or pull-back verification missing", errors)
    require("cosign sign --yes" in release and "gh attestation verify" in release, "signature/attestation controls missing", errors)
    require("release-record.json" in release and "gh release create" in release, "durable GitHub Release record missing", errors)
    require("release_exists == 'true'" in release and "forgejo-release-verification.yml" in release, "idempotent existing-release path missing", errors)
    require("GH_REPO: ${{ github.repository }}" in release, "GitHub CLI repository binding missing", errors)
    require("ghcr.io/creator-signal/forgejo" in release and "latest-rootless" in release, "governed GHCR aliases missing", errors)
    require("Independent published Forgejo verification" in verify and "docker pull --platform linux/amd64" in verify, "independent pull-back workflow missing", errors)
    for forbidden in ("docker.io/gitea/gitea", "gitea/gitea", "s3-sync-action", "DOCKERHUB_", "AWS_S3_BUCKET", "--clobber"):
        require(forbidden not in combined, f"forbidden legacy or overwrite path present: {forbidden}", errors)
    if errors:
        print("\n".join(f"ERROR: {item}" for item in errors), file=sys.stderr)
        return 1
    print("Forgejo workflow contracts passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
