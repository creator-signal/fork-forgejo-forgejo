# Creator Signal Forgejo mirror and container release

This runbook governs the GitHub cross-forge mirror of
`https://codeberg.org/forgejo/forgejo.git`. The protected default branch
`creator-signal/automation` owns Creator Signal automation. Mirrored `forgejo`,
maintained `v*/forgejo` branches and semantic-version tags remain distinct
upstream source refs.

No workflow deploys Sales Pulse, calls Coolify, changes an environment branch,
or publishes to Docker Hub, S3, or a legacy Gitea image name. The deliverable
is a verified OCI image release in GHCR and a durable GitHub Release record.

## Upstream refresh

`upstream-sync.yml` runs daily at 04:43 UTC and supports manual `dry-run` and
`apply` modes. Both modes fetch anonymously from the exact Codeberg URL. The
script selects `forgejo`, maintained `v*/forgejo` branches and semantic-version
tags. It never deletes a ref, never force-pushes, refuses non-fast-forward
branch updates, and fails with both SHAs when an existing tag differs.

```sh
gh workflow run upstream-sync.yml --ref creator-signal/automation -f mode=dry-run
gh workflow run upstream-sync.yml --ref creator-signal/automation -f mode=apply
```

Inspect the retained JSON dry-run before applying. A tag mismatch is an
integrity incident: do not delete or replace either tag. Confirm the Codeberg
ref, preserve the report, and repair through a reviewed automation change.

## Release and security policy

For a named upstream tag, `forgejo-release.yml`:

1. proves the mirrored tag commit equals the authoritative Codeberg tag;
2. natively builds rootful and rootless images on Linux amd64 without
   publication (ARM64 is intentionally outside Creator Signal's supported host
   contract);
3. starts each image with isolated SQLite state and requires `/api/healthz`;
4. fails on fixed HIGH or CRITICAL OS/library vulnerabilities;
5. generates an SPDX JSON SBOM for every variant/platform;
6. builds Linux amd64 OCI candidates with BuildKit SBOM and provenance;
7. pulls exact manifest digests back and repeats startup/readiness;
8. promotes only verified digests to immutable version tags;
9. keylessly signs and publishes provenance and SBOM attestations; and
10. creates one GitHub Release containing the source SHA, both manifest
    digests, SBOMs, scan summaries and workflow link.

The references are `ghcr.io/creator-signal/forgejo:<version>` and
`ghcr.io/creator-signal/forgejo:<version>-rootless`. Stable aliases move only
after every gate and the immutable Release succeed. Prereleases do not move
stable aliases. BuildKit provenance records resolved build inputs; consumers
must select the final manifest digest.

The scanner ignores unfixed findings because no remediation exists, but has no
blanket exception file. A fixed HIGH or CRITICAL finding blocks publication.
Repair the dependency instead of weakening the severity policy.

```sh
gh workflow run forgejo-release.yml --ref creator-signal/automation -f tag=v16.0.3
gh workflow run forgejo-release-verification.yml --ref creator-signal/automation -f tag=v16.0.3
```

## Idempotency and recovery

A completed GitHub Release is immutable. Rerunning its tag takes the independent
verification path and does not rebuild or replace it. Before version-tag
promotion, an existing tag is accepted only when its digest equals the verified
digest. The workflows never use `--clobber`.

If a run stops after a temporary candidate push, retain its evidence and rerun.
Candidate tags are not stable release selections. If a run stops after an
immutable version tag but before the GitHub Release, rerun: only the identical
digest can resume. Never delete a differing tag to make a run pass.

Keep the GHCR package private unless an owner explicitly approves public
visibility. Authenticated consumers pull by the digest in `release-record.json`.
Verify Cosign with the GitHub Actions issuer and exact release-workflow identity;
verify GitHub attestations with `gh attestation verify oci://<ref>@<digest>`.
