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
from gen_api_field_coverage import generate


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
        api_fields = {
            "version": 1,
            "resources": [
                {
                    "type": "KickRequestSpec",
                    "file": "api/v1alpha1/kickrequest_types.go",
                    "fields": [
                        {
                            "path": "targetRef.name",
                            "required": True,
                            "coverage": {
                                "unit": ["internal/unit_test.go"],
                                "envtest": ["test/envtest/envtest_test.go"],
                                "e2e": ["KICK-E2E-001"],
                            },
                        }
                    ],
                }
            ],
        }
        api_generated = {
            "version": 1,
            "resources": [
                {
                    "type": "KickRequestSpec",
                    "file": "api/v1alpha1/kickrequest_types.go",
                    "generated": True,
                    "fields": [{"path": "targetRef.name"}],
                }
            ],
        }
        (root / "traceability" / "features.yaml").write_text(yaml.safe_dump(features, sort_keys=False), encoding="utf-8")
        (root / "traceability" / "e2e-scenarios.yaml").write_text(yaml.safe_dump(scenarios, sort_keys=False), encoding="utf-8")
        (root / "traceability" / "api-field-coverage.yaml").write_text(yaml.safe_dump(api_fields, sort_keys=False), encoding="utf-8")
        (root / "traceability" / "api-field-coverage.generated.yaml").write_text(yaml.safe_dump(api_generated, sort_keys=False), encoding="utf-8")

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

    def _write_api_fields(self, mutate):
        path = self.tmpdir / "traceability" / "api-field-coverage.yaml"
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
        mutate(data)
        path.write_text(yaml.safe_dump(data, sort_keys=False), encoding="utf-8")

    def test_valid_repo_passes(self):
        result = validate_repository(self.tmpdir)
        self.assertEqual([], result.errors)
        self.assertIn("| Feature | Unit | Envtest | E2E |", result.report)
        self.assertIn("| Type | Field | Required | Unit | Envtest | E2E | Result |", result.report)

    def test_missing_api_field_matrix_fails(self):
        (self.tmpdir / "traceability" / "api-field-coverage.yaml").unlink()
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("missing traceability/api-field-coverage.yaml" in e for e in result.errors))

    def test_missing_generated_api_field_matrix_fails(self):
        (self.tmpdir / "traceability" / "api-field-coverage.generated.yaml").unlink()
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("missing traceability/api-field-coverage.generated.yaml" in e for e in result.errors))

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

    def test_api_field_missing_coverage_path_fails(self):
        def mutate(data):
            data["resources"][0]["fields"][0]["coverage"]["unit"] = ["internal/missing_test.go"]

        self._write_api_fields(mutate)
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("unit coverage path missing" in e for e in result.errors))

    def test_api_field_unknown_e2e_scenario_fails(self):
        def mutate(data):
            data["resources"][0]["fields"][0]["coverage"]["e2e"] = ["KICK-E2E-999"]

        self._write_api_fields(mutate)
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("e2e coverage scenario missing" in e for e in result.errors))

    def test_required_api_field_without_direct_coverage_fails(self):
        def mutate(data):
            data["resources"][0]["fields"][0]["coverage"] = {"e2e": ["KICK-E2E-001"]}

        self._write_api_fields(mutate)
        (self.tmpdir / "test" / "e2e" / "scenarios" / "KICK-E2E-001-example" / "chainsaw-test.yaml").write_text(
            "echo \"TODO: implement KICK-E2E-001\"\n",
            encoding="utf-8",
        )
        result = validate_repository(self.tmpdir)
        self.assertTrue(any("required field lacks implemented direct coverage" in e for e in result.errors))

    def test_generated_field_missing_from_manual_matrix_fails(self):
        def mutate(data):
            data["resources"][0]["fields"].append({"path": "targetRef.kind"})

        path = self.tmpdir / "traceability" / "api-field-coverage.generated.yaml"
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
        mutate(data)
        path.write_text(yaml.safe_dump(data, sort_keys=False), encoding="utf-8")

        result = validate_repository(self.tmpdir)
        self.assertTrue(any("missing from api-field-coverage.yaml" in e for e in result.errors))

    def test_api_field_partial_e2e_status_is_reported(self):
        def mutate_features(data):
            data["features"][0]["e2e"] = ["KICK-E2E-001", "KICK-E2E-002"]

        def mutate_scenarios(data):
            data["scenarios"].append(
                {
                    "id": "KICK-E2E-002",
                    "name": "example-2",
                    "directory": "test/e2e/scenarios/KICK-E2E-002-example",
                    "status": "required",
                    "features": ["KICK-FEAT-001"],
                }
            )

        def mutate_api_fields(data):
            data["resources"][0]["fields"][0]["coverage"]["e2e"] = ["KICK-E2E-001", "KICK-E2E-002"]

        self._write_features(mutate_features)
        self._write_scenarios(mutate_scenarios)
        self._write_api_fields(mutate_api_fields)
        scenario_dir = self.tmpdir / "test" / "e2e" / "scenarios" / "KICK-E2E-002-example"
        scenario_dir.mkdir(parents=True, exist_ok=True)
        (scenario_dir / "chainsaw-test.yaml").write_text('echo "TODO: implement KICK-E2E-002"\n', encoding="utf-8")
        (scenario_dir / "trace.yaml").write_text("scenarioID: KICK-E2E-002\n", encoding="utf-8")

        result = validate_repository(self.tmpdir)
        self.assertIn("| KickRequestSpec | targetRef.name | yes | Covered | Covered | Partial | PASS |", result.report)


class ApiFieldGeneratorTests(unittest.TestCase):
    def test_generate_discovers_root_spec_and_status_paths(self):
        tmpdir = Path(tempfile.mkdtemp(prefix="kick-api-field-gen-"))
        try:
            api_dir = tmpdir / "api" / "v1alpha1"
            api_dir.mkdir(parents=True, exist_ok=True)
            (api_dir / "example_types.go").write_text(
                """
package v1alpha1

type InnerSpec struct {
    Name string `json:"name,omitempty"`
}

type ExampleSpec struct {
    Inner InnerSpec `json:"inner,omitempty"`
}

type ExampleStatus struct {
    Ready bool `json:"ready,omitempty"`
}

type Example struct {
    Spec ExampleSpec `json:"spec,omitempty"`
    Status ExampleStatus `json:"status,omitempty"`
}
""",
                encoding="utf-8",
            )
            generated = generate(api_dir)
        finally:
            shutil.rmtree(tmpdir)

        resources = {resource["type"]: resource for resource in generated["resources"]}
        self.assertIn("ExampleSpec", resources)
        self.assertIn("ExampleStatus", resources)
        self.assertEqual([{"path": "inner.name"}], resources["ExampleSpec"]["fields"])
        self.assertEqual([{"path": "ready"}], resources["ExampleStatus"]["fields"])


if __name__ == "__main__":
    unittest.main()
