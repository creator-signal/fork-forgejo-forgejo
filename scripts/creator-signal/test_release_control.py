from __future__ import annotations

import argparse
import json
import tempfile
import unittest
from pathlib import Path

from scripts.creator_signal_import import release_control


class ReleaseControlTests(unittest.TestCase):
    def test_platform_verification_requires_exact_linux_pair(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "manifest.json"
            path.write_text(json.dumps({"manifests": [
                {"platform": {"os": "linux", "architecture": "amd64"}},
                {"platform": {"os": "linux", "architecture": "arm64"}},
                {"platform": {"os": "unknown", "architecture": "unknown"}},
            ]}), encoding="utf-8")
            release_control.verify_platforms(argparse.Namespace(file=str(path)))
            path.write_text(json.dumps({"manifests": [{"platform": {"os": "linux", "architecture": "amd64"}}]}), encoding="utf-8")
            with self.assertRaises(RuntimeError):
                release_control.verify_platforms(argparse.Namespace(file=str(path)))

    def test_release_record_round_trip_is_source_and_digest_bound(self):
        with tempfile.TemporaryDirectory() as directory:
            evidence = Path(directory) / "evidence"
            evidence.mkdir()
            (evidence / "scan.json").write_text("{}\n", encoding="utf-8")
            record = Path(directory) / "record.json"
            digest = "sha256:" + "a" * 64
            release_control.write_record(argparse.Namespace(
                tag="v16.0.3", source_sha="b" * 40, rootful_digest=digest,
                rootless_digest="sha256:" + "c" * 64, workflow_url="https://example.invalid/run",
                evidence=str(evidence), output=str(record),
            ))
            release_control.verify_record(argparse.Namespace(file=str(record), tag="v16.0.3", source_sha="b" * 40))
            with self.assertRaises(ValueError):
                release_control.verify_record(argparse.Namespace(file=str(record), tag="v16.0.4", source_sha="b" * 40))

    def test_semver_and_release_branch_patterns_fail_closed(self):
        self.assertIsNotNone(release_control.SEMVER.fullmatch("v16.0.3"))
        self.assertIsNotNone(release_control.SEMVER.fullmatch("v17.0.0-rc.1"))
        self.assertIsNone(release_control.SEMVER.fullmatch("latest"))
        self.assertIsNotNone(release_control.RELEASE_BRANCH.fullmatch("v16/forgejo"))
        self.assertIsNotNone(release_control.RELEASE_BRANCH.fullmatch("v16.0/forgejo"))
        self.assertIsNone(release_control.RELEASE_BRANCH.fullmatch("main"))


if __name__ == "__main__":
    unittest.main()
