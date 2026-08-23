#!/usr/bin/env python3
"""Turn chainsaw JUnit reports into a markdown timing table.

CI writes one JUnit report per e2e suite (`make test-e2e-* E2E_REPORT_DIR=...`).
This script aggregates them so the slowest scenarios are visible in the job
summary instead of buried in the raw log.
"""

from __future__ import annotations

import argparse
import sys
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import Iterable, NamedTuple


class Case(NamedTuple):
    suite: str
    name: str
    seconds: float
    failed: bool


def parse_report(path: Path) -> list[Case]:
    """Read one JUnit XML file into cases. The suite name is the file stem."""
    root = ET.parse(path).getroot()
    suite = path.stem
    cases: list[Case] = []
    for case in root.iter("testcase"):
        name = case.get("name", "")
        if not name:
            continue
        try:
            seconds = float(case.get("time", "0"))
        except ValueError:
            seconds = 0.0
        failed = any(child.tag in ("failure", "error") for child in case)
        cases.append(Case(suite, name, seconds, failed))
    return cases


def collect(paths: Iterable[Path]) -> list[Case]:
    """Expand directories to their *.xml reports and parse everything found."""
    cases: list[Case] = []
    for path in paths:
        files = sorted(path.glob("**/*.xml")) if path.is_dir() else [path]
        for report in files:
            cases.extend(parse_report(report))
    return cases


def render(cases: list[Case], top: int) -> str:
    if not cases:
        return "No e2e timing reports found.\n"

    total = sum(case.seconds for case in cases)
    failures = sum(1 for case in cases if case.failed)
    lines = [
        "## e2e scenario timings",
        "",
        f"{len(cases)} scenarios, {total:.0f}s of scenario time, {failures} failed.",
        "",
        "| Suite | Seconds | Scenario |",
        "|---|---:|---|",
    ]
    slowest = sorted(cases, key=lambda case: case.seconds, reverse=True)[:top]
    for case in slowest:
        marker = " (failed)" if case.failed else ""
        lines.append(f"| {case.suite} | {case.seconds:.1f} | {case.name}{marker} |")
    lines.append("")
    return "\n".join(lines)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "paths",
        nargs="+",
        type=Path,
        help="JUnit report files or directories containing them",
    )
    parser.add_argument(
        "--top",
        type=int,
        default=15,
        help="how many of the slowest scenarios to list (default: 15)",
    )
    args = parser.parse_args(argv)

    existing = [path for path in args.paths if path.exists()]
    sys.stdout.write(render(collect(existing), args.top))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
