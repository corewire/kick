#!/usr/bin/env python3

from __future__ import annotations

import argparse
import re
from pathlib import Path

import yaml


STRUCT_START_RE = re.compile(r"^type\s+(\w+)\s+struct\s*\{")
FIELD_RE = re.compile(r'^\s*(\w+)\s+([^\s`]+)\s+`json:"([^"]+)"`')


def normalize_type(go_type: str) -> str:
    cleaned = go_type.lstrip("[]*")
    if "." in cleaned:
        cleaned = cleaned.split(".")[-1]
    return cleaned


def parse_structs(api_dir: Path) -> tuple[dict[str, dict], list[str]]:
    structs: dict[str, dict] = {}
    roots: list[str] = []

    for path in sorted(api_dir.glob("*_types.go")):
        if path.name == "zz_generated.deepcopy.go":
            continue
        lines = path.read_text(encoding="utf-8").splitlines()
        index = 0
        while index < len(lines):
            match = STRUCT_START_RE.match(lines[index].strip())
            if not match:
                index += 1
                continue

            type_name = match.group(1)
            index += 1
            fields: list[dict] = []
            while index < len(lines) and lines[index].strip() != "}":
                field_match = FIELD_RE.match(lines[index])
                if field_match:
                    json_tag = field_match.group(3).split(",", 1)[0]
                    if json_tag and json_tag != "-":
                        fields.append(
                            {
                                "name": field_match.group(1),
                                "type": field_match.group(2),
                                "json": json_tag,
                            }
                        )
                index += 1
            structs[type_name] = {"file": str(path.relative_to(api_dir.parent.parent)), "fields": fields}

    for type_name, info in structs.items():
        for field in info["fields"]:
            if field["json"] == "spec" and normalize_type(field["type"]).endswith("Spec"):
                roots.append(normalize_type(field["type"]))
            if field["json"] == "status" and normalize_type(field["type"]).endswith("Status"):
                roots.append(normalize_type(field["type"]))

    return structs, roots


def expand_paths(structs: dict[str, dict], type_name: str, prefix: str = "") -> list[str]:
    info = structs.get(type_name)
    if info is None:
        return []
    paths: list[str] = []
    for field in info["fields"]:
        current = f"{prefix}.{field['json']}" if prefix else field["json"]
        nested_type = normalize_type(field["type"])
        if nested_type in structs:
            nested_paths = expand_paths(structs, nested_type, current)
            if nested_paths:
                paths.extend(nested_paths)
            else:
                paths.append(current)
        else:
            paths.append(current)
    return paths


def generate(api_dir: Path) -> dict:
    structs, roots = parse_structs(api_dir)
    resources = []
    seen: set[str] = set()
    for root in roots:
        if root in seen:
            continue
        seen.add(root)
        info = structs[root]
        resources.append(
            {
                "type": root,
                "file": info["file"],
                "generated": True,
                "fields": [{"path": path} for path in expand_paths(structs, root)],
            }
        )
    return {"version": 1, "resources": resources}


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate API field coverage skeleton")
    parser.add_argument("--api-dir", default="api/v1alpha1")
    parser.add_argument("--output", default="traceability/api-field-coverage.generated.yaml")
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[1]
    api_dir = (repo_root / args.api_dir).resolve()
    output = Path(args.output)
    if not output.is_absolute():
        output = repo_root / output

    generated = generate(api_dir)
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(yaml.safe_dump(generated, sort_keys=False), encoding="utf-8")
    print(f"Generated API field coverage skeleton: {output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())