# Creator Signal Forgejo mirror and container release

This runbook governs the GitHub cross-forge mirror of
`https://codeberg.org/forgejo/forgejo.git`. The protected
`creator-signal/automation` branch owns Creator Signal automation. Mirrored
upstream branches and tags remain distinct from Creator Signal downstream
source tags.

No workflow deploys Sales Pulse, invokes Coolify, changes an environment branch,
or publishes to Docker Hub, S3, or a legacy Gitea image name. The authorized
output is an immutable GHCR image pair and a durable GitHub Release record.

## Immutable identities

The original `v16.0.3` Release and its AMD64-only rootful and rootless manifests
remain unchanged. They are retained as historical evidence and are not relabelled
as multi-architecture or downstream source.

`creator-signal/forgejo-release-policy.json` reserves `v16.0.3-cs.1` for the
explicit Creator Signal source commit. That commit has the exact upstream
`v16.0.3` source as its sole parent and commits only these build-input changes:

- pin the `xx`, Go builder, and Alpine runtime OCI indexes by digest in both
  Dockerfiles; and
- apply the bounded `apk upgrade --no-cache` runtime security refresh in both
  Dockerfiles.

The policy binds the upstream tag and SHA, downstream source and tree SHAs,
changed-path inventory, raw Git patch SHA256, and every resolved base-image
digest. The release controller reproduces these values before qualification,
tagging, publication, or existing-release verification. The `-cs.N` namespace
is reserved for Creator Signal; synchronization reports it as
`reserved-unpublished` or `immutable-downstream` and rejects upstream collision
or tag movement.

## Upstream refresh

`upstream-sync.yml` runs daily at 04:43 UTC and supports manual `dry-run` and
`apply` modes. It never deletes or force-pushes a ref. It refuses
non-fast-forward branch updates, mismatched upstream tags, moved downstream
tags, and a downstream name that appears upstream.

```sh
gh workflow run upstream-sync.yml --ref creator-signal/automation -f mode=dry-run
gh workflow run upstream-sync.yml --ref creator-signal/automation -f mode=apply
```

Inspect the retained JSON dry-run before apply. A mismatch is an integrity
incident; preserve both identities and repair the reviewed policy or source.

## Native qualification and publication

For both rootful and rootless variants, qualification runs independently on
GitHub-hosted native `linux/amd64` and native `linux/arm64` hosts. Each cell:

1. verifies the exact committed source, sole upstream parent, patch SHA256, and
   digest-pinned Dockerfile inputs;
2. builds for its native architecture without publication or QEMU;
3. starts isolated SQLite state, requires `/api/healthz`, and verifies
   `v16.0.3-cs.1` through `/api/v1/version`;
4. fails on fixed HIGH or CRITICAL OS/library vulnerabilities; and
5. generates a platform SPDX JSON SBOM.

Only after all four cells succeed may the workflow create the immutable
downstream source tag. Publication uses BuildKit to create rootful and rootless
AMD64+ARM64 candidates with registry-native SBOM and maximum provenance. QEMU is
only a publication mechanism; it is never accepted as the ARM64 qualification.

Before final tags or the GitHub Release are created, matching native hosts pull
each published platform child by digest, verify source/version labels, repeat
startup/readiness/version, and scan the exact published bytes. Finalization then:

- verifies each manifest contains exactly AMD64 and ARM64 plus only allowed
  attestation descriptors;
- generates manifest-level SPDX SBOMs;
- refuses an existing version tag unless it already selects the same digest;
- promotes only the verified rootful and rootless digests;
- keylessly signs both immutable indexes and all four exact platform manifests;
- publishes and verifies GitHub provenance and SPDX attestations for every one
  of those six digests; and
- publishes a Release record containing both manifest digests and all four
  child-platform digests, source identity, resolved bases, and evidence hashes.

Because `v16.0.3-cs.1` is a downstream SemVer prerelease, it does not move the
historical `latest` aliases.

```sh
gh workflow run forgejo-release.yml --ref creator-signal/automation -f tag=v16.0.3-cs.1
gh workflow run forgejo-release-verification.yml --ref creator-signal/automation -f tag=v16.0.3-cs.1
```

## Independent verification and recovery

The independent workflow downloads the Release record afresh. Native AMD64 and
ARM64 jobs verify exact manifest platforms, Cosign identity, GitHub provenance
and SPDX attestations, record-to-child digest binding, source/version labels,
startup/readiness/version, and the fixed HIGH/CRITICAL scan policy.

A completed GitHub Release is immutable. Rerunning its tag selects independent
verification and never rebuilds or overwrites it. The release workflow retains
the existing all-or-none recovery inputs for a run interrupted after candidates
were published. Supply the prior run ID and both exact manifest digests; the
workflow revalidates their platform children natively before finalization.

```sh
gh workflow run forgejo-release.yml --ref creator-signal/automation \
  -f tag=v16.0.3-cs.1 \
  -f resume_run_id=<failed-run-id> \
  -f resume_rootful_digest=sha256:<rootful-digest> \
  -f resume_rootless_digest=sha256:<rootless-digest>
```

Never delete or replace an existing tag, manifest, attestation, or Release to
make a recovery pass. Consumers select the exact manifest digest from
`release-record.json`; no repository Release authorizes an environment deploy.
