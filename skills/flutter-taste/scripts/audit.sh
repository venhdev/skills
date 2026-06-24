#!/bin/bash

# HEADER_START
# Flutter Taste Audit Script
# Scans a Flutter project for design polish concerns
#
# Usage: bash audit.sh [PROJECT_ROOT] [--json] [--output FILE] [--strict]
#   PROJECT_ROOT  Path to Flutter project (default: current directory)
#   --json        Output machine-readable JSON instead of human-readable text
#   --output FILE Write the report to FILE (default: stdout only — no file is written)
#   --strict      Exit with code 1 if any high-impact finding is present
#   --help        Show this help
#
# Exit codes:
#   0  Audit completed (use --strict to gate on findings)
#   1  lib/ directory not found, or --strict and findings present
#   2  Invalid arguments
# HEADER_END

set -e
# Note: not using `set -o pipefail` — the script has many grep pipelines that
# legitimately exit 1 (no matches). With pipefail, the `|| echo "0"` fallback
# would double-print the zero. Each grep pipeline below uses `|| true` to
# absorb the no-match case instead.

# ---- Argument parsing ----
PROJECT_ROOT="."
JSON_OUTPUT=0
STRICT=0
OUTPUT_FILE=""  # empty = print to stdout only (no file written)

while [[ $# -gt 0 ]]; do
  case "$1" in
    --json)            JSON_OUTPUT=1; shift ;;
    --strict)          STRICT=1; shift ;;
    --output|-o)
      if [[ -z "${2:-}" ]]; then
        echo "Error: --output requires a file path" >&2
        exit 2
      fi
      OUTPUT_FILE="$2"; shift 2 ;;
    --help|-h)
      # Print lines between HEADER_START and HEADER_END markers, stripping the
      # leading "# " comment prefix and the marker lines themselves.
      sed -n '/^# HEADER_START/,/^# HEADER_END/p' "$0" \
        | sed -e '/^# HEADER_/d' -e 's/^# \?//'
      exit 0
      ;;
    -*)
      echo "Unknown flag: $1" >&2
      exit 2
      ;;
    *)
      PROJECT_ROOT="$1"; shift ;;
  esac
done

# ---- Helpers ----
# Escape a string for safe inclusion in JSON (handle " and \)
json_escape() {
  local s="$1"
  s="${s//\\/\\\\}"
  s="${s//\"/\\\"}"
  printf '%s' "$s"
}

# Strip trailing /lib if user passed a lib path
PROJECT_ROOT="${PROJECT_ROOT%/lib}"

# ---- Pre-flight ----
if [ ! -d "$PROJECT_ROOT" ]; then
  if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{"error":"project directory not found","project":"%s"}\n' "$(json_escape "$PROJECT_ROOT")"
  else
    echo "❌ Project directory not found: $PROJECT_ROOT" >&2
  fi
  exit 1
fi

if [ ! -d "$PROJECT_ROOT/lib" ]; then
  if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{"error":"lib/ directory not found","project":"%s"}\n' "$(json_escape "$PROJECT_ROOT")"
  else
    echo "❌ No lib/ directory found at: $PROJECT_ROOT" >&2
    echo "   Run this from a Flutter project root, or pass the project path." >&2
  fi
  exit 1
fi

# ---- Run all checks (collect data first, format later) ----
gesture_count=$(grep -r "GestureDetector(" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
anim_count=$(grep -r "AnimationController" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")

# xargs -r: don't run grep if no input
custom_buttons=$(grep -rl "Container\|ClipRRect" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | xargs -r grep -l "onTap" 2>/dev/null | wc -l || echo "0")
selectable=$(grep -r "SelectableText(" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")

unsize_images=$(grep -r "Image\." "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | grep -v "width:" | grep -v "height:" | wc -l || echo "0")
safearea=$(grep -r "SafeArea(" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
# -A 6 catches padding that wraps to a few lines below
listview_padded=$(grep -r "ListView(" "$PROJECT_ROOT/lib" --include="*.dart" -A 6 2>/dev/null | grep "padding:" | wc -l || echo "0")

dynamic_numbers=$(grep -r "\$" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | grep -i "price\|amount\|count\|views" | grep -v "NumberFormat\|intl\|DecimalFormat" | wc -l || echo "0")
# Catch both single and double-quote empty fallbacks
unsafe_fallback_single=$(grep -r "?? ''" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
unsafe_fallback_double=$(grep -r '?? ""' "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
unsafe_fallback=$((unsafe_fallback_single + unsafe_fallback_double))
raw_datetime=$(grep -r "\.toString()" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | grep -i "date\|time\|created\|updated" | wc -l || echo "0")

textfield=$(grep -r "TextField(" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | grep -v "autofocus:" | wc -l || echo "0")
focus_nodes=$(grep -r "FocusNode" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | grep -v "dispose" | wc -l || echo "0")

bottom_nav=$(grep -r "BottomNavigationBar\|NavigationBar" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
modals=$(grep -r "showModalBottomSheet\|showDialog" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
custom_routes=$(grep -r "PageRoute\|MaterialPageRoute\|CupertinoPageRoute" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")

platform_checks=$(grep -r "Platform.is\|defaultTargetPlatform\|TargetPlatform" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")

# Web checks
if [ -f "$PROJECT_ROOT/web/index.html" ]; then
  web_dir_present=1
  # Look for the actual Title widget pattern, not a non-existent comment marker
  has_title=$(grep -E "Title\(\s*$|Title\(\s*title:" "$PROJECT_ROOT/web/index.html" 2>/dev/null | grep -v "AppBar\|Tooltip" | wc -l || echo "0")
  has_loader=$(grep -E "progress|spinner|loading" "$PROJECT_ROOT/web/index.html" 2>/dev/null | wc -l || echo "0")
  has_og=$(grep -c 'property="og:' "$PROJECT_ROOT/web/index.html" 2>/dev/null || echo "0")
else
  has_title=0
  has_loader=0
  has_og=0
  web_dir_present=0
fi

null_display=$(grep -r "\.toString()" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | grep -v "//" | wc -l || echo "0")
error_catch=$(grep -r "on Exception\|catch" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | grep -v "//" | wc -l || echo "0")

slidable=$(grep -r "Slidable\|swipeable" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")

precache=$(grep -r "precacheImage" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
google_fonts=$(grep -r "GoogleFonts\." "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
version_check=$(grep -r "PackageInfo\|versionCheck" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
tz_check=$(grep -r "timeZone\|timezone\|TimeZone" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")

# High-impact findings (gate --strict on these)
high_impact_count=$((unsafe_fallback + unsize_images + modals))

# ---- Output helper ----
# Print to stdout, or write to file if OUTPUT_FILE is set.
# When writing to file, also tee to stdout so the user sees progress.
print_or_save() {
  if [ -n "$OUTPUT_FILE" ]; then
    cat | tee "$OUTPUT_FILE"
  else
    cat
  fi
}

# ---- JSON output ----
if [ "$JSON_OUTPUT" -eq 1 ]; then
  json_project=$(json_escape "$PROJECT_ROOT")
  json_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")
  cat <<EOF
{
  "project": "$json_project",
  "generated": "$json_date",
  "summary": {
    "high_impact_findings": $high_impact_count
  },
  "checks": [
    {"category": "1_animations",      "label": "GestureDetector instances",       "count": $gesture_count,   "warning": "Review for spring physics on tap animations"},
    {"category": "1_animations",      "label": "AnimationController instances",   "count": $anim_count,      "warning": "Check curves: use elasticOut or spring physics instead of linear/easeInOut"},
    {"category": "2_feedback",        "label": "Custom buttons w/o feedback",     "count": $custom_buttons,  "warning": "Ensure custom tap targets provide visual feedback (scale, opacity, etc.)"},
    {"category": "2_feedback",        "label": "SelectableText instances",        "count": $selectable,      "warning": "Check if selection color matches design (custom selectionColor applied?)"},
    {"category": "3_layout",          "label": "Images possibly without size",    "count": $unsize_images,   "high_impact": true, "warning": "Add explicit width/height or AspectRatio to prevent layout shift"},
    {"category": "3_layout",          "label": "SafeArea instances",              "count": $safearea,        "warning": "Verify SafeArea is NOT wrapping scrollable widgets (ListView, SingleChildScrollView)"},
    {"category": "3_layout",          "label": "ListView with padding",           "count": $listview_padded, "warning": "If horizontal, check for clipping issues (should use UnconstrainedBox)"},
    {"category": "4_text",            "label": "Unformatted numbers",            "count": $dynamic_numbers, "warning": "Use NumberFormat('pattern') for human-readable numbers"},
    {"category": "4_text",            "label": "Unsafe null fallbacks (?? '' or ?? \"\")",   "count": $unsafe_fallback, "high_impact": true, "warning": "Use meaningful fallbacks: ?? 'Guest', ?? 'Unknown', etc."},
    {"category": "4_text",            "label": "Raw DateTime .toString()",        "count": $raw_datetime,    "warning": "Use DateFormat from intl package"},
    {"category": "5_forms",           "label": "TextField w/o autofocus",         "count": $textfield,       "warning": "Single-field pages should set autofocus: true"},
    {"category": "5_forms",           "label": "FocusNode uses",                  "count": $focus_nodes,     "warning": "Ensure all FocusNode instances are disposed in cleanup"},
    {"category": "6_navigation",      "label": "Bottom nav instances",            "count": $bottom_nav,      "warning": "Does tab re-tap scroll nested list to top? (Check for controller listener)"},
    {"category": "6_navigation",      "label": "Modal instances",                 "count": $modals,          "high_impact": true, "warning": "Check: iOS uses adaptive? Android back button handled? (PopScope + AndroidManifest android:enableOnBackInvokedCallback)"},
    {"category": "6_navigation",      "label": "Route instances",                 "count": $custom_routes,   "warning": "Use physics-based transitions or platform-aware routes"},
    {"category": "7_platform",        "label": "Platform adaptation points",      "count": $platform_checks, "warning": "Verify date pickers, buttons, sheets use platform-appropriate widgets"},
    {"category": "8_web",             "label": "Web directory present",           "count": $web_dir_present, "warning": "Web support not configured; web polish checks skipped"},
    {"category": "8_web",             "label": "Title widget in HTML",            "count": $has_title,       "warning": "Title widget not detected in web/index.html (expected Title( title: ... ) wrapper)"},
    {"category": "8_web",             "label": "Loading indicator",               "count": $has_loader,      "warning": "Loading indicator not configured in web/index.html"},
    {"category": "8_web",             "label": "Open Graph meta tags",            "count": $has_og,          "warning": "Open Graph meta tags not present in web/index.html"},
    {"category": "9_data",            "label": "Suspect .toString() on displays", "count": $null_display,    "warning": "Ensure nulls never reach display (use ?? with safe fallback)"},
    {"category": "9_data",            "label": "Error handlers",                  "count": $error_catch,     "warning": "Verify errors show user-friendly messages, not raw exception"},
    {"category": "10_interaction",    "label": "Slidable / swipeable instances",  "count": $slidable,        "warning": "Call .open() on load to hint users, or add chevron icon"},
    {"category": "11_performance",    "label": "precacheImage calls",             "count": $precache,        "warning": "Icon assets should be precached in initState or main()"},
    {"category": "11_performance",    "label": "GoogleFonts instances",           "count": $google_fonts,    "warning": "Precache fonts in main() or use downloaded .ttf files to prevent jank"},
    {"category": "11_performance",    "label": "Version check instances",         "count": $version_check,   "warning": "Show changelog dialog on first run of new version"},
    {"category": "11_performance",    "label": "Timezone references",             "count": $tz_check,        "warning": "If using notifications, reschedule on timezone change"}
  ]
}
EOF
  if [ "$STRICT" -eq 1 ] && [ "$high_impact_count" -gt 0 ]; then
    exit 1
  fi
  # If --output is set, write the JSON to that file too.
  if [ -n "$OUTPUT_FILE" ]; then
    cat <<EOF > "$OUTPUT_FILE"
{
  "project": "$json_project",
  "generated": "$json_date",
  "summary": {
    "high_impact_findings": $high_impact_count
  },
  "checks": [
    {"category": "1_animations",      "label": "GestureDetector instances",       "count": $gesture_count,   "warning": "Review for spring physics on tap animations"},
    {"category": "1_animations",      "label": "AnimationController instances",   "count": $anim_count,      "warning": "Check curves: use elasticOut or spring physics instead of linear/easeInOut"},
    {"category": "2_feedback",        "label": "Custom buttons w/o feedback",     "count": $custom_buttons,  "warning": "Ensure custom tap targets provide visual feedback (scale, opacity, etc.)"},
    {"category": "2_feedback",        "label": "SelectableText instances",        "count": $selectable,      "warning": "Check if selection color matches design (custom selectionColor applied?)"},
    {"category": "3_layout",          "label": "Images possibly without size",    "count": $unsize_images,   "high_impact": true, "warning": "Add explicit width/height or AspectRatio to prevent layout shift"},
    {"category": "3_layout",          "label": "SafeArea instances",              "count": $safearea,        "warning": "Verify SafeArea is NOT wrapping scrollable widgets (ListView, SingleChildScrollView)"},
    {"category": "3_layout",          "label": "ListView with padding",           "count": $listview_padded, "warning": "If horizontal, check for clipping issues (should use UnconstrainedBox)"},
    {"category": "4_text",            "label": "Unformatted numbers",            "count": $dynamic_numbers, "warning": "Use NumberFormat('pattern') for human-readable numbers"},
    {"category": "4_text",            "label": "Unsafe null fallbacks (?? '' or ?? \"\")",   "count": $unsafe_fallback, "high_impact": true, "warning": "Use meaningful fallbacks: ?? 'Guest', ?? 'Unknown', etc."},
    {"category": "4_text",            "label": "Raw DateTime .toString()",        "count": $raw_datetime,    "warning": "Use DateFormat from intl package"},
    {"category": "5_forms",           "label": "TextField w/o autofocus",         "count": $textfield,       "warning": "Single-field pages should set autofocus: true"},
    {"category": "5_forms",           "label": "FocusNode uses",                  "count": $focus_nodes,     "warning": "Ensure all FocusNode instances are disposed in cleanup"},
    {"category": "6_navigation",      "label": "Bottom nav instances",            "count": $bottom_nav,      "warning": "Does tab re-tap scroll nested list to top? (Check for controller listener)"},
    {"category": "6_navigation",      "label": "Modal instances",                 "count": $modals,          "high_impact": true, "warning": "Check: iOS uses adaptive? Android back button handled? (PopScope + AndroidManifest android:enableOnBackInvokedCallback)"},
    {"category": "6_navigation",      "label": "Route instances",                 "count": $custom_routes,   "warning": "Use physics-based transitions or platform-aware routes"},
    {"category": "7_platform",        "label": "Platform adaptation points",      "count": $platform_checks, "warning": "Verify date pickers, buttons, sheets use platform-appropriate widgets"},
    {"category": "8_web",             "label": "Web directory present",           "count": $web_dir_present, "warning": "Web support not configured; web polish checks skipped"},
    {"category": "8_web",             "label": "Title widget in HTML",            "count": $has_title,       "warning": "Title widget not detected in web/index.html (expected Title( title: ... ) wrapper)"},
    {"category": "8_web",             "label": "Loading indicator",               "count": $has_loader,      "warning": "Loading indicator not configured in web/index.html"},
    {"category": "8_web",             "label": "Open Graph meta tags",            "count": $has_og,          "warning": "Open Graph meta tags not present in web/index.html"},
    {"category": "9_data",            "label": "Suspect .toString() on displays", "count": $null_display,    "warning": "Ensure nulls never reach display (use ?? with safe fallback)"},
    {"category": "9_data",            "label": "Error handlers",                  "count": $error_catch,     "warning": "Verify errors show user-friendly messages, not raw exception"},
    {"category": "10_interaction",    "label": "Slidable / swipeable instances",  "count": $slidable,        "warning": "Call .open() on load to hint users, or add chevron icon"},
    {"category": "11_performance",    "label": "precacheImage calls",             "count": $precache,        "warning": "Icon assets should be precached in initState or main()"},
    {"category": "11_performance",    "label": "GoogleFonts instances",           "count": $google_fonts,    "warning": "Precache fonts in main() or use downloaded .ttf files to prevent jank"},
    {"category": "11_performance",    "label": "Version check instances",         "count": $version_check,   "warning": "Show changelog dialog on first run of new version"},
    {"category": "11_performance",    "label": "Timezone references",             "count": $tz_check,        "warning": "If using notifications, reschedule on timezone change"}
  ]
}
EOF
    echo "📄 JSON report saved to: $OUTPUT_FILE" >&2
  fi
  exit 0
fi

# ---- Text output (default) ----
output=$({
  echo "🔍 Flutter Taste Audit"
  echo "================================"
  echo "Scanning: $PROJECT_ROOT"
  echo ""
  if [ -n "$OUTPUT_FILE" ]; then
    echo "Output: $OUTPUT_FILE"
  else
    echo "Output: stdout (use --output FILE to save)"
  fi
  echo ""

  echo "Flutter Taste Audit Report"
  echo "Generated: $(date)"
  echo "Project: $PROJECT_ROOT"
  echo ""
  echo "================================"
  echo ""

  # Category 1: Animations & Spring Physics
  echo "1️⃣  ANIMATIONS & SPRING PHYSICS"
  echo "================================"
  echo ""
  echo "Checking for GestureDetector patterns..."
  echo "  Found: $gesture_count GestureDetector instances"
  echo "  ⚠️  Review each for spring physics on tap animations"
  echo ""

  echo "Checking for AnimationController usage..."
  echo "  Found: $anim_count AnimationController instances"
  echo "  ⚠️  Check curves—use elasticOut or spring physics instead of linear/easeInOut"
  echo ""

  # Category 2: Visual Feedback
  echo "2️⃣  VISUAL FEEDBACK & HAPTICS"
  echo "================================"
  echo ""
  echo "Checking for custom buttons without feedback..."
  echo "  Found: ~$custom_buttons potential custom buttons"
  echo "  ⚠️  Ensure custom tap targets provide visual feedback (scale, opacity, etc.)"
  echo ""

  echo "Checking for SelectableText..."
  echo "  Found: $selectable SelectableText instances"
  echo "  ⚠️  Check if selection color matches design (custom selectionColor applied?)"
  echo ""

  # Category 3: Layout & Spacing
  echo "3️⃣  LAYOUT STABILITY & SPACING"
  echo "================================"
  echo ""
  echo "Checking for Image widgets without explicit size..."
  echo "  Found: ~$unsize_images Image instances possibly without dimensions"
  echo "  ⚠️  Add explicit width/height or AspectRatio to prevent layout shift"
  echo ""

  echo "Checking for SafeArea usage..."
  echo "  Found: $safearea SafeArea instances"
  echo "  ⚠️  Verify SafeArea is NOT wrapping scrollable widgets (ListView, SingleChildScrollView)"
  echo ""

  echo "Checking for ListView with padding..."
  echo "  Found: ~$listview_padded ListView instances with padding"
  echo "  ⚠️  If horizontal, check for clipping issues (should use UnconstrainedBox)"
  echo ""

  # Category 4: Text & Numbers
  echo "4️⃣  TEXT & NUMBER FORMATTING"
  echo "================================"
  echo ""
  echo "Checking for potential unformatted numbers..."
  echo "  Found: ~$dynamic_numbers potential unformatted number displays"
  echo "  ⚠️  Use NumberFormat('pattern') for human-readable numbers"
  echo ""

  echo "Checking for null ?? '' patterns..."
  echo "  Found: $unsafe_fallback instances of ?? '' or ?? \"\" (empty string fallback)"
  echo "  ⚠️  Use meaningful fallbacks: ?? 'Guest', ?? 'Unknown', etc."
  echo ""

  echo "Checking for unformatted DateTime..."
  echo "  Found: ~$raw_datetime potential raw DateTime displays"
  echo "  ⚠️  Use DateFormat from intl package"
  echo ""

  # Category 5: Input & Keyboard
  echo "5️⃣  INPUT & KEYBOARD HANDLING"
  echo "================================"
  echo ""
  echo "Checking for TextField without autofocus..."
  echo "  Found: ~$textfield TextField instances without autofocus specified"
  echo "  ⚠️  Single-field pages should set autofocus: true"
  echo ""

  echo "Checking for FocusNode declarations..."
  echo "  Found: ~$focus_nodes FocusNode uses"
  echo "  ⚠️  Ensure all FocusNode instances are disposed in cleanup"
  echo ""

  # Category 6: Navigation
  echo "6️⃣  NAVIGATION & STATE TRANSITION"
  echo "================================"
  echo ""
  echo "Checking for bottom navigation patterns..."
  echo "  Found: $bottom_nav instances"
  echo "  ⚠️  Does tab re-tap scroll nested list to top? (Check for controller listener)"
  echo ""

  echo "Checking for modal patterns..."
  echo "  Found: $modals modal instances"
  echo "  ⚠️  Check: iOS uses adaptive? Android back button handled? (PopScope + AndroidManifest android:enableOnBackInvokedCallback)"
  echo ""

  echo "Checking for custom PageRoute implementations..."
  echo "  Found: $custom_routes route instances"
  echo "  ⚠️  Use physics-based transitions or platform-aware routes"
  echo ""

  # Category 7: Platform Adaptation
  echo "7️⃣  PLATFORM ADAPTATION"
  echo "================================"
  echo ""
  echo "Checking for Platform checks..."
  echo "  Found: $platform_checks platform adaptation points"
  echo "  ⚠️  Verify date pickers, buttons, sheets use platform-appropriate widgets"
  echo ""

  # Category 8: Web-Specific
  echo "8️⃣  WEB-SPECIFIC POLISH"
  echo "================================"
  echo ""
  if [ "$web_dir_present" -eq 1 ]; then
    echo "Checking web/index.html..."
    echo "  Title widget: $([[ $has_title -gt 0 ]] && echo '✅' || echo '❌')"
    echo "  Loading indicator: $([[ $has_loader -gt 0 ]] && echo '✅' || echo '❌')"
    echo "  Open Graph tags: $([[ $has_og -gt 0 ]] && echo '✅' || echo '❌')"
  else
    echo "  ⚠️  No web/ directory—web support not checked"
  fi
  echo ""

  # Category 9: Data & Null-Safety
  echo "9️⃣  DATA & NULL-SAFETY UX"
  echo "================================"
  echo ""
  echo "Checking for null display issues..."
  echo "  Found: ~$null_display .toString() calls"
  echo "  ⚠️  Ensure nulls never reach display (use ?? with safe fallback)"
  echo ""

  echo "Checking for error handling..."
  echo "  Found: $error_catch error handlers"
  echo "  ⚠️  Verify errors show user-friendly messages, not raw exception"
  echo ""

  # Category 10: Interaction Hints
  echo "1️⃣0️⃣  INTERACTION HINTS & DISCOVERABILITY"
  echo "================================"
  echo ""
  echo "Checking for Slidable usage (swipe actions)..."
  echo "  Found: $slidable swipe action instances"
  echo "  ⚠️  Call .open() on load to hint users, or add chevron icon"
  echo ""

  # Category 11: Performance
  echo "1️⃣1️⃣  PERFORMANCE & POLISH"
  echo "================================"
  echo ""
  echo "Checking for precacheImage calls..."
  echo "  Found: $precache precache instances"
  echo "  ⚠️  Icon assets should be precached in initState or main()"
  echo ""

  echo "Checking for GoogleFonts..."
  echo "  Found: $google_fonts GoogleFonts instances"
  echo "  ⚠️  Precache fonts in main() or use downloaded .ttf files to prevent jank"
  echo ""

  echo "Checking for version checking..."
  echo "  Found: $version_check version check instances"
  echo "  ⚠️  Show changelog dialog on first run of new version"
  echo ""

  echo "Checking for timezone handling..."
  echo "  Found: $tz_check timezone references"
  echo "  ⚠️  If using notifications, reschedule on timezone change"
  echo ""

  # Summary
  echo ""
  echo "================================"
  echo "✅ AUDIT COMPLETE"
  echo "================================"
  echo ""
  if [ "$high_impact_count" -gt 0 ]; then
    echo "⚠️  High-impact findings: $high_impact_count (unsafe fallbacks + unsized images + modals)"
    echo "    Address these first for maximum polish impact."
  else
    echo "✨ No high-impact findings detected."
  fi
  echo ""
  echo "Next steps:"
  echo "1. Review each category for concerns relevant to your app"
  echo "2. Prioritize HIGH-IMPACT: null display, layout shifts, nav bugs"
  echo "3. Then MEDIUM: animations, platform adaptation"
  echo "4. Finally POLISH: icon caching, GoogleFonts, notifications"
  echo ""
  echo "See references/ directory for implementation patterns."
})

# Emit: to stdout always; to file if --output is set
if [ -n "$OUTPUT_FILE" ]; then
  printf '%s\n' "$output" | tee "$OUTPUT_FILE"
  echo ""
  echo "📄 Report saved to: $OUTPUT_FILE" >&2
else
  printf '%s\n' "$output"
fi

if [ "$STRICT" -eq 1 ] && [ "$high_impact_count" -gt 0 ]; then
  exit 1
fi
exit 0
