#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: discover-toolchain.sh [folder] [--file <path.md>] [--dry-run]

Probe repository manifests, active runtime wrappers, canonical test/analysis
commands, installed CLI tools, and tracker authentication. Updates or creates
the `## Toolchain & Environment` section in the target Markdown file, ensures
the file is listed in `.gitignore`, and emits the section block to stdout.

Arguments:
  [folder]          Repository directory to inspect (default: current directory).

Options:
  --file <path.md>  Target Markdown file to create or update (must end in .md;
                    default: .agents/dev-environment.md inside [folder],
                    falling back to .agents/host-toolchain.md if present).
  --dry-run         Emit the section block to stdout without modifying files.
  -h, --help        Show this help.
EOF
}

fail_usage() {
  echo "error: $1" >&2
  echo "usage: discover-toolchain.sh [folder] [--file <path.md>] [--dry-run]" >&2
  exit 2
}

target_root=""
seen_root=false
file_arg=""
seen_file=false
dry_run=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --file)
      [[ "$seen_file" == false ]] || fail_usage '--file may be provided only once'
      [[ $# -gt 1 && -n "${2:-}" ]] || fail_usage '--file requires a .md file path'
      file_arg="$2"
      seen_file=true
      shift 2
      ;;
    --dry-run)
      dry_run=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      fail_usage "unknown option: $1"
      ;;
    *)
      [[ "$seen_root" == false ]] || fail_usage "unexpected argument: $1"
      target_root="$1"
      seen_root=true
      shift
      ;;
  esac
done

[[ -n "$target_root" ]] || target_root="."
[[ -d "$target_root" ]] || fail_usage "target folder is not a directory: $target_root"

if [[ -z "$file_arg" ]]; then
  if [[ -f "${target_root}/.agents/host-toolchain.md" && ! -f "${target_root}/.agents/dev-environment.md" ]]; then
    file_arg=".agents/host-toolchain.md"
  else
    file_arg=".agents/dev-environment.md"
  fi
fi

if [[ "${file_arg,,}" != *.md ]]; then
  fail_usage "--file must be a Markdown (.md) file: $file_arg"
fi

cd -- "$target_root"
resolved_root="$(pwd -P)"

if [[ "$file_arg" == /* ]]; then
  target_md="$file_arg"
else
  target_md="${resolved_root}/${file_arg#./}"
fi

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

join_csv() {
  [[ $# -gt 0 ]] || return 0
  local first="$1"
  shift
  printf '%s' "$first"
  local item
  for item in "$@"; do
    printf ', %s' "$item"
  done
}

collect_installed() {
  local -a found=()
  local cmd
  for cmd in "$@"; do
    case "$cmd" in
      fd)
        if has_cmd fd; then
          found+=("fd")
        elif has_cmd fdfind; then
          found+=("fdfind")
        fi
        ;;
      ast-grep)
        if has_cmd ast-grep; then
          found+=("ast-grep")
        elif has_cmd sg && sg --version 2>/dev/null | grep -qi 'ast-grep'; then
          found+=("sg(ast-grep)")
        fi
        ;;
      nvm)
        if has_cmd nvm || [[ -s "${NVM_DIR:-$HOME/.nvm}/nvm.sh" ]]; then
          found+=("nvm")
        fi
        ;;
      sdkman)
        if has_cmd sdk || [[ -s "${SDKMAN_DIR:-$HOME/.sdkman}/bin/sdkman-init.sh" ]]; then
          found+=("sdkman")
        fi
        ;;
      docker-compose)
        if has_cmd docker-compose; then
          found+=("docker-compose")
        elif has_cmd docker && docker compose version >/dev/null 2>&1; then
          found+=("docker-compose(plugin)")
        fi
        ;;
      *)
        if has_cmd "$cmd"; then
          found+=("$cmd")
        fi
        ;;
    esac
  done
  join_csv "${found[@]}"
}

find_prunes=(
  -name .git -o -name .hg -o -name .svn -o -name node_modules
  -o -name vendor -o -name build -o -name dist -o -name .dart_tool
  -o -name .gradle -o -name .cache -o -name Pods -o -name .fvm
  -o -path '*/.agents/skills' -o -path '*/.claude/skills' -o -path '*/.codex/skills'
)

# 1. Tracker CLI Auth Status (only installed CLIs)
tracker_auth=()
if has_cmd gh; then
  if gh auth status >/dev/null 2>&1; then
    tracker_auth+=("gh (authenticated)")
  else
    tracker_auth+=("gh (unauthenticated)")
  fi
fi
if has_cmd glab; then
  if glab auth status >/dev/null 2>&1; then
    tracker_auth+=("glab (authenticated)")
  else
    tracker_auth+=("glab (unauthenticated)")
  fi
fi

# 2. Infer Canonical Runtime Wrapper, Test Runner & Static Analysis
wrappers=()
test_cmds=()
lint_cmds=()

if [[ -f "mise.toml" || -f ".mise.toml" ]] && has_cmd mise; then
  wrappers+=("mise exec --")
elif [[ -f ".tool-versions" ]] && has_cmd asdf; then
  wrappers+=("asdf exec")
fi

if [[ -f "package.json" ]]; then
  js_runner="npm"
  if [[ -f "bun.lockb" || -f "bun.lock" ]] && has_cmd bun; then
    js_runner="bun"
    wrappers+=("bun")
  elif [[ -f "pnpm-lock.yaml" || -f "pnpm-workspace.yaml" ]] && has_cmd pnpm; then
    js_runner="pnpm"
    wrappers+=("pnpm")
  elif [[ -f "yarn.lock" ]] && has_cmd yarn; then
    js_runner="yarn"
    wrappers+=("yarn")
  fi
  if [[ -f "turbo.json" ]]; then
    wrappers+=("turbo (via ${js_runner})")
  fi
  test_cmds+=("${js_runner} test")
  lint_cmds+=("${js_runner} lint")
fi

pubspec_files=()
while IFS= read -r -d '' pf; do
  pubspec_files+=("${pf#./}")
done < <(find . -maxdepth 3 -type d \( "${find_prunes[@]}" \) -prune -o \
  -type f -name "pubspec.yaml" -print0 2>/dev/null | LC_ALL=C sort -z -u)

if [[ ${#pubspec_files[@]} -gt 0 ]]; then
  for pf in "${pubspec_files[@]}"; do
    pdir="$(dirname "$pf")"
    is_fl=false
    grep -Eq '^[[:space:]]*flutter:' "$pf" 2>/dev/null && is_fl=true || true

    d_prefix=""
    if [[ -f "${pdir}/.fvmrc" || -f ".fvmrc" || -d "${pdir}/.fvm" || -d ".fvm" ]] && has_cmd fvm; then
      [[ " ${wrappers[*]} " == *" fvm "* ]] || wrappers+=("fvm")
      d_prefix="fvm "
    elif has_cmd puro && [[ -d "${pdir}/.puro" || -d ".puro" ]]; then
      [[ " ${wrappers[*]} " == *" puro "* ]] || wrappers+=("puro")
      d_prefix="puro "
    fi

    scope_prefix=""
    [[ "$pdir" != "." ]] && scope_prefix="(in ${pdir}) "

    if [[ "$is_fl" == true ]]; then
      test_cmds+=("${scope_prefix}${d_prefix}flutter test")
      lint_cmds+=("${scope_prefix}${d_prefix}flutter analyze")
    else
      test_cmds+=("${scope_prefix}${d_prefix}dart test")
      lint_cmds+=("${scope_prefix}${d_prefix}dart analyze")
    fi
  done
fi

if [[ -f "pyproject.toml" || -f "requirements.txt" ]]; then
  py_prefix=""
  if [[ -f "uv.lock" ]] && has_cmd uv; then
    wrappers+=("uv run")
    py_prefix="uv run "
  elif [[ -f "poetry.lock" ]] && has_cmd poetry; then
    wrappers+=("poetry run")
    py_prefix="poetry run "
  fi
  test_cmds+=("${py_prefix}pytest")
  has_cmd ruff && lint_cmds+=("${py_prefix}ruff check") || true
fi

if [[ -f "Cargo.toml" ]]; then
  if has_cmd cargo-nextest; then
    test_cmds+=("cargo nextest run")
  else
    test_cmds+=("cargo test")
  fi
  lint_cmds+=("cargo clippy")
fi

if [[ -f "go.mod" ]]; then
  test_cmds+=("go test ./...")
  if has_cmd golangci-lint; then
    lint_cmds+=("golangci-lint run")
  else
    lint_cmds+=("go vet ./...")
  fi
fi

# 3. Collect Installed Host Tools by Category
env_managers="$(collect_installed mise asdf direnv nix proto fvm puro melos fnm volta nvm corepack uv uvx poetry pyenv pipenv conda rustup sdkman rbenv)"
runtimes_build="$(collect_installed flutter dart bun pnpm yarn npm node deno python3 pytest ruff pyright mypy cargo cargo-nextest rustc go golangci-lint zig cmake ninja gradle mvn java kotlin swift pod dotnet composer bundle ruby php)"
mobile_tools="$(collect_installed adb bundletool fastlane shorebird flutterfire fluttercraft scrcpy xcodebuild)"
task_runners="$(collect_installed just make task bazel bazelisk turbo nx)"
cli_utilities="$(collect_installed rg fd fzf ast-grep tree-sitter jq yq tree curl)"
db_and_infra="$(collect_installed git jj gh glab docker docker-compose podman kubectl psql sqlite3 redis-cli mongosh mysql)"

# 4. Build the Canonical Section Block (strictly non-empty lines)
section_lines=("## Toolchain & Environment" "")

append_kv() {
  local key="$1"
  local val="${2:-}"
  if [[ -n "$val" ]]; then
    section_lines+=("- **${key}**: ${val}")
  fi
}

append_kv "runtime_wrapper" "$(join_csv "${wrappers[@]}")"
append_kv "test_runner" "$(join_csv "${test_cmds[@]}")"
append_kv "static_analysis" "$(join_csv "${lint_cmds[@]}")"
append_kv "tracker_cli" "$(join_csv "${tracker_auth[@]}")"
append_kv "version_managers" "$env_managers"
append_kv "runtimes_and_build" "$runtimes_build"
append_kv "mobile_and_device" "$mobile_tools"
append_kv "task_runners" "$task_runners"
append_kv "search_ast_data" "$cli_utilities"
append_kv "vcs_containers_db" "$db_and_infra"

section_block="$(printf '%s\n' "${section_lines[@]}")"

ensure_gitignore_entry() {
  local entry="$1"
  local gitignore_path="${resolved_root}/.gitignore"

  if [[ ! -f "$gitignore_path" ]]; then
    printf '%s\n' "$entry" > "$gitignore_path"
    return
  fi

  if grep -Fqx "$entry" "$gitignore_path" || grep -Fqx "/$entry" "$gitignore_path"; then
    return
  fi

  if [[ -s "$gitignore_path" && -n "$(tail -c 1 "$gitignore_path")" ]]; then
    printf '\n%s\n' "$entry" >> "$gitignore_path"
  else
    printf '%s\n' "$entry" >> "$gitignore_path"
  fi
}

# 5. Create or Update Section in Target Markdown File & Ensure .gitignore
if [[ "$dry_run" == false ]]; then
  mkdir -p "$(dirname "$target_md")"

  if [[ ! -f "$target_md" ]]; then
    printf '%s\n' "$section_block" > "$target_md"
  elif grep -Eq '^## ([0-9]+\.[[:space:]]+)?Toolchain & Environment[[:space:]]*$' "$target_md"; then
    BLOCK="$section_block" awk '
      BEGIN {
        block = ENVIRON["BLOCK"]
        in_section = 0
        replaced = 0
      }
      /^## ([0-9]+\.[[:space:]]+)?Toolchain & Environment[[:space:]]*$/ {
        if (!replaced) {
          print block
          replaced = 1
        }
        in_section = 1
        next
      }
      /^## / {
        if (in_section) {
          print ""
          in_section = 0
        }
      }
      !in_section { print }
    ' "$target_md" > "${target_md}.tmp"
    mv "${target_md}.tmp" "$target_md"
  else
    if [[ -s "$target_md" ]]; then
      if [[ -n "$(tail -n 1 "$target_md")" ]]; then
        printf '\n%s\n' "$section_block" >> "$target_md"
      else
        printf '%s\n' "$section_block" >> "$target_md"
      fi
    else
      printf '%s\n' "$section_block" > "$target_md"
    fi
  fi

  if [[ "$target_md" == "${resolved_root}/"* ]]; then
    rel_target="${target_md#"${resolved_root}/"}"
    ensure_gitignore_entry "$rel_target"
  fi
fi

# 6. Output Section Block to Console
printf '%s\n' "$section_block"
