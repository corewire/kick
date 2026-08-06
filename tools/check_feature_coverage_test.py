#!/usr/bin/env python3

from __future__ import annotations

import shutil
import sys
import tempfile
import unittest
from pathlib import Path

import yaml

sys.path.insert(0, str(Path(__file__).resolve().parent))

from check_feature_coverage import validate_repository


class FeatureCoverageCheckerTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmpdir = Path(tempfile.mkdtemp(prefix="kick-feature-coverage-"))
        self._create_valid_repo(self.tmpdir)

    def tearDown(self) -> None:
        shutil.rmtree(self.tmpdir)

    def _create_valid_repo(self, root: Path) -> None:
        (root / "traceability").mkdir(parents=True, exist_ok=True)
        (root / "docs" / "tasks").mkdir(parents=True, exist_ok=True)
        (root / "internal").mkdir(parents=True, exist_ok=True)
        (root / "test" / "envtest").mkdir(parents=True, exist_ok=True)
        (root / "tools").mkdir(parents=True, exist_ok=True)

        (root / "docs" / "tasks" / "01-example.md").write_text(
            "# Task\n\nFeature IDs:\n- KICK-FEAT-001\n",
            encoding="utf-8",
        )
        (root / "internal" / "unit_test.go").write_text("package internal\n", encoding="utf-8")
        (root / "test" / "envtest" / "envtest_test.go").write_text("package envtest\n", encoding="utf-8")
        (root / "tools" / "check_feature_coverage_test.py").write_text("# KICK-FEAT-001\n", encoding="utf-8")

        scenario_dir = root / "test" / "e2e" / "scenarios" / "KICK-E2E-001-example"
        scenario_dir.mkdir(parents=True, exist_ok=True)
        (scenario_dir / "chainsaw-test.yaml").write_text("apiVersion: chainsaw.kyverno.io/v1alpha1\n", encoding="utf-8")
        (scenario_dir / "trace.yaml").write_text("scenarioID: KICK-E2E-001\n", encoding="utf-8")

        features = {
            "version": 1,
            "features": [
                {
                    "id": "KICK-FEAT-001",
                    "name": "Example",
                    "spec": "specs/example.md",
                    "task": "tasks/01-example.md",
                    "tests": {"unit": "required", "envtest": "required", "e2e": "required"},
                    "coverage": {
                        "unit": ["internal/unit_test.go"],
                        "envtest": ["test/envtest/envtest_test.go"],
                    },
                    "e2e": ["KICK-E2E-001"],
                }
            ],
        }
        scenarios = {
            "version": 1,
            "scenarios": [
                {
                    "id": "KICK-E2E-001",
                    "name": "example",
                    "directory": "test/e2e/scenarios/KICK-E2E-001-example",
                    "status": "required",
                    "features": ["KICK-FEAT-001"],
                }
            ],
        }
        (root / "traceability" / "features.yaml").write_text(yaml.safe_dump(features, sort_keys=False), encoding="utf-8")
        (root / "traceability" / "e2e-scenarios.yaml").write_text(yaml.safe_dump(scenarios, sort_keys=False), encoding="utf-8")

    def _write_features(self, mutate):
        path = self.tmpdir / "traceability" / "features.yaml"
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
        mutate(data)
        path.write_text(yaml.safe_dump(data, sort_keys=False), encoding="utf-8")

    def _write_scenarios(self, mutate):
        path = self.tmpdir / "traceability" / "e2e-scenarios.yaml"
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
        mutate(data)
        path.write_text(yaml.safe_dump(data, sort_keys=False), encoding="utf-8")

    def test_valid_repo_passes(self):
        result = validate_repository(self.tmpdir)
        self.assertEqual([], result.errors)
        self.assertIn("| Feature | Unit | Envtest | E2E |", result.report)

    def test_required_level_without_coverage_fails(self):
        def mutate(data):
            data["features"][0]["coverage"]["unit"] = []

        self._write_features(mutate)
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("unit is required but no coverage paths are configured" in e for e in result.errors))

    def test_required_e2e_without_mapping_fails(self):
        def mutate(data):
            data["features"][0]["e2e"] = []

        self._write_features(mutate)
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("e2e is required but no scenarios are mapped" in e for e in result.errors))

    def test_unknown_feature_in_scenario_fails(self):
        def mutate(data):
            data["scenarios"][0]["features"] = ["KICK-FEAT-999"]

        self._write_scenarios(mutate)
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("unknown feature" in e for e in result.errors))

    def test_missing_required_scenario_directory_fails(self):
        shutil.rmtree(self.tmpdir / "test" / "e2e" / "scenarios" / "KICK-E2E-001-example")
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("required scenario directory is missing" in e or "directory or required files are missing" in e for e in result.errors))

    def test_duplicate_ids_fail(self):
        def mutate_features(data):
            data["features"].append(dict(data["features"][0]))

        def mutate_scenarios(data):
            data["scenarios"].append(dict(data["scenarios"][0]))

        self._write_features(mutate_features)
        self._write_scenarios(mutate_scenarios)
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("duplicate feature IDs" in e for e in result.errors))
        self.assertTrue(any("duplicate scenario IDs" in e for e in result.errors))

    def test_not_applicable_without_rationale_fails(self):
        def mutate(data):
            data["features"][0]["tests"]["envtest"] = "not_applicable"
            data["features"][0].pop("rationale", None)

        self._write_features(mutate)
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("envtest=not_applicable requires rationale" in e for e in result.errors))

    def test_task_unknown_feature_reference_fails(self):
        (self.tmpdir / "docs" / "tasks" / "01-example.md").write_text(
            "Feature IDs:\n- KICK-FEAT-999\n",
            encoding="utf-8",
        )
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("unknown feature ID reference in tests/tasks" in e for e in result.errors))

    def test_planned_only_required_coverage_fails(self):
        def mutate(data):
            data["features"][0]["coverage"]["unit"] = ["planned:add unit test"]

        self._write_features(mutate)
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("planned-only coverage entries" in e for e in result.errors))


if __name__ == "__main__":
    unittest.main()
