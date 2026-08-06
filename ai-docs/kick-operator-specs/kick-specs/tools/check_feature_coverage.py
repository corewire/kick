#!/usr/bin/env python3
"""Validate the specification-level KICK feature/e2e mapping.

The implementation repository should extend this checker to inspect real Go tests,
Envtest suites, Chainsaw trace files, and implementation status.
"""
from __future__ import annotations

import sys
from pathlib import Path

try:
    import yaml
except ImportError:
    print("PyYAML is required: python -m pip install pyyaml", file=sys.stderr)
    raise SystemExit(2)

ROOT = Path(__file__).resolve().parents[1]
FEATURES = ROOT / "traceability" / "features.yaml"
SCENARIOS = ROOT / "traceability" / "e2e-scenarios.yaml"
VALID = {"required", "optional", "not_applicable"}


def load(path: Path):
    with path.open(encoding="utf-8") as handle:
        return yaml.safe_load(handle)


def main() -> int:
    errors: list[str] = []
    features = load(FEATURES).get("features", [])
    scenarios = load(SCENARIOS).get("scenarios", [])

    feature_ids = [f.get("id") for f in features]
    scenario_ids = [s.get("id") for s in scenarios]
    if len(feature_ids) != len(set(feature_ids)):
        errors.append("duplicate feature IDs")
    if len(scenario_ids) != len(set(scenario_ids)):
        errors.append("duplicate scenario IDs")

    known_features = set(feature_ids)
    known_scenarios = set(scenario_ids)
    scenario_map = {s["id"]: s for s in scenarios}

    for feature in features:
        fid = feature.get("id", "<missing>")
        tests = feature.get("tests", {})
        rationale = feature.get("rationale", {})
        for level in ("unit", "envtest", "e2e"):
            state = tests.get(level)
            if state not in VALID:
                errors.append(f"{fid}: invalid or missing {level} requirement: {state!r}")
            if state == "not_applicable" and not rationale.get(level):
                errors.append(f"{fid}: {level}=not_applicable requires rationale")
        mapped = feature.get("e2e", [])
        if tests.get("e2e") == "required" and not mapped:
            errors.append(f"{fid}: e2e is required but no scenarios are mapped")
        for sid in mapped:
            if sid not in known_scenarios:
                errors.append(f"{fid}: unknown e2e scenario {sid}")
            elif fid not in scenario_map[sid].get("features", []):
                errors.append(f"{fid}: {sid} does not map back to the feature")

    for scenario in scenarios:
        sid = scenario.get("id", "<missing>")
        if scenario.get("status") == "required" and not scenario.get("features"):
            errors.append(f"{sid}: required scenario maps to no feature")
        for fid in scenario.get("features", []):
            if fid not in known_features:
                errors.append(f"{sid}: unknown feature {fid}")
            else:
                feature = next(f for f in features if f["id"] == fid)
                if sid not in feature.get("e2e", []):
                    errors.append(f"{sid}: {fid} does not map back to the scenario")

    if errors:
        print("Feature coverage validation failed:")
        for error in errors:
            print(f"- {error}")
        return 1

    print(f"Feature coverage mapping valid: {len(features)} features, {len(scenarios)} e2e scenarios")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
