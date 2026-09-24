#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
from pathlib import Path

import yaml


VERSION = re.compile(r"v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)")
CONVENTIONAL = re.compile(r"^([a-z]+)(?:\([^\n]+\))?(!)?:\s", re.MULTILINE)


def version_tuple(tag: str) -> tuple[int, ...]:
    match = VERSION.fullmatch(tag)
    if not match:
        raise ValueError(f"Not a stable release tag: {tag}")
    return tuple(int(part) for part in match.groups())


def bump_level(messages: list[str], pull_requests: list[dict]) -> str:
    labels = {
        label["name"].lower()
        for pull_request in pull_requests
        for label in pull_request.get("labels", [])
    }
    texts = messages + [
        f"{pull_request['title']}\n{pull_request.get('body') or ''}"
        for pull_request in pull_requests
    ]
    commits = [match for text in texts for match in CONVENTIONAL.finditer(text)]
    if "breaking" in labels or any(match.group(2) for match in commits) or any(
        re.search(r"^BREAKING[ -]CHANGE:\s", text, re.MULTILINE) for text in texts
    ):
        return "major"
    if "feature" in labels or any(match.group(1) == "feat" for match in commits):
        return "minor"
    return "patch"


def next_tag(previous: str, level: str) -> str:
    major, minor, patch = version_tuple(previous)
    if level == "major":
        return f"v{major + 1}.0.0"
    if level == "minor":
        return f"v{major}.{minor + 1}.0"
    if level == "patch":
        return f"v{major}.{minor}.{patch + 1}"
    raise ValueError(f"Unknown release level: {level}")


def run(*args: str) -> str:
    return subprocess.check_output(args, text=True).strip()


def merged_pull_requests(repository: str, commits: list[str]) -> list[dict]:
    found = {}
    for commit in commits:
        pages = json.loads(run(
            "gh", "api", "--paginate", "--slurp",
            f"repos/{repository}/commits/{commit}/pulls?per_page=100",
        ))
        for page in pages:
            for pull_request in page:
                if pull_request.get("merged_at") and pull_request["base"]["ref"] == "main":
                    found[pull_request["number"]] = pull_request
    return list(found.values())


def plan(repository: str, requested_tag: str = "") -> dict[str, str]:
    sha = run("git", "rev-parse", "HEAD")
    run("git", "merge-base", "--is-ancestor", sha, "origin/main")
    if requested_tag:
        version_tuple(requested_tag)
        if run("git", "rev-parse", f"{requested_tag}^{{commit}}") != sha:
            raise ValueError("Release tag does not point to the checked-out commit")
    tags = run("git", "tag", "--merged", sha, "--list", "v*").splitlines()
    stable = [tag for tag in tags if VERSION.fullmatch(tag) and tag != requested_tag]
    previous = max(stable, key=version_tuple) if stable else ""
    revision = f"{previous}..{sha}" if previous else sha
    commits = run("git", "rev-list", revision).splitlines()
    if requested_tag:
        if previous and version_tuple(requested_tag) <= version_tuple(previous):
            raise ValueError("Release tag must be newer than the previous stable release")
        tag = requested_tag
    elif not previous:
        chart = yaml.safe_load(Path("charts/kick/Chart.yaml").read_text())
        tag = f"v{chart['version']}"
    elif commits:
        messages = run("git", "log", "--format=%B%x00", revision).split("\0")
        tag = next_tag(previous, bump_level(messages, merged_pull_requests(repository, commits)))
    else:
        tag = previous
    version_tuple(tag)
    return {
        "publish": str(bool(commits) or bool(requested_tag)).lower(),
        "sha": sha,
        "tag": tag,
        "version": tag[1:],
        "previous_tag": previous,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repository", default=os.environ.get("GITHUB_REPOSITORY"), required=False)
    parser.add_argument("--tag", default="")
    args = parser.parse_args()
    if not args.repository:
        parser.error("--repository or GITHUB_REPOSITORY is required")
    result = plan(args.repository, args.tag)
    print(json.dumps(result, indent=2))
    if output := os.environ.get("GITHUB_OUTPUT"):
        with open(output, "a", encoding="utf-8") as stream:
            stream.write("".join(f"{key}={value}\n" for key, value in result.items()))


if __name__ == "__main__":
    main()