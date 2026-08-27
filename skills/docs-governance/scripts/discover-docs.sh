#!/usr/bin/env bash
set -euo pipefail

readonly default_extensions='md,mdx,adoc,rst,txt,mmd,puml'
readonly supported_extensions='md mdx adoc rst txt mmd puml yaml yml toml json'

usage() {
  cat <<'EOF'
Usage: discover-docs.sh <folder> [options]

Inventory text documentation and surface governance and metadata signals.
Output is written to stdout; redirect reports outside the audit folder or exclude
the report path. Requires Bash 4.3 or later. ripgrep (`rg`) is optional.

Arguments:
  <folder>  Repository or directory to audit (required).

Options:
  --ext     Replace the default extension list.
  --status  Filter by status (e.g. active, draft, completed, deprecated, superseded).
            Accepts a comma-separated list. Defaults to all.
  --exclude Exclude a path relative to <folder>. May be repeated. A literal
            directory excludes its tree; * and ? globs may span directories.
  -h, --help
            Show this help.

Defaults:
  --ext md,mdx,adoc,rst,txt,mmd,puml
  --status (all)

Supported extensions:
  md mdx adoc rst txt mmd puml yaml yml toml json

Built-in ignored directories:
  .git .hg .svn node_modules vendor build dist .dart_tool .gradle .cache
  .agents/skills .claude/skills .codex/skills

Examples:
  discover-docs.sh /path/to/repository
  discover-docs.sh /path/to/repository --status active
  discover-docs.sh /path/to/repository --status deprecated,superseded
  discover-docs.sh /path/to/repository --ext md,mdx
  discover-docs.sh /path/to/repository --exclude archive
  discover-docs.sh /path/to/repository \
    --exclude archive \
    --exclude 'docs/generated/**'
  discover-docs.sh /path/to/repository --ext yaml,yml,toml,json > /tmp/audit.txt
EOF
}

fail_usage() {
  echo "error: $1" >&2
  echo "usage: discover-docs.sh <folder> [options]" >&2
  exit 2
}

fail_scan() {
  echo "error: failed to audit $1" >&2
  exit 2
}

if ((BASH_VERSINFO[0] < 4 ||
      (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] < 3))); then
  echo 'error: Bash 4.3 or later is required' >&2
  exit 2
fi

if [[ "${1:-}" == '-h' || "${1:-}" == '--help' ]]; then
  usage
  exit 0
fi

[[ $# -gt 0 ]] || fail_usage 'audit folder is required'

target_root="$1"
shift
extension_csv="$default_extensions"
seen_ext=false
status_filter_csv=""
seen_status=false
custom_excludes=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ext)
      [[ "$seen_ext" == false ]] || fail_usage '--ext may be provided only once'
      [[ $# -gt 1 && "${2:-}" != -* ]] || fail_usage '--ext requires a value'
      extension_csv="$2"
      seen_ext=true
      shift 2
      ;;
    --status)
      [[ "$seen_status" == false ]] || fail_usage '--status may be provided only once'
      [[ $# -gt 1 && "${2:-}" != -* ]] || fail_usage '--status requires a value'
      status_filter_csv="${2,,}"
      seen_status=true
      shift 2
      ;;
    --exclude)
      [[ $# -gt 1 && "${2:-}" != -* ]] || \
        fail_usage '--exclude requires a value'
      exclude="${2#./}"
      exclude="${exclude%/}"
      custom_excludes+=("$exclude")
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail_usage "unknown argument: $1"
      ;;
  esac
done

[[ -d "$target_root" ]] || fail_usage "audit folder is not a directory: $target_root"
[[ "$extension_csv" =~ ^[a-z0-9]+(,[a-z0-9]+)*$ ]] || \
  fail_usage '--ext must be a comma-separated list without dots or spaces'

declare -a status_filters=()
if [[ -n "$status_filter_csv" && "$status_filter_csv" != "all" ]]; then
  [[ "$status_filter_csv" =~ ^[a-z0-9_-]+(,[a-z0-9_-]+)*$ ]] || \
    fail_usage '--status must be a comma-separated list of status names (e.g. active,draft)'
  IFS=',' read -r -a status_filters <<< "$status_filter_csv"
fi

for exclude in "${custom_excludes[@]}"; do
  [[ -n "$exclude" ]] || fail_usage '--exclude cannot be empty'
  [[ "$exclude" != /* && "$exclude" != '..' && "$exclude" != ../* &&
     "$exclude" != */../* && "$exclude" != */.. && "$exclude" != '!'* ]] || \
    fail_usage "--exclude must be a relative path glob: $exclude"
  [[ "$exclude" != *'['* && "$exclude" != *']'* &&
     "$exclude" != *'{'* && "$exclude" != *'}'* &&
     "$exclude" != *'\\'* ]] || \
    fail_usage "--exclude supports only literal paths, * and ?: $exclude"
done

IFS=',' read -r -a extensions <<< "$extension_csv"
declare -A seen_extensions=()
find_types=('(')

for extension in "${extensions[@]}"; do
  [[ " $supported_extensions " == *" $extension "* ]] || \
    fail_usage "unsupported extension: $extension"
  [[ -z "${seen_extensions[$extension]:-}" ]] || \
    fail_usage "duplicate extension: $extension"
  seen_extensions[$extension]=1
  [[ ${#find_types[@]} -eq 1 ]] || find_types+=(-o)
  find_types+=(-iname "*.${extension}")
done
find_types+=(')')

authority_pattern='single source of truth|source of truth|(^|[^[:alnum:]_])SSOT([^[:alnum:]_]|$)|(this |the )?(document|file|record|source|specification|reference) (is|remains) authoritative|authoritative (document|record|source|specification|reference)|(this |the )?(document|file|specification|reference) (is|remains) canonical|canonical (document|source|specification|reference)|conflict order|authority order|wins over|takes precedence|higher priority than|overrides? (older|previous|legacy) (document|documentation|descriptions?|rules?|specifications?)|supersed(es|ed|ing)|replaced by|replaces|deprecated;[[:space:]]*prefer|read[- ]only (document|file|directory|folder|reference|snapshot|material)|read[- ]only[^[:alnum:]]{0,3}(do not|must not) (edit|modify)|(this |the )?(file|document|documentation) (is|was) (auto[- ]?)?generated|(auto[- ]?)?generated (file|document|documentation)([^[:alnum:]_]|$)|(do not|must not) (edit|modify) (this|the|it)( directly)?|edit .+ and regenerate (this|the) (file|document|documentation)|edit .+ instead of (this|the) (file|document)|(this |the )?(file|document|documentation) (is|was) managed by|governed by|((rules?|contract|behavior|architecture|policy) (is|are) (defined|documented) (in|by))'
metadata_pattern='^(title|description|status|created|updated|kind|authority|ssot|supersedes|superseded[-_]?by|replaced[-_]?by)[[:space:]]*:'

find_prunes=(
  -name .git -o -name .hg -o -name .svn -o -name node_modules
  -o -name vendor -o -name build -o -name dist -o -name .dart_tool
  -o -name .gradle -o -name .cache -o -path '*/.agents/skills'
  -o -path '*/.claude/skills' -o -path '*/.codex/skills'
)

matches_exclude() {
  local path="$1"
  local pattern

  for pattern in "${custom_excludes[@]}"; do
    if [[ "$path" == $pattern ]]; then
      return 0
    fi
    if [[ "$pattern" != *'*'* && "$pattern" != *'?'* &&
          "$path" == "$pattern/"* ]]; then
      return 0
    fi
  done
  return 1
}

scan_file() {
  local pattern="$1"
  local file="$2"
  local result status

  if [[ "$use_rg" == true ]]; then
    set +e
    result="$(rg --no-messages --no-filename -n -i -- "$pattern" "$file")"
    status=$?
    set -e
  else
    set +e
    result="$(grep -IhnEi -- "$pattern" "$file")"
    status=$?
    set -e
  fi

  if [[ $status -eq 0 ]]; then
    printf '%s\n' "$result"
  elif [[ $status -ne 1 ]]; then
    return "$status"
  fi
}

append_matches() {
  local map_name="$1"
  local file="$2"
  local matches="$3"
  local -n findings_by_file="$map_name"
  local line content

  while IFS=: read -r line content; do
    [[ -n "$line" ]] || continue
    findings_by_file["$file"]+="${line} | ${content}"$'\n'
  done <<< "$matches"
}

remove_exact_matches() {
  local candidates="$1"
  local exclusions="$2"
  local candidate exclusion duplicate

  while IFS= read -r candidate; do
    [[ -n "$candidate" ]] || continue
    duplicate=false
    while IFS= read -r exclusion; do
      if [[ "$candidate" == "$exclusion" ]]; then
        duplicate=true
        break
      fi
    done <<< "$exclusions"
    [[ "$duplicate" == true ]] || printf '%s\n' "$candidate"
  done <<< "$candidates"
}

display_path() {
  local path="$1"
  path="${path//$'\n'/\\n}"
  path="${path//$'\r'/\\r}"
  printf '%s' "$path"
}

trim_value() {
  local value="$1"

  while [[ "$value" =~ ^[[:space:]] ]]; do
    value="${value:1}"
  done
  while [[ "$value" =~ [[:space:]]$ ]]; do
    value="${value:0:${#value}-1}"
  done
  printf '%s' "$value"
}

normalize_scalar() {
  local value

  value="$(trim_value "$1")"
  if [[ "$value" =~ ^\"(.*)\"$ || "$value" =~ ^\'(.*)\'$ ]]; then
    value="${BASH_REMATCH[1]}"
  elif [[ "$value" =~ ^(.*[^[:space:]])[[:space:]]+#.*$ ]]; then
    value="$(trim_value "${BASH_REMATCH[1]}")"
    if [[ "$value" =~ ^\"(.*)\"$ || "$value" =~ ^\'(.*)\'$ ]]; then
      value="${BASH_REMATCH[1]}"
    fi
  fi
  printf '%s' "$value"
}

add_contract_candidate() {
  contract_candidate_files+=("$1")
  contract_candidate_lines+=("$2")
  contract_candidate_fields+=("$3")
  contract_candidate_values+=("$4")
  contract_candidate_reasons+=("$5")
}

audit_frontmatter() {
  local file="$1"
  local lower_file="${file,,}"
  local base_file="${file##*/}"
  local lower_base="${base_file,,}"
  local line field value
  local line_number=1
  local closed=false
  local -a frontmatter_lines=()
  local -a frontmatter_line_numbers=()
  local -A seen_fields=()

  [[ "$lower_file" == *.md || "$lower_file" == *.mdx ]] || return 0

  {
    IFS= read -r line || return 0
    line="${line%$'\r'}"
    [[ "$line" == '---' ]] || return 0

    while IFS= read -r line || [[ -n "$line" ]]; do
      ((line_number += 1))
      line="${line%$'\r'}"
      if [[ "$line" == '---' ]]; then
        closed=true
        break
      fi
      frontmatter_lines+=("$line")
      frontmatter_line_numbers+=("$line_number")
    done
  } < "$file"

  [[ "$closed" == true ]] || return 0

  for index in "${!frontmatter_lines[@]}"; do
    line="${frontmatter_lines[$index]}"
    [[ "$line" =~ ^([a-zA-Z0-9_-]+)[[:space:]]*:[[:space:]]*(.*)$ ]] || continue
    field="${BASH_REMATCH[1],,}"
    value="$(normalize_scalar "${BASH_REMATCH[2]}")"
    line_number="${frontmatter_line_numbers[$index]}"

    if [[ -n "${seen_fields[$field]:-}" ]]; then
      add_contract_candidate "$file" "$line_number" "$field" "$value" \
        'Duplicate frontmatter field'
      continue
    fi
    seen_fields[$field]="$value"

    if [[ "$field" == status ]]; then
      if [[ "$value" != draft && "$value" != active &&
            "$value" != accepted && "$value" != proposed && "$value" != rejected &&
            "$value" != completed && "$value" != deprecated &&
            ! "$value" =~ ^deprecated\;[[:space:]]prefer[[:space:]].+ &&
            ! "$value" =~ ^superseded[[:space:]]by[[:space:]].+ ]]; then
        add_contract_candidate "$file" "$line_number" "$field" "$value" \
          'Not in preferred status taxonomy'
      fi
      continue
    fi
  done

  # Warn if frontmatter exists on a specification document but description is missing or blank
  if [[ "$lower_base" != readme.md && "$lower_base" != changelog.md &&
        "$lower_base" != contributing.md && "$lower_base" != license* ]]; then
    if [[ -z "${seen_fields[description]:-}" ]]; then
      add_contract_candidate "$file" "1" "description" "(missing)" \
        'Missing description in frontmatter'
    fi
  fi

  status_by_file["$file"]="${seen_fields[status]:-}"
}

matches_status_filter() {
  local file="$1"
  [[ ${#status_filters[@]} -eq 0 ]] && return 0

  local raw_status="${status_by_file[$file]:-}"
  [[ -n "$raw_status" ]] || return 1

  local base_status="$raw_status"
  if [[ "$raw_status" =~ ^deprecated\; ]]; then
    base_status="deprecated"
  elif [[ "$raw_status" =~ ^superseded[[:space:]]by ]]; then
    base_status="superseded"
  fi

  for filter in "${status_filters[@]}"; do
    if [[ "$filter" == "$raw_status" || "$filter" == "$base_status" ]]; then
      return 0
    fi
    # Map active <-> accepted and draft <-> proposed
    if [[ "$filter" == "active" && "$raw_status" == "accepted" ]]; then
      return 0
    fi
    if [[ "$filter" == "draft" && "$raw_status" == "proposed" ]]; then
      return 0
    fi
  done
  return 1
}

escape_table_cell() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//|/\\|}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  printf '%s' "$value"
}

if command -v rg >/dev/null 2>&1; then
  use_rg=true
else
  use_rg=false
fi

declare -A status_by_file=()

cd -- "$target_root"
resolved_root="$(pwd -P)"

if [[ "$use_rg" == false ]]; then
  echo '[!] missing `rg` tools.'
fi
echo "documentation inventory root: $resolved_root"
echo "extensions: $extension_csv"
if [[ -n "$status_filter_csv" && "$status_filter_csv" != "all" ]]; then
  echo "status filter: $status_filter_csv"
fi
if [[ ${#custom_excludes[@]} -gt 0 ]]; then
  printf 'custom excludes: %s\n' "${custom_excludes[*]}"
fi

candidate_files=()
while IFS= read -r -d '' file; do
  file="${file#./}"
  if [[ "$file" == *$'\n'* || "$file" == *$'\r'* ]]; then
    echo 'warning: skipped a path containing a newline or carriage return' >&2
    continue
  fi
  matches_exclude "$file" || candidate_files+=("$file")
done < <(find . -type d \( "${find_prunes[@]}" \) -prune -o \
  -type f "${find_types[@]}" -print0)

document_files=()
if [[ ${#candidate_files[@]} -gt 0 ]]; then
  mapfile -t document_files < <(
    printf '%s\n' "${candidate_files[@]}" | LC_ALL=C sort -u
  )
fi

declare -A governance_by_file=()
declare -A metadata_by_file=()
contract_candidate_files=()
contract_candidate_lines=()
contract_candidate_fields=()
contract_candidate_values=()
contract_candidate_reasons=()

# Discover status across all candidate files
for file in "${document_files[@]}"; do
  audit_frontmatter "$file"
done

# Apply status filter if specified
if [[ ${#status_filters[@]} -gt 0 ]]; then
  filtered_docs=()
  for file in "${document_files[@]}"; do
    if matches_status_filter "$file"; then
      filtered_docs+=("$file")
    fi
  done
  document_files=()
  if [[ ${#filtered_docs[@]} -gt 0 ]]; then
    document_files=("${filtered_docs[@]}")
  fi

  # Reset contract candidate findings to only reflect filtered files
  contract_candidate_files=()
  contract_candidate_lines=()
  contract_candidate_fields=()
  contract_candidate_values=()
  contract_candidate_reasons=()
  for file in "${document_files[@]}"; do
    audit_frontmatter "$file"
  done
fi

for file in "${document_files[@]}"; do
  governance_matches=''
  metadata_matches=''
  filtered_governance=''

  governance_matches="$(scan_file "$authority_pattern" "$file")" || fail_scan "$file"
  metadata_matches="$(scan_file "$metadata_pattern" "$file")" || fail_scan "$file"
  filtered_governance="$(
    remove_exact_matches "$governance_matches" "$metadata_matches"
  )"

  append_matches governance_by_file "$file" "$filtered_governance"
  append_matches metadata_by_file "$file" "$metadata_matches"
done

echo "documents: ${#document_files[@]}"
echo "files with governance signals: ${#governance_by_file[@]}"
echo "files with metadata signals: ${#metadata_by_file[@]}"
echo "preferred-contract candidates: ${#contract_candidate_files[@]}"

if [[ ${#contract_candidate_files[@]} -gt 0 ]]; then
  printf '\n## Preferred-contract candidates\n\n'
  printf '| File | Line | Field | Observed | Reason |\n'
  printf '| --- | ---: | --- | --- | --- |\n'
  for index in "${!contract_candidate_files[@]}"; do
    printf '| %s | %s | %s | %s | %s |\n' \
      "$(escape_table_cell "${contract_candidate_files[$index]}")" \
      "${contract_candidate_lines[$index]}" \
      "${contract_candidate_fields[$index]}" \
      "$(escape_table_cell "${contract_candidate_values[$index]}")" \
      "$(escape_table_cell "${contract_candidate_reasons[$index]}")"
  done
fi

printf '\n## Documentation files\n'
for file in "${document_files[@]}"; do
  display_path "$file"
  printf '\n'
done

printf '\n## Findings by file\n'
for file in "${document_files[@]}"; do
  governance="${governance_by_file[$file]-}"
  metadata="${metadata_by_file[$file]-}"
  [[ -n "$governance" || -n "$metadata" ]] || continue

  printf '\n### '
  display_path "$file"
  printf '\n'
  if [[ -n "$governance" ]]; then
    printf 'Governance:\n%s' "$governance"
  fi
  if [[ -n "$metadata" ]]; then
    printf 'Metadata:\n%s' "$metadata"
  fi
done
