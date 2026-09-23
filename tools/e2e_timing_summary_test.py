#!/usr/bin/env python3

from __future__ import annotations

import shutil
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from e2e_timing_summary import collect, render

REPORT = """<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="chainsaw">
    <testcase name="kick-e2e-001-fast" time="3.5"/>
    <testcase name="kick-e2e-002-slow" time="42.25"/>
    <testcase name="kick-e2e-003-broken" time="7">
      <failure message="boom"/>
    </testcase>
  </testsuite>
</testsuites>
"""


class E2ETimingSummaryTests(unittest.TestCase):
    def setUp(self) -> None:
        self.tmpdir = Path(tempfile.mkdtemp(prefix="kick-e2e-timing-"))
        (self.tmpdir / "core.xml").write_text(REPORT, encoding="utf-8")

    def tearDown(self) -> None:
        shutil.rmtree(self.tmpdir)

    def test_collect_reads_every_report_in_a_directory(self) -> None:
        cases = collect([self.tmpdir])
        self.assertEqual([case.name for case in cases], [
            "kick-e2e-001-fast",
            "kick-e2e-002-slow",
            "kick-e2e-003-broken",
        ])
        self.assertEqual([case.suite for case in cases], ["core"] * 3)
        self.assertEqual(cases[1].seconds, 42.25)
        self.assertEqual([case.failed for case in cases], [False, False, True])

    def test_render_sorts_slowest_first_and_marks_failures(self) -> None:
        output = render(collect([self.tmpdir]), top=2)
        self.assertIn("3 scenarios, 53s of scenario time, 1 failed.", output)
        rows = [line for line in output.splitlines() if line.startswith("| core")]
        self.assertEqual(len(rows), 2)
        self.assertIn("kick-e2e-002-slow", rows[0])
        self.assertIn("kick-e2e-003-broken (failed)", rows[1])

    def test_render_without_reports(self) -> None:
        self.assertIn("No e2e timing reports found", render([], top=5))


if __name__ == "__main__":
    unittest.main()
