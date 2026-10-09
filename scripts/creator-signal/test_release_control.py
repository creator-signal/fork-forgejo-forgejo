from __future__ import annotations

import argparse
import json
import tempfile
from types import SimpleNamespace
import unittest
from pathlib import Path
from unittest.mock import patch

from scripts.creator_signal_import import release_control


def manifest(*platforms: str) -> dict[str, object]:
    manifests = []
    for index, platform in enumerate(platforms, 1):
        os_name, architecture = platform.split("/", 1)
        manifests.append(
            {
                "digest": "sha256:" + str(index) * 64,
                "platform": {"os": os_name, "architecture": architecture},
            }
        )
    manifests.append(
        {
            "digest": "sha256:" + "f" * 64,
            "platform": {"os": "unknown", "architecture": "unknown"},
        }
    )
    return {"manifests": manifests}


class ReleaseControlTests(unittest.TestCase):
    def test_runtime_source_binds_full_diff_without_treating_go_as_dockerfile(self):
        configured = dict(release_control.load_policy()["downstreamReleases"]["v16.0.3-cs.1"])
        configured["sourceFormat"] = "creator-signal.forgejo-secret-pair-source/v1"
        configured["changedPaths"] = ["Dockerfile", "Dockerfile.rootless", "models/secret/pair_operation.go"]
        recipes = configured["securityRefresh"] + "\n" + "\n".join(
            f"FROM {image['reference']}@{image['digest']}" for image in configured["baseImages"].values()
        )

        def observed_git(*args):
            if args[:3] == ("git", "rev-list", "--parents"):
                return f"{configured['sourceCommitSha']} {configured['baseSourceSha']}"
            if args[:2] == ("git", "rev-parse"):
                return configured["sourceTreeSha"]
            if args[:3] == ("git", "show", "-s"):
                return configured["sourceCommitSubject"]
            if args[:2] == ("git", "show"):
                self.assertIn(args[2].split(":", 1)[1], ["Dockerfile", "Dockerfile.rootless"])
                return recipes
            self.fail(f"unexpected Git observation: {args}")

        refs = {configured["baseTag"]: configured["baseSourceSha"]}
        with patch.object(release_control, "exact_commit_exists", return_value=True), \
             patch.object(release_control, "validate_runtime_commit_chain"), \
             patch.object(release_control, "run", side_effect=observed_git), \
             patch.object(release_control, "source_patch_identity", return_value=(configured["sourcePatchSha256"], configured["changedPaths"])) as diff:
            release_control.validate_downstream_source("v16.0.3-cs.2", configured, refs, refs)
            diff.assert_called_once_with(configured["baseSourceSha"], configured["sourceCommitSha"], configured["changedPaths"])
            configured["changedPaths"] = ["Dockerfile", "models/secret/pair_operation.go"]
            diff.return_value = (configured["sourcePatchSha256"], configured["changedPaths"])
            with self.assertRaisesRegex(release_control.ControlError, "both reviewed image recipes"):
                release_control.validate_downstream_source("v16.0.3-cs.2", configured, refs, refs)
            configured["sourceFormat"] = "unknown"
            with self.assertRaisesRegex(release_control.ControlError, "unsupported downstream source format"):
                release_control.validate_downstream_source("v16.0.3-cs.2", configured, refs, refs)

    def test_runtime_chain_requires_every_exact_ordered_sole_parent(self):
        base, first, last = "a" * 40, "b" * 40, "c" * 40
        configured = {"baseSourceSha": base, "sourceCommitSha": last, "sourceCommitChain": [first, last]}
        objects = {first: f"tree {'d' * 40}\nparent {base}\n\nfirst\n".encode(),
                   last: f"tree {'e' * 40}\nparent {first}\n\nlast\n".encode()}

        def observe(args, **kwargs):
            self.assertEqual(kwargs["timeout"], 15)
            value = objects[args[-1]]
            return SimpleNamespace(returncode=0, stdout=(str(len(value)).encode() + b"\n") if args[2] == "-s" else value)

        with patch.object(release_control.subprocess, "run", side_effect=observe):
            release_control.validate_runtime_commit_chain(configured)
            objects[last] = objects[last].replace(first.encode(), base.encode())
            with self.assertRaisesRegex(release_control.ControlError, "unreviewed parent"):
                release_control.validate_runtime_commit_chain(configured)
            objects[last] = objects[last].replace(b"\n\n", f"\nparent {first}\n\n".encode())
            with self.assertRaises(release_control.ControlError):
                release_control.validate_runtime_commit_chain(configured)
        for chain in [[], [first] * 33, [first, first], [last, first], [{}]]:
            configured["sourceCommitChain"] = chain
            with self.assertRaises(release_control.ControlError):
                release_control.validate_runtime_commit_chain(configured)

    def test_runtime_chain_denies_oversized_object_before_read(self):
        configured = {"baseSourceSha": "a" * 40, "sourceCommitSha": "b" * 40, "sourceCommitChain": ["b" * 40]}
        with patch.object(release_control.subprocess, "run", return_value=SimpleNamespace(returncode=0, stdout=b"65537\n")) as read:
            with self.assertRaisesRegex(release_control.ControlError, "custody bound"):
                release_control.validate_runtime_commit_chain(configured)
            self.assertEqual(read.call_count, 1)

    def test_platform_policy_preserves_legacy_and_requires_downstream_arm64(self):
        policy = release_control.load_policy()
        self.assertEqual(
            release_control.expected_platforms("v16.0.3", policy), {"linux/amd64"}
        )
        self.assertEqual(
            release_control.expected_platforms("v16.0.3-cs.1", policy),
            {"linux/amd64", "linux/arm64"},
        )

    def test_manifest_verification_is_exact_for_each_release_identity(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "manifest.json"
            path.write_text(json.dumps(manifest("linux/amd64", "linux/arm64")), encoding="utf-8")
            release_control.verify_platforms(
                argparse.Namespace(file=str(path), tag="v16.0.3-cs.1")
            )
            with self.assertRaises(release_control.ControlError):
                release_control.verify_platforms(
                    argparse.Namespace(file=str(path), tag="v16.0.3")
                )
            path.write_text(json.dumps(manifest("linux/amd64")), encoding="utf-8")
            release_control.verify_platforms(
                argparse.Namespace(file=str(path), tag="v16.0.3")
            )
            with self.assertRaises(release_control.ControlError):
                release_control.verify_platforms(
                    argparse.Namespace(file=str(path), tag="v16.0.3-cs.1")
                )

    def test_downstream_release_record_is_source_patch_base_and_platform_bound(self):
        policy = release_control.load_policy()
        configured = policy["downstreamReleases"]["v16.0.3-cs.1"]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            evidence = root / "evidence"
            evidence.mkdir()
            (evidence / "scan.json").write_text("{}\n", encoding="utf-8")
            rootful = root / "rootful.json"
            rootless = root / "rootless.json"
            rootful.write_text(json.dumps(manifest("linux/amd64", "linux/arm64")), encoding="utf-8")
            rootless.write_text(json.dumps(manifest("linux/amd64", "linux/arm64")), encoding="utf-8")
            record = root / "release-record.json"
            args = argparse.Namespace(
                tag="v16.0.3-cs.1",
                source_sha=configured["sourceCommitSha"],
                rootful_digest="sha256:" + "a" * 64,
                rootless_digest="sha256:" + "b" * 64,
                rootful_manifest=str(rootful),
                rootless_manifest=str(rootless),
                workflow_url="https://example.invalid/run",
                evidence=str(evidence),
                output=str(record),
            )
            with patch.object(release_control, "run", return_value=configured["sourceTreeSha"]):
                release_control.write_record(args)
            release_control.verify_record(
                argparse.Namespace(
                    file=str(record),
                    tag="v16.0.3-cs.1",
                    source_sha=configured["sourceCommitSha"],
                )
            )
            data = json.loads(record.read_text(encoding="utf-8"))
            self.assertEqual(
                data["sourceIdentity"]["sourcePatchSha256"], configured["sourcePatchSha256"]
            )
            self.assertEqual(data["sourceIdentity"]["baseImages"], configured["baseImages"])
            self.assertEqual(
                set(data["images"]["rootless"]["platformDigests"]),
                {"linux/amd64", "linux/arm64"},
            )

    def test_policy_reserves_cs_identity_and_exact_committed_source(self):
        policy = release_control.load_policy()
        configured = policy["downstreamReleases"]["v16.0.3-cs.1"]
        self.assertRegex("v16.0.3-cs.1", policy["downstreamTagPattern"])
        self.assertNotRegex("v16.0.3", policy["downstreamTagPattern"])
        self.assertEqual(configured["baseSourceSha"], "eccddb2d17c93b42b2c8995725e03e549ac9ec0c")
        self.assertEqual(configured["sourceCommitSha"], "ea71be6eb248b928ee5d446ed441bf78d8dd42ee")
        self.assertEqual(configured["sourceTreeSha"], "a10677d2b09db7484cd48af7a3ea21eb0b322d1a")
        self.assertEqual(
            configured["sourcePatchSha256"],
            "2a1452f9cb0d69b63abc3327301928401d395a9af7a74e3e9b95920154ee8c4f",
        )
        self.assertTrue(release_control.RELEASE_BRANCH.fullmatch("v16.0/forgejo"))
        self.assertFalse(release_control.RELEASE_BRANCH.fullmatch("main"))

    def test_atomic_native_evidence_requires_complete_bound_cells(self):
        checks = ["atomic-create", "concurrent-replay", "binding-conflict", "legacy-mutation-denial",
                  "closed-json", "restart-no-write-replay", "repository-tombstone",
                  "reserved-destination-denial", "unrelated-secret-compatibility", "actions-disabled-route-denial"]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            with self.assertRaises(release_control.ControlError):
                release_control.validate_pair_evidence(root, "a" * 40)
            for variant in ("rootful", "rootless"):
                for arch in ("amd64", "arm64"):
                    (root / f"secret-pair-tests-{variant}-{arch}.log").write_text("native test output\n")
                    value = {"schema": "creator-signal.forgejo-secret-pair-native/v1", "sourceRevision": "a" * 40,
                             "imageId": "sha256:" + "b" * 64, "variant": variant, "architecture": arch,
                             "checks": checks, "sourceQualified": False, "factoryQualified": False}
                    for prefix in ("secret-pair-", "published-secret-pair-"):
                        (root / f"{prefix}{variant}-{arch}.json").write_text(json.dumps(value))
            self.assertEqual(len(release_control.validate_pair_evidence(root, "a" * 40)), 12)
            path = root / "published-secret-pair-rootless-arm64.json"
            value = json.loads(path.read_text())
            for field, mutation in (("sourceRevision", "c" * 40), ("checks", checks[:-1]), ("factoryQualified", True), ("unknown", "extra")):
                changed = dict(value, **{field: mutation})
                path.write_text(json.dumps(changed))
                with self.assertRaises(release_control.ControlError):
                    release_control.validate_pair_evidence(root, "a" * 40)
            path.write_text(json.dumps(value))
            (root / "secret-pair-tests-rootless-arm64.log").write_text("")
            with self.assertRaises(release_control.ControlError):
                release_control.validate_pair_evidence(root, "a" * 40)


if __name__ == "__main__":
    unittest.main()
