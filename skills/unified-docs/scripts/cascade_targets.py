#!/usr/bin/env python3
import json
import re
import sys
from pathlib import Path


EXCLUDED_PARTS = {
    ".claude", ".agents", ".agent", ".codex", ".cursor", ".github", ".vscode", ".idea",
    "tmp", "temp", ".tmp", ".cache", "dist", "build", "coverage", "node_modules", "vendor",
    ".git",
}


def is_excluded(path: Path) -> bool:
    return any(part in EXCLUDED_PARTS for part in path.parts)


def parse_list(value: str | list[str]) -> list[str]:
    if isinstance(value, list):
        return value
    return [item.strip() for item in re.findall(r"[A-Za-z0-9_./-]+", value) if item.strip()]


def parse_frontmatter(path: Path) -> dict[str, str | list[str]]:
    lines = path.read_text(encoding="utf-8").splitlines()
    if not lines or lines[0].strip() != "---":
        return {}
    data = {}
    current_key = None
    for line in lines[1:]:
        stripped = line.strip()
        if stripped == "---":
            break
        if stripped.startswith("-") and current_key:
            value = stripped[1:].strip()
            existing = data.get(current_key)
            if not isinstance(existing, list):
                existing = []
            existing.append(value)
            data[current_key] = existing
            continue
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        current_key = key.strip()
        data[current_key] = value.strip()
    return data


if len(sys.argv) < 2:
    raise SystemExit("usage: cascade_targets.py <file> [root]")

target = Path(sys.argv[1]).resolve()
root = Path(sys.argv[2]).resolve() if len(sys.argv) > 2 else target.parent.resolve()

frontmatter = parse_frontmatter(target)
outgoing = {
    "depends-on": parse_list(frontmatter.get("depends-on", "")),
    "updates": parse_list(frontmatter.get("updates", "")),
}
metadata = {
    "type": frontmatter.get("type", ""),
    "kind": parse_list(frontmatter.get("kind", "")),
    "status": frontmatter.get("status", ""),
    "replacedBy": frontmatter.get("replacedBy", ""),
}

incoming = []
for doc in root.rglob("*.md"):
    if doc == target or is_excluded(doc.relative_to(root)):
        continue
    fm = parse_frontmatter(doc)
    refs = parse_list(fm.get("depends-on", "")) + parse_list(fm.get("updates", ""))
    if target.name in refs or str(target.relative_to(root)) in refs:
        incoming.append(str(doc.relative_to(root)))

print(json.dumps({
    "file": str(target),
    "metadata": metadata,
    "outgoing": outgoing,
    "incoming": sorted(incoming),
}, indent=2))
