from __future__ import annotations

import argparse
import json
import tempfile
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


if __name__ == "__main__":
    unittest.main()
