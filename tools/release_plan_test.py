from __future__ import annotations

import subprocess
import sys
import unittest
from pathlib import Path
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))

from release_plan import bump_level, merged_pull_requests, next_tag, plan, version_tuple


class ReleasePlanTests(unittest.TestCase):
    def test_default_patch(self) -> None:
        for message in ["fix: race setup", "chore(deps): bump Go", "update docs", ""]:
            self.assertEqual(bump_level([message], []), "patch")

    def test_feature_commit_and_label(self) -> None:
        self.assertEqual(bump_level(["feat(api): new field"], []), "minor")
        self.assertEqual(bump_level([], [{"title": "New field", "labels": [{"name": "feature"}]}]), "minor")
        self.assertEqual(bump_level([], [{"title": "feat: new field"}]), "minor")

    def test_breaking_wins(self) -> None:
        for message in ["feat!: new API", "fix(api)!: removed field", "fix: API\n\nBREAKING CHANGE: removed field", "chore: update\n\nBREAKING-CHANGE: removed field"]:
            self.assertEqual(bump_level(["feat: feature", message], []), "major")
        self.assertEqual(bump_level(["feat: feature"], [{"title": "change", "labels": [{"name": "breaking"}]}]), "major")

    def test_version_bumps_including_zero_major(self) -> None:
        self.assertEqual(next_tag("v1.2.3", "patch"), "v1.2.4")
        self.assertEqual(next_tag("v1.2.3", "minor"), "v1.3.0")
        self.assertEqual(next_tag("v0.2.3", "major"), "v1.0.0")

    def test_reject_invalid_tags(self) -> None:
        for tag in ["main", "v1.2", "v1.2.3-rc.1", "v01.2.3", "v1.2.3\nother"]:
            with self.assertRaises(ValueError):
                version_tuple(tag)

    @patch("release_plan.run")
    def test_prs_are_paginated_deduplicated_and_merged(self, command) -> None:
        command.return_value = '[[{"number":1,"merged_at":"today","base":{"ref":"main"}}, {"number":2,"merged_at":null,"base":{"ref":"main"}}], [{"number":3,"merged_at":"today","base":{"ref":"other"}}]]'
        self.assertEqual([item["number"] for item in merged_pull_requests("owner/repo", ["first", "second"])], [1])
        self.assertIn("--paginate", command.call_args.args)

    def mock_git(self, tags="v0.1.0", commits="new", message="fix: change"):
        def command(*args):
            if args[1:3] == ("rev-parse", "HEAD"):
                return "head"
            if args[1] == "merge-base":
                return ""
            if args[1] == "rev-parse":
                return "head"
            if args[1] == "tag":
                return tags
            if args[1] == "rev-list":
                return commits
            if args[1] == "log":
                return message
            raise AssertionError(args)
        return command

    @patch("release_plan.merged_pull_requests", return_value=[])
    def test_unchanged_week_skips(self, requests) -> None:
        with patch("release_plan.run", side_effect=self.mock_git(commits="")):
            self.assertEqual(plan("owner/repo")["publish"], "false")
        requests.assert_not_called()

    @patch("release_plan.merged_pull_requests", return_value=[])
    def test_semver_order_not_lexical(self, requests) -> None:
        with patch("release_plan.run", side_effect=self.mock_git(tags="v0.9.0\nv0.10.0\nv9.0.0-rc.1")):
            self.assertEqual(plan("owner/repo")["tag"], "v0.10.1")

    @patch("release_plan.merged_pull_requests", return_value=[])
    def test_first_release_uses_chart(self, requests) -> None:
        with patch("release_plan.run", side_effect=self.mock_git(tags="")), patch("release_plan.Path.read_text", return_value="version: 0.1.0"):
            self.assertEqual(plan("owner/repo")["tag"], "v0.1.0")
        requests.assert_not_called()

    def test_explicit_tag_is_not_bumped(self) -> None:
        with patch("release_plan.run", side_effect=self.mock_git(tags="v0.1.0\nv0.2.0")):
            self.assertEqual(plan("owner/repo", "v0.2.0")["tag"], "v0.2.0")

    def test_old_tag_is_rejected(self) -> None:
        with patch("release_plan.run", side_effect=self.mock_git(tags="v0.1.0\nv0.2.0")):
            with self.assertRaises(ValueError):
                plan("owner/repo", "v0.1.0")

    def test_non_main_commit_is_rejected(self) -> None:
        with patch("release_plan.run", side_effect=["head", subprocess.CalledProcessError(1, "git")]):
            with self.assertRaises(subprocess.CalledProcessError):
                plan("owner/repo")


if __name__ == "__main__":
    unittest.main()