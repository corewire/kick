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


def validate_repository(repo_root: Path) -> ValidationResult:
    errors: list[str] = []
    features_file = repo_root / "traceability" / "features.yaml"
    scenarios_file = repo_root / "traceability" / "e2e-scenarios.yaml"
    features = load_yaml(features_file).get("features", [])
    scenarios = load_yaml(scenarios_file).get("scenarios", [])

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

    report = build_report(rows)
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
