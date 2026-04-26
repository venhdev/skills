#!/usr/bin/env python3
import re
import sys
from pathlib import Path

UNIVERSAL = {"title", "type", "kind", "audience", "owner", "created", "lastReviewed", "depends-on", "updates"}
ADR = {"adr-id", "status", "deciders", "decided", "supersededBy", "supersedes"}
PLAN = {"status"}


def parse_frontmatter(text: str) -> dict[str, str]:
    lines = text.splitlines()
    if len(lines) < 3 or lines[0].strip() != "---":
        raise ValueError("missing frontmatter opening fence")
    fields = {}
    for line in lines[1:]:
        if line.strip() == "---":
            return fields
        if ":" not in line:
            continue
        key, value = line.split(":", 1)
        fields[key.strip()] = value.strip()
    raise ValueError("missing frontmatter closing fence")


def fail(msg: str) -> None:
    print(f"ERROR: {msg}", file=sys.stderr)
    raise SystemExit(1)


path = Path(sys.argv[1]) if len(sys.argv) > 1 else None
if not path or not path.exists():
    fail("usage: check_frontmatter.py <file>")

fields = parse_frontmatter(path.read_text(encoding="utf-8"))
missing = sorted(UNIVERSAL - fields.keys())
if missing:
    fail(f"missing required fields: {', '.join(missing)}")

kind_value = fields.get("kind", "")
kind_items = set(re.findall(r"[A-Za-z0-9_-]+", kind_value))
doc_type = fields.get("type", "")

if doc_type == "decision" or "adr" in kind_items:
    adr_missing = sorted(ADR - fields.keys())
    if adr_missing:
        fail(f"missing ADR fields: {', '.join(adr_missing)}")
    if doc_type != "decision":
        fail("ADR docs must use 'type: decision'")

if "plan" in kind_items:
    plan_missing = sorted(PLAN - fields.keys())
    if plan_missing:
        fail(f"missing plan fields: {', '.join(plan_missing)}")
    review_cadence = fields.get("reviewCadence")
    if review_cadence and review_cadence != "90":
        fail("plan docs must use 'reviewCadence: 90' when reviewCadence is present")

if "adr" in kind_items:
    for field_name in ("supersededBy", "supersedes"):
        value = fields.get(field_name, "")
        if value and "/" in value:
            fail(f"{field_name} must use ADR IDs, not paths")

print(f"OK: {path}")
