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

## Atomic Actions pair extension (Issue #7)

The cs.2 recipes read the original Go and Alpine immutable indexes through their
Docker Official Images `docker.io/library` repositories. Actual bounded reads
proved the identical index bytes and hashes after the inherited Forgejo mirror
returned 404: Go is 10293 bytes at `b17af760035fc2f338eed92d448a6c67f2d45438844fc6c60678fa5f99e44b57`,
and Alpine is 9218 bytes at `fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40`.
Versions, digests and the available xx reference remain unchanged; cs.1 is frozen.
This Source transport repair does not establish native image qualification.

The `v16.0.3-cs.2` extension uses this same fork release lane. The historical
upstream mirror, `v16.0.3`, and `v16.0.3-cs.1` remain immutable. Its runtime
owner uses one Issue #7 branch rooted at the exact `v16.0.3` upstream commit,
retaining the reviewed cs.1 versions, digests, and security behavior with the
documented registry reference repair. The automation owner retains that
runtime history through the existing no-content merge pattern. One PR into
`creator-signal/automation` reviews both histories. Policy pins the complete
ordered linear runtime commit list, source tree, all changed paths, and the raw
upstream-to-head patch hash. The first runtime commit has only the upstream
parent; every correction has only its preceding reviewed runtime parent.
Commit objects are admitted by size before reading, with at most 32 commits,
64 KiB per object, and 15 seconds per read. No hidden commits or merge parents
are admitted. The native checkout verification is a local object check;
the existing parent controller separately authenticates upstream/tag authority.

The repository-admin, Actions-write operation is
`POST /repos/{owner}/{repo}/actions/secret-pair-operations/{operationId}`.
Its closed `creator-signal.actions-secret-pair-operation/v1` body contains
`purpose: vm-artifact-publisher`, the matching lowercase 64-hex operation ID,
the original lowercase 64-hex `ownershipId`, `transactionId`, and `nonce` in
`binding`, and exactly the two fixed entries in `secrets`:
`ZOT_VM_ARTIFACT_USERNAME: vm-image-publisher` and the original intended
`ZOT_VM_ARTIFACT_PASSWORD` matching `cs-vm-artifact-` plus 43 base64url characters.
The provider derives the original numeric repository ID from authentication.
It does not accept a repository ID, caller-selected namespace, secret digest,
or additional body field. Duplicate JSON fields are malformed.

The database transaction creates both encrypted secret rows and their retained
operation together. A fresh operation returns 201. An identical replay returns
200 only after rereading the actual original rows and intended material; it
does not update them. Both replies have the exact closed result
`creator-signal.actions-secret-pair-operation-result/v1` with `operationId`,
`binding`, `state: Applied`, and the boolean `replayed`. Foreign names,
mismatched binding/material, revoked operations, or inconsistent retained rows
return generic 409 without private values or secret-derived hashes. Malformed
requests return 400. Actions disabled denies the operation and task projection
before database effects. HTTP success is provider evidence only: Normal remains
`ActionsPending` until independent qualified whole-pair Factory readback.

Central model guards protect both existing names and insert/rename destinations
across API and UI paths. Legacy mutations of either reserved name conflict;
they cannot create, update, rotate, or delete the managed pair. Unrelated secrets
retain their existing behavior. Task materialization admits the pair together;
workflow-call transport uses `secrets: inherit`. Explicit reserved destinations,
reserved source AST references, and dynamic/unsupported access capable of
reading the pair deny before exposure. Repository deletion revokes operations
in the same transaction and retains terminal tombstones under the original
numeric repository ID. Recreating the same slug cannot resurrect an operation.

The existing four native rootful/rootless AMD64/ARM64 cells own runtime model,
service, migration, and task-projection tests, generated Swagger byte comparison,
and real image API acceptance. `scripts/creator-signal/qualify_secret_pair.py`
is a helper of those cells and the existing published-platform pull-back and
independent verification jobs. It takes only the exact observed image ID,
Source SHA, native run ID, variant, and architecture. It creates synthetic
SQLite repositories in labeled isolated containers on owned internal networks,
publishes only random loopback ports, and inherits no release token. It checks atomic creation,
concurrent no-write replay, conflicts, closed JSON, legacy destination denial,
unrelated secrets, restart, repository tombstones, and Actions-disabled denial.
The runtime tests separately cover fault atomicity and actual task projection.
Each of the four cells also closes, copies, and reopens actual file SQLite
fixtures before creation, after committed creation, and after terminal
revocation. Private assertions compare exact database bytes, encrypted material,
and row identities, then exercise the real task projection with Actions disabled.
These fixtures prove inactive restore only; no database bytes or private hashes
are uploaded, and original-provider/history recovery and re-enabling stay unproved.
The helper has a 600-second monotonic budget, CLI command bounds of 30 seconds
and 64 KiB, and HTTP read bounds of 15 seconds and 64 KiB. It emits bounded
identity/check metadata without private material. Failed observations emit only
the denial event and a finite code for an exact Source-owned static guard;
parser errors, unknown exceptions and altered messages emit `UnknownDenied`.
Exception text, API responses and private input are never forwarded. A denial
still exits unsuccessfully and provides no acceptance. Cleanup rechecks the original
container ID, immutable image, labels, isolation settings, and original anonymous
volume inventory before removing only its container and anonymous volumes. The
original internal network is removed only after identity and no remaining
consumers are proved; unknown custody is retained and fails qualification.
No new job, workflow, provider dispatcher, or publication credential is added.

A restored database is inactive and quarantined with Actions disabled. Its
operation rows and tombstones alone cannot establish antirollback freshness.
Reactivation requires the separately governed recovery boundary to fence the
former writer and independently join original provider operation custody and
Normal signed history. Missing proof retains quarantine; it never remints an
operation, replays a blind PUT, restores an old binary capable of bypassing the
guards, or claims automatic recovery. The Sales Pulse recovery owner supplies
that admission; the fork does not invent an external ledger or rotation route.

The admitted disposable compiler container creates a fixed `forgejo` UID/GID
1000; existing account, group, or private-home collisions deny. It hands only
its image checkout to that identity and executes the unchanged tests through
`su` after checking the actual UID and GID. Its new private home and Go caches
are confined to the container; the original image module cache stays untouched.
The existing 20-minute timeout includes setup and tests, and original custody
and cleanup checks remain required. Forgejo's production root-user guard stays
enabled; no unsafe root configuration or additional privilege is used.

Fork PR/native qualification, signed immutable release publication, Sales Pulse
Source adoption of its exact digest, and live activation are separate gates.
This Source extension does not itself publish a release or deploy a provider.
