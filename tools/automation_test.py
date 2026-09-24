from __future__ import annotations

import json
import os
import re
import subprocess
import tarfile
import unittest
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[1]


class AutomationTests(unittest.TestCase):
    def test_go_versions_are_equal_and_grouped(self) -> None:
        versions = []
        for filename in ["go.mod", "docs/go.mod"]:
            match = re.search(r"^go (\S+)$", (ROOT / filename).read_text(), re.MULTILINE)
            self.assertIsNotNone(match)
            versions.append(match.group(1))
        docker = re.search(r"FROM golang:([^\s@]+)", (ROOT / "Dockerfile").read_text())
        versions.append(docker.group(1))
        self.assertEqual(len(set(versions)), 1)
        config = json.loads((ROOT / "renovate.json").read_text())
        rules = [rule for rule in config["packageRules"] if rule.get("groupSlug") == "go-version"]
        self.assertEqual({manager for rule in rules for manager in rule["matchManagers"]}, {"gomod", "dockerfile"})
        self.assertTrue(all(rule["separateMajorMinor"] is False for rule in rules))

    def test_all_tool_pins_have_renovate_matchers(self) -> None:
        config = json.loads((ROOT / "renovate.json").read_text())
        patterns = [
            re.compile(pattern.replace("(?<", "(?P<"))
            for manager in config["customManagers"]
            if manager["managerFilePatterns"] == ["/^hack/tool-versions\\.mk$/"]
            for pattern in manager["matchStrings"]
        ]
        for line in (ROOT / "hack/tool-versions.mk").read_text().splitlines():
            if " ?= " in line:
                with self.subTest(pin=line):
                    self.assertTrue(any(pattern.search(line) for pattern in patterns))

    def test_fixture_paths_are_not_ignored(self) -> None:
        config = json.loads((ROOT / "renovate.json").read_text())
        for path in ["test/e2e/scenarios/example/resources.yaml", "tools/requirements.txt", "docs/go.mod"]:
            self.assertFalse(any(Path(path).match(pattern) for pattern in config["ignorePaths"]))

    def test_ci_rejects_unintended_skips(self) -> None:
        workflow = yaml.safe_load((ROOT / ".github/workflows/ci.yml").read_text())
        script = workflow["jobs"]["ci"]["steps"][0]["run"]
        for code, job, result, expected in [
            ("true", "e2e", "success", 0),
            ("true", "e2e", "skipped", 1),
            ("false", "e2e", "skipped", 0),
            ("false", "lint", "skipped", 1),
            ("true", "race-tests", "failure", 1),
            ("true", "image", "cancelled", 1),
        ]:
            needs = {name: {"result": "success"} for name in workflow["jobs"]["ci"]["needs"]}
            needs["changes"]["outputs"] = {"code": code}
            needs[job]["result"] = result
            with self.subTest(code=code, job=job, result=result):
                completed = subprocess.run(["bash", "-c", script], env={**os.environ, "NEEDS": json.dumps(needs)}, capture_output=True)
                self.assertEqual(completed.returncode, expected, completed.stderr)

    def test_publish_depends_on_tests_and_artifacts(self) -> None:
        workflow = yaml.safe_load((ROOT / ".github/workflows/release.yml").read_text())
        jobs = workflow["jobs"]
        self.assertEqual(jobs["ci"]["uses"], "./.github/workflows/ci.yml")
        self.assertIn("ci", jobs["container"]["needs"])
        self.assertIn("container", jobs["artifacts"]["needs"])
        self.assertIn("artifacts", jobs["publish"]["needs"])
        self.assertFalse(workflow["concurrency"]["cancel-in-progress"])

    def test_release_chart_uses_matching_image_without_changing_sources(self) -> None:
        original = (ROOT / "charts/kick/values.yaml").read_bytes()
        version = "0.0.0-ci"
        subprocess.run(["make", "release-chart", f"VERSION={version}"], cwd=ROOT, check=True, capture_output=True)
        with tarfile.open(ROOT / f"dist/kick-{version}.tgz") as archive:
            chart = yaml.safe_load(archive.extractfile("kick/Chart.yaml"))
            values = yaml.safe_load(archive.extractfile("kick/values.yaml"))
        self.assertEqual(chart["version"], version)
        self.assertEqual(chart["appVersion"], f"v{version}")
        self.assertEqual(values["image"]["tag"], f"v{version}")
        self.assertEqual(values["image"]["repository"], "ghcr.io/corewire/kick")
        self.assertEqual((ROOT / "charts/kick/values.yaml").read_bytes(), original)


if __name__ == "__main__":
    unittest.main()