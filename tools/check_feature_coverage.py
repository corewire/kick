#!/usr/bin/env python3
"""Validate KICK feature-to-test traceability and emit a markdown report."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

try:
    import yaml
except ImportError:
    print("PyYAML is required: python -m pip install pyyaml", file=sys.stderr)
    raise SystemExit(2)

VALID = {"required", "optional", "not_applicable"}
DISABLED_SCENARIO_STATUSES = {"disabled", "skipped", "planned"}
FEATURE_ID_RE = re.compile(r"KICK-FEAT-[0-9]{3}$")
SCENARIO_ID_RE = re.compile(r"KICK-E2E-[0-9]{3}$")
FEATURE_TOKEN_RE = re.compile(r"KICK-FEAT-[0-9]{3}")
E2E_TODO_RE = re.compile(r"TODO: implement KICK-E2E")


@dataclass
class ValidationResult:
    errors: list[str]
    report: str


def load_yaml(path: Path) -> dict:
    with path.open(encoding="utf-8") as handle:
        loaded = yaml.safe_load(handle)
    if not isinstance(loaded, dict):
        raise ValueError(f"invalid yaml root in {path}")
    return loaded


def find_feature_tokens(paths: Iterable[Path]) -> set[str]:
    tokens: set[str] = set()
    for path in paths:
        if not path.exists() or not path.is_file():
            continue
        text = path.read_text(encoding="utf-8", errors="ignore")
        tokens.update(FEATURE_TOKEN_RE.findall(text))
    return tokens


def resolve_coverage_paths(repo_root: Path, paths: list[str]) -> tuple[list[Path], list[str]]:
    existing: list[Path] = []
    planned: list[str] = []
    for item in paths:
        if item.startswith("planned:"):
            planned.append(item)
            continue
        resolved = repo_root / item
        if resolved.exists():
            existing.append(resolved)
    return existing, planned


def level_status(required: str, has_coverage: bool, planned_only: bool, has_rationale: bool) -> str:
    if required == "required":
        if planned_only:
            return "Planned only"
        return "Pass" if has_coverage else "Missing"
    if required == "not_applicable":
        return "N/A" if has_rationale else "N/A missing rationale"
    return "Optional" if not has_coverage else "Optional covered"


def build_report(rows: list[dict]) -> str:
    header = "| Feature | Unit | Envtest | E2E | Required E2E scenarios | Result |"
    sep = "|---|---:|---:|---:|---|---|"
    lines = ["# Feature Coverage Report", "", header, sep]
    for row in rows:
        req_scenarios = ", ".join(row["required_e2e_scenarios"]) if row["required_e2e_scenarios"] else "-"
        lines.append(
            "| {feature} | {unit} | {envtest} | {e2e} | {scenarios} | {result} |".format(
                feature=row["feature"],
                unit=row["unit"],
                envtest=row["envtest"],
                e2e=row["e2e"],
                scenarios=req_scenarios,
                result=row["result"],
            )
        )
    return "\n".join(lines) + "\n"


def build_api_field_report(rows: list[dict]) -> str:
    header = "| Type | Field | Required | Unit | Envtest | E2E | Result | Evidence |"
    sep = "|---|---|---:|---:|---:|---:|---|---|"
    lines = ["# API Field Coverage Report", "", header, sep]
    for row in rows:
        lines.append(
            "| {type_name} | {field_path} | {required} | {unit} | {envtest} | {e2e} | {result} | {evidence} |".format(
                type_name=row["type_name"],
                field_path=row["field_path"],
                required="yes" if row["required"] else "no",
                unit=row["unit"],
                envtest=row["envtest"],
                e2e=row["e2e"],
                result=row["result"],
                evidence=row["evidence"],
            )
        )
    return "\n".join(lines) + "\n"


def file_level_status(paths: list[str], repo_root: Path) -> tuple[str, list[str], list[str]]:
    existing, planned = resolve_coverage_paths(repo_root, paths)
    missing = [p for p in paths if not p.startswith("planned:") and not (repo_root / p).exists()]
    if existing:
        return "Covered", [str(p.relative_to(repo_root)) for p in existing], missing
    if planned:
        return "Planned", planned, missing
    return "Missing", [], missing


def scenario_level_status(scenarios: list[str], scenario_map: dict, repo_root: Path) -> tuple[str, list[str], list[str]]:
    if not scenarios:
        return "Missing", [], []

    evidence: list[str] = []
    missing: list[str] = []
    all_implemented = True

    for sid in scenarios:
        scenario = scenario_map.get(sid)
        if scenario is None:
            missing.append(sid)
            all_implemented = False
            continue
        directory = scenario.get("directory", "")
        chainsaw_path = repo_root / directory / "chainsaw-test.yaml"
        if not chainsaw_path.exists():
            missing.append(f"{sid}:chainsaw-test.yaml")
            all_implemented = False
            continue
        content = chainsaw_path.read_text(encoding="utf-8", errors="ignore")
        if E2E_TODO_RE.search(content):
            evidence.append(f"{sid}(scaffold)")
            all_implemented = False
        else:
            evidence.append(f"{sid}(covered)")

    if missing:
        return "Missing", evidence, missing
    if all_implemented:
        return "Covered", evidence, []
    return "Scaffolded", evidence, []


def validate_repository(repo_root: Path) -> ValidationResult:
    errors: list[str] = []
    features_file = repo_root / "traceability" / "features.yaml"
    scenarios_file = repo_root / "traceability" / "e2e-scenarios.yaml"
    api_fields_file = repo_root / "traceability" / "api-field-coverage.yaml"
    api_fields_generated_file = repo_root / "traceability" / "api-field-coverage.generated.yaml"
    features = load_yaml(features_file).get("features", [])
    scenarios = load_yaml(scenarios_file).get("scenarios", [])
    if not api_fields_file.exists():
        errors.append("missing traceability/api-field-coverage.yaml")
        api_fields = []
    else:
        api_fields = load_yaml(api_fields_file).get("resources", [])
    if not api_fields_generated_file.exists():
        errors.append("missing traceability/api-field-coverage.generated.yaml")
        api_generated = []
    else:
        api_generated = load_yaml(api_fields_generated_file).get("resources", [])

    feature_ids = [f.get("id") for f in features]
    scenario_ids = [s.get("id") for s in scenarios]

    if len(feature_ids) != len(set(feature_ids)):
        errors.append("duplicate feature IDs")
    if len(scenario_ids) != len(set(scenario_ids)):
        errors.append("duplicate scenario IDs")

    for fid in feature_ids:
        if not FEATURE_ID_RE.match(str(fid)):
            errors.append(f"invalid feature ID format: {fid!r}")
    for sid in scenario_ids:
        if not SCENARIO_ID_RE.match(str(sid)):
            errors.append(f"invalid scenario ID format: {sid!r}")

    known_features = set(feature_ids)
    known_scenarios = set(scenario_ids)
    feature_map = {f["id"]: f for f in features if isinstance(f, dict) and "id" in f}
    scenario_map = {s["id"]: s for s in scenarios if isinstance(s, dict) and "id" in s}

    # Unknown feature IDs referenced directly in tests or task docs must fail.
    task_files = list((repo_root / "docs" / "tasks").glob("*.md"))
    test_files = list(repo_root.rglob("*_test.go"))
    trace_files = list((repo_root / "test" / "e2e" / "scenarios").rglob("trace.yaml"))
    token_sources = task_files + test_files + trace_files
    for token in sorted(find_feature_tokens(token_sources)):
        if token not in known_features:
            errors.append(f"unknown feature ID reference in tests/tasks: {token}")

    rows: list[dict] = []

    for feature in features:
        if not isinstance(feature, dict):
            errors.append("invalid feature entry")
            continue

        fid = feature.get("id", "<missing>")
        tests = feature.get("tests", {})
        rationale = feature.get("rationale", {})
        coverage = feature.get("coverage", {})
        mapped = feature.get("e2e", []) or []

        missing_by_level: list[str] = []
        statuses: dict[str, str] = {}

        for level in ("unit", "envtest", "e2e"):
            state = tests.get(level)
            if state not in VALID:
                errors.append(f"{fid}: invalid or missing {level} requirement: {state!r}")

            if state == "not_applicable" and not rationale.get(level):
                errors.append(f"{fid}: {level}=not_applicable requires rationale")

        for level in ("unit", "envtest"):
            required = tests.get(level)
            configured_paths = coverage.get(level, []) or []
            existing, planned = resolve_coverage_paths(repo_root, configured_paths)
            has_cov = len(existing) > 0
            planned_only = len(planned) > 0 and not has_cov
            has_rationale = bool(rationale.get(level))
            statuses[level] = level_status(str(required), has_cov, planned_only, has_rationale)

            if required == "required":
                if not configured_paths:
                    errors.append(f"{fid}: {level} is required but no coverage paths are configured")
                    missing_by_level.append(level)
                elif planned_only:
                    errors.append(f"{fid}: {level} has planned-only coverage entries")
                    missing_by_level.append(level)
                elif not has_cov:
                    errors.append(f"{fid}: {level} is required but no configured coverage path exists")
                    missing_by_level.append(level)

        e2e_required = tests.get("e2e") == "required"
        missing_e2e: list[str] = []
        actual_e2e = 0
        if e2e_required and not mapped:
            errors.append(f"{fid}: e2e is required but no scenarios are mapped")
            missing_by_level.append("e2e")

        for sid in mapped:
            if sid not in known_scenarios:
                errors.append(f"{fid}: unknown e2e scenario {sid}")
                missing_e2e.append(sid)
                continue

            scenario = scenario_map[sid]
            if fid not in scenario.get("features", []):
                errors.append(f"{fid}: {sid} does not map back to the feature")

            status = scenario.get("status")
            directory = scenario.get("directory", "")
            dir_path = repo_root / directory if directory else None
            has_required_assets = bool(
                dir_path
                and dir_path.exists()
                and (dir_path / "chainsaw-test.yaml").exists()
                and (dir_path / "trace.yaml").exists()
            )

            if status == "required" and has_required_assets:
                actual_e2e += 1

            if e2e_required:
                if status != "required":
                    errors.append(f"{fid}: required scenario {sid} is not marked required")
                    missing_e2e.append(sid)
                if status in DISABLED_SCENARIO_STATUSES:
                    errors.append(f"{fid}: required scenario {sid} is disabled/skipped/planned")
                    missing_e2e.append(sid)
                if not has_required_assets:
                    errors.append(f"{fid}: required scenario {sid} directory or required files are missing")
                    missing_e2e.append(sid)

        if e2e_required and missing_e2e:
            missing_by_level.append("e2e")

        e2e_state = tests.get("e2e")
        statuses["e2e"] = level_status(
            str(e2e_state),
            has_coverage=actual_e2e > 0,
            planned_only=False,
            has_rationale=bool(rationale.get("e2e")),
        )
        if e2e_required and missing_e2e:
            statuses["e2e"] = "Missing"

        if missing_by_level:
            result = "FAIL (missing: " + ", ".join(sorted(set(missing_by_level))) + ")"
        else:
            result = "PASS"

        rows.append(
            {
                "feature": fid,
                "unit": statuses["unit"],
                "envtest": statuses["envtest"],
                "e2e": statuses["e2e"],
                "required_e2e_scenarios": mapped,
                "result": result,
            }
        )

    for scenario in scenarios:
        sid = scenario.get("id", "<missing>")
        if scenario.get("status") == "required" and not scenario.get("features"):
            errors.append(f"{sid}: required scenario maps to no feature")

        directory = scenario.get("directory", "")
        dir_path = repo_root / directory if directory else None
        if scenario.get("status") == "required":
            if not dir_path or not dir_path.exists():
                errors.append(f"{sid}: required scenario directory is missing")
            else:
                if not (dir_path / "chainsaw-test.yaml").exists():
                    errors.append(f"{sid}: required scenario missing chainsaw-test.yaml")
                if not (dir_path / "trace.yaml").exists():
                    errors.append(f"{sid}: required scenario missing trace.yaml")

        for fid in scenario.get("features", []):
            if fid not in known_features:
                errors.append(f"{sid}: unknown feature {fid}")
            else:
                feature = feature_map[fid]
                if sid not in feature.get("e2e", []):
                    errors.append(f"{sid}: {fid} does not map back to the scenario")

    api_rows: list[dict] = []
    seen_type_names: set[str] = set()
    generated_field_map: dict[str, set[str]] = {}

    for resource in api_generated:
        type_name = resource.get("type")
        if not type_name:
            continue
        generated_field_map[type_name] = {field.get("path") for field in resource.get("fields", []) if isinstance(field, dict) and field.get("path")}

    for resource in api_fields:
        if not isinstance(resource, dict):
            errors.append("invalid api-field resource entry")
            continue
        type_name = resource.get("type")
        if not type_name:
            errors.append("api-field resource is missing type")
            continue
        if type_name in seen_type_names:
            errors.append(f"duplicate api-field resource type: {type_name}")
            continue
        seen_type_names.add(type_name)
        expected_fields = generated_field_map.get(type_name)
        if expected_fields is None:
            errors.append(f"{type_name}: not present in generated api field skeleton")
            expected_fields = set()

        fields = resource.get("fields", []) or []
        seen_field_paths: set[str] = set()

        for field in fields:
            if not isinstance(field, dict):
                errors.append(f"{type_name}: invalid field entry")
                continue
            field_path = field.get("path")
            if not field_path:
                errors.append(f"{type_name}: field entry missing path")
                continue
            if field_path in seen_field_paths:
                errors.append(f"{type_name}: duplicate field path {field_path}")
                continue
            seen_field_paths.add(field_path)
            if field_path not in expected_fields:
                errors.append(f"{type_name}.{field_path}: not present in generated api field skeleton")

            coverage = field.get("coverage", {}) or {}
            required = bool(field.get("required", False))
            unit_paths = coverage.get("unit", []) or []
            envtest_paths = coverage.get("envtest", []) or []
            e2e_scenarios = coverage.get("e2e", []) or []

            unit_status, unit_evidence, unit_missing = file_level_status(unit_paths, repo_root)
            envtest_status, envtest_evidence, envtest_missing = file_level_status(envtest_paths, repo_root)
            e2e_status, e2e_evidence, e2e_missing = scenario_level_status(e2e_scenarios, scenario_map, repo_root)

            for missing in unit_missing:
                errors.append(f"{type_name}.{field_path}: unit coverage path missing: {missing}")
            for missing in envtest_missing:
                errors.append(f"{type_name}.{field_path}: envtest coverage path missing: {missing}")
            for missing in e2e_missing:
                errors.append(f"{type_name}.{field_path}: e2e coverage scenario missing: {missing}")

            direct_coverage = unit_status == "Covered" or envtest_status == "Covered" or e2e_status == "Covered"
            if required and not direct_coverage:
                errors.append(f"{type_name}.{field_path}: required field lacks implemented direct coverage")

            evidence = []
            if unit_evidence:
                evidence.append("unit=" + ",".join(unit_evidence))
            if envtest_evidence:
                evidence.append("envtest=" + ",".join(envtest_evidence))
            if e2e_evidence:
                evidence.append("e2e=" + ",".join(e2e_evidence))
            evidence_text = "; ".join(evidence) if evidence else "-"
            result = "PASS" if (not required or direct_coverage) else "FAIL"

            api_rows.append(
                {
                    "type_name": type_name,
                    "field_path": field_path,
                    "required": required,
                    "unit": unit_status,
                    "envtest": envtest_status,
                    "e2e": e2e_status,
                    "result": result,
                    "evidence": evidence_text,
                }
            )

        missing_generated = expected_fields - seen_field_paths
        for missing_path in sorted(missing_generated):
            errors.append(f"{type_name}.{missing_path}: missing from api-field-coverage.yaml")

    for generated_type in generated_field_map:
        if generated_type not in seen_type_names:
            errors.append(f"{generated_type}: missing resource entry in api-field-coverage.yaml")

    report = build_report(rows) + "\n" + build_api_field_report(api_rows)
    return ValidationResult(errors=errors, report=report)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validate KICK feature coverage")
    parser.add_argument(
        "--repo-root",
        default=str(Path(__file__).resolve().parents[1]),
        help="repository root containing traceability/",
    )
    parser.add_argument(
        "--report",
        default="traceability/feature-coverage-report.md",
        help="path to markdown report (relative to repo root unless absolute)",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    repo_root = Path(args.repo_root).resolve()
    report_path = Path(args.report)
    if not report_path.is_absolute():
        report_path = repo_root / report_path

    result = validate_repository(repo_root)

    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text(result.report, encoding="utf-8")

    if result.errors:
        print("Feature coverage validation failed:")
        for error in result.errors:
            print(f"- {error}")
        print(f"\nCoverage report written: {report_path}")
        return 1

    features = load_yaml(repo_root / "traceability" / "features.yaml").get("features", [])
    scenarios = load_yaml(repo_root / "traceability" / "e2e-scenarios.yaml").get("scenarios", [])
    print(f"Feature coverage mapping valid: {len(features)} features, {len(scenarios)} e2e scenarios")
    print(f"Coverage report written: {report_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
