#!/usr/bin/env python3
import re
import sys
from pathlib import Path

UNIVERSAL = {"title", "type", "kind", "created", "updated", "depends-on", "updates"}
ADR = {"adr-id", "status", "deciders", "decided", "supersededBy", "supersedes"}
PLAN = {"status"}
ALLOWED_TYPES = {"tutorial", "how-to", "reference", "explanation", "decision"}
ALLOWED_KINDS = {"plan", "spec", "adr", "ssot", "draft", "til"}
LIST_FIELDS = {"kind", "depends-on", "updates"}
EXCLUDED_PARTS = {
    ".claude", ".agents", ".agent", ".codex", ".cursor", ".github", ".vscode", ".idea",
    "tmp", "temp", ".tmp", ".cache", "dist", "build", "coverage", "node_modules", "vendor",
    ".git",
}


def parse_frontmatter(text: str) -> dict[str, str | list[str]]:
    lines = text.splitlines()
    if len(lines) < 3 or lines[0].strip() != "---":
        raise ValueError("missing frontmatter opening fence")
    fields = {}
    current_key = None
    for line in lines[1:]:
        stripped = line.strip()
        if stripped == "---":
            return fields
        if stripped.startswith("-") and current_key:
            existing = fields.get(current_key)
            if not isinstance(existing, list):
                existing = []
            existing.append(stripped[1:].strip())
            fields[current_key] = existing
            continue
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        current_key = key.strip()
        fields[current_key] = value.strip()
    raise ValueError("missing frontmatter closing fence")


def fail(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    raise SystemExit(1)


def is_excluded(path: Path) -> bool:
    return any(part in EXCLUDED_PARTS for part in path.parts)


def list_items(value: str | list[str]) -> set[str]:
    if isinstance(value, list):
        return set(value)
    return set(re.findall(r"[A-Za-z0-9_./-]+", value))


def resolve_reference(ref: str, source: Path, root: Path) -> Path | None:
    for candidate in (source.parent / ref, root / ref):
        if candidate.exists():
            return candidate
    matches = list(root.rglob(ref))
    return matches[0] if len(matches) == 1 else None


path = Path(sys.argv[1]) if len(sys.argv) > 1 else None
root = Path(sys.argv[2]) if len(sys.argv) > 2 else None
if not path or not path.exists():
    fail("usage: check_frontmatter.py <file> [root]")
if root and is_excluded(path.resolve().relative_to(root.resolve())):
    print(f"SKIP: {path}")
    raise SystemExit(0)

fields = parse_frontmatter(path.read_text(encoding="utf-8"))
missing = sorted(UNIVERSAL - fields.keys())
if missing:
    fail(f"missing required fields: {', '.join(missing)}")

for field_name in LIST_FIELDS:
    value = fields.get(field_name, "")
    if isinstance(value, list):
        continue
    if not (value.startswith("[") and value.endswith("]")):
        fail(f"{field_name} must use YAML list syntax")

kind_value = fields.get("kind", "")
kind_items = list_items(kind_value)
invalid_kinds = sorted(kind_items - ALLOWED_KINDS)
if invalid_kinds:
    fail(f"invalid kind values: {', '.join(invalid_kinds)}")

doc_type = fields.get("type", "")
if doc_type not in ALLOWED_TYPES:
    fail(f"invalid type: {doc_type}")

status = fields.get("status", "")

if doc_type == "decision" or "adr" in kind_items:
    adr_missing = sorted(ADR - fields.keys())
    if adr_missing:
        fail(f"missing ADR fields: {', '.join(adr_missing)}")
    if doc_type != "decision":
        fail("ADR docs must use 'type: decision'")
    if status and status not in {"draft", "accepted", "completed", "superseded"}:
        fail("ADR docs must use status 'draft', 'accepted', 'completed', or 'superseded'")

if "plan" in kind_items:
    plan_missing = sorted(PLAN - fields.keys())
    if plan_missing:
        fail(f"missing plan fields: {', '.join(plan_missing)}")
    if doc_type != "explanation":
        fail("plan docs must use 'type: explanation'")
    if status and status not in {"draft", "in-progress", "completed", "archived"}:
        fail("plan docs must use status 'draft', 'in-progress', 'completed', or 'archived'")
    if status in {"completed", "archived"} and not fields.get("replacedBy"):
        fail("completed and archived plan docs must set replacedBy once the accepted current spec exists")
    if status in {"completed", "archived"} and root and fields.get("replacedBy"):
        replacement = resolve_reference(str(fields["replacedBy"]), path, root)
        if not replacement:
            fail("replacedBy target could not be resolved from root")
        replacement_fields = parse_frontmatter(replacement.read_text(encoding="utf-8"))
        replacement_kind = list_items(replacement_fields.get("kind", ""))
        if replacement_fields.get("type") != "reference" or "spec" not in replacement_kind or replacement_fields.get("status") != "accepted":
            fail("replacedBy target must be an accepted spec when root is provided")

if "spec" in kind_items:
    if doc_type != "reference":
        fail("spec docs must use 'type: reference'")
    if status and status not in {"draft", "accepted"}:
        fail("spec docs must use status 'draft' or 'accepted'")

if "adr" in kind_items:
    for field_name in ("supersededBy", "supersedes"):
        value = fields.get(field_name, "")
        if value and "/" in value:
            fail(f"{field_name} must use ADR IDs, not paths")

print(f"OK: {path}")
