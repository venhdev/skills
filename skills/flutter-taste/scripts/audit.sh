#!/bin/bash

# Flutter Taste Audit Script
# Scans a Flutter project for design polish concerns

set -e

PROJECT_ROOT="${1:-.}"
RESULTS_FILE="flutter_taste_report.txt"

echo "🔍 Flutter Taste Audit"
echo "================================"
echo "Scanning: $PROJECT_ROOT"
echo "Output: $RESULTS_FILE"
echo ""

{
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
  gesture_count=$(grep -r "GestureDetector(" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $gesture_count GestureDetector instances"
  echo "  ⚠️  Review each for spring physics on tap animations"
  echo ""

  echo "Checking for AnimationController usage..."
  anim_count=$(grep -r "AnimationController" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $anim_count AnimationController instances"
  echo "  ⚠️  Check curves—use elasticOut or spring physics instead of linear/easeInOut"
  echo ""

  # Category 2: Visual Feedback
  echo "2️⃣  VISUAL FEEDBACK & HAPTICS"
  echo "================================"
  echo ""

  echo "Checking for custom buttons without feedback..."
  custom_buttons=$(grep -r "Container\|ClipRRect" "$PROJECT_ROOT/lib" --include="*.dart" -l 2>/dev/null | xargs grep -l "onTap" 2>/dev/null | wc -l || echo "0")
  echo "  Found: ~$custom_buttons potential custom buttons"
  echo "  ⚠️  Ensure custom tap targets provide visual feedback (scale, opacity, etc.)"
  echo ""

  echo "Checking for SelectableText..."
  selectable=$(grep -r "SelectableText(" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $selectable SelectableText instances"
  echo "  ⚠️  Check if selection color matches design (custom selectionColor applied?)"
  echo ""

  # Category 3: Layout & Spacing
  echo "3️⃣  LAYOUT STABILITY & SPACING"
  echo "================================"
  echo ""

  echo "Checking for Image widgets without explicit size..."
  unsize_images=$(grep -r "Image\." "$PROJECT_ROOT/lib" --include="*.dart" | grep -v "width:" | grep -v "height:" | wc -l || echo "0")
  echo "  Found: ~$unsize_images Image instances possibly without dimensions"
  echo "  ⚠️  Add explicit width/height or AspectRatio to prevent layout shift"
  echo ""

  echo "Checking for SafeArea usage..."
  safearea=$(grep -r "SafeArea(" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $safearea SafeArea instances"
  echo "  ⚠️  Verify SafeArea is NOT wrapping scrollable widgets (ListView, SingleChildScrollView)"
  echo ""

  echo "Checking for ListView with padding..."
  listview_padded=$(grep -r "ListView(" "$PROJECT_ROOT/lib" --include="*.dart" -A 3 | grep "padding:" | wc -l || echo "0")
  echo "  Found: ~$listview_padded ListView instances with padding"
  echo "  ⚠️  If horizontal, check for clipping issues (should use UnconstrainedBox)"
  echo ""

  # Category 4: Text & Numbers
  echo "4️⃣  TEXT & NUMBER FORMATTING"
  echo "================================"
  echo ""

  echo "Checking for potential unformatted numbers..."
  dynamic_numbers=$(grep -r "\$" "$PROJECT_ROOT/lib" --include="*.dart" | grep -i "price\|amount\|count\|views" | grep -v "NumberFormat\|intl\|DecimalFormat" | wc -l || echo "0")
  echo "  Found: ~$dynamic_numbers potential unformatted number displays"
  echo "  ⚠️  Use NumberFormat('pattern') for human-readable numbers"
  echo ""

  echo "Checking for null ?? '' patterns..."
  unsafe_fallback=$(grep -r "?? ''" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $unsafe_fallback instances of ?? '' (empty string fallback)"
  echo "  ⚠️  Use meaningful fallbacks: ?? 'Guest', ?? 'Unknown', etc."
  echo ""

  echo "Checking for unformatted DateTime..."
  raw_datetime=$(grep -r "\.toString()" "$PROJECT_ROOT/lib" --include="*.dart" | grep -i "date\|time\|created\|updated" | wc -l || echo "0")
  echo "  Found: ~$raw_datetime potential raw DateTime displays"
  echo "  ⚠️  Use DateFormat from intl package"
  echo ""

  # Category 5: Input & Keyboard
  echo "5️⃣  INPUT & KEYBOARD HANDLING"
  echo "================================"
  echo ""

  echo "Checking for TextField without autofocus..."
  textfield=$(grep -r "TextField(" "$PROJECT_ROOT/lib" --include="*.dart" | grep -v "autofocus:" | wc -l || echo "0")
  echo "  Found: ~$textfield TextField instances without autofocus specified"
  echo "  ⚠️  Single-field pages should set autofocus: true"
  echo ""

  echo "Checking for FocusNode declarations..."
  focus_nodes=$(grep -r "FocusNode" "$PROJECT_ROOT/lib" --include="*.dart" | grep -v "dispose" | wc -l || echo "0")
  echo "  Found: ~$focus_nodes FocusNode uses"
  echo "  ⚠️  Ensure all FocusNode instances are disposed in cleanup"
  echo ""

  # Category 6: Navigation
  echo "6️⃣  NAVIGATION & STATE TRANSITION"
  echo "================================"
  echo ""

  echo "Checking for bottom navigation patterns..."
  bottom_nav=$(grep -r "BottomNavigationBar\|NavigationBar" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $bottom_nav instances"
  echo "  ⚠️  Does tab re-tap scroll nested list to top? (Check for controller listener)"
  echo ""

  echo "Checking for modal patterns..."
  modals=$(grep -r "showModalBottomSheet\|showDialog" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $modals modal instances"
  echo "  ⚠️  Check: iOS uses adaptive? Android back button handled? (WillPopScope/PopScope)"
  echo ""

  echo "Checking for custom PageRoute implementations..."
  custom_routes=$(grep -r "PageRoute\|MaterialPageRoute\|CupertinoPageRoute" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $custom_routes route instances"
  echo "  ⚠️  Use physics-based transitions or platform-aware routes"
  echo ""

  # Category 7: Platform Adaptation
  echo "7️⃣  PLATFORM ADAPTATION"
  echo "================================"
  echo ""

  echo "Checking for Platform checks..."
  platform_checks=$(grep -r "Platform.is\|defaultTargetPlatform\|TargetPlatform" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $platform_checks platform adaptation points"
  echo "  ⚠️  Verify date pickers, buttons, sheets use platform-appropriate widgets"
  echo ""

  # Category 8: Web-Specific
  echo "8️⃣  WEB-SPECIFIC POLISH"
  echo "================================"
  echo ""

  if [ -f "$PROJECT_ROOT/web/index.html" ]; then
    echo "Checking web/index.html..."
    has_title=$(grep -c "flutter_web_app_title" "$PROJECT_ROOT/web/index.html" 2>/dev/null || echo "0")
    has_loader=$(grep -c "loading" "$PROJECT_ROOT/web/index.html" 2>/dev/null || echo "0")
    has_og=$(grep -c "og:" "$PROJECT_ROOT/web/index.html" 2>/dev/null || echo "0")
    
    echo "  Title customization: $([[ $has_title -gt 0 ]] && echo '✅' || echo '❌')"
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
  null_display=$(grep -r "\.toString()" "$PROJECT_ROOT/lib" --include="*.dart" | grep -v "//\|// " | wc -l || echo "0")
  echo "  Found: ~$null_display .toString() calls"
  echo "  ⚠️  Ensure nulls never reach display (use ?? with safe fallback)"
  echo ""

  echo "Checking for error handling..."
  error_catch=$(grep -r "on Exception\|catch" "$PROJECT_ROOT/lib" --include="*.dart" | grep -v "//\|// " | wc -l || echo "0")
  echo "  Found: $error_catch error handlers"
  echo "  ⚠️  Verify errors show user-friendly messages, not raw exception"
  echo ""

  # Category 10: Interaction Hints
  echo "1️⃣0️⃣  INTERACTION HINTS & DISCOVERABILITY"
  echo "================================"
  echo ""

  echo "Checking for Slidable usage (swipe actions)..."
  slidable=$(grep -r "Slidable\|swipeable" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $slidable swipe action instances"
  echo "  ⚠️  Call .open() on load to hint users, or add chevron icon"
  echo ""

  # Category 11: Performance
  echo "1️⃣1️⃣  PERFORMANCE & POLISH"
  echo "================================"
  echo ""

  echo "Checking for precacheImage calls..."
  precache=$(grep -r "precacheImage" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $precache precache instances"
  echo "  ⚠️  Icon assets should be precached in initState or main()"
  echo ""

  echo "Checking for GoogleFonts..."
  google_fonts=$(grep -r "GoogleFonts\." "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $google_fonts GoogleFonts instances"
  echo "  ⚠️  Precache fonts in main() or use downloaded .ttf files to prevent jank"
  echo ""

  echo "Checking for version checking..."
  version_check=$(grep -r "PackageInfo\|versionCheck" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $version_check version check instances"
  echo "  ⚠️  Show changelog dialog on first run of new version"
  echo ""

  echo "Checking for timezone handling..."
  tz_check=$(grep -r "timeZone\|timezone\|TimeZone" "$PROJECT_ROOT/lib" --include="*.dart" 2>/dev/null | wc -l || echo "0")
  echo "  Found: $tz_check timezone references"
  echo "  ⚠️  If using notifications, reschedule on timezone change"
  echo ""

  # Summary
  echo ""
  echo "================================"
  echo "✅ AUDIT COMPLETE"
  echo "================================"
  echo ""
  echo "Next steps:"
  echo "1. Review each category for concerns relevant to your app"
  echo "2. Prioritize HIGH-IMPACT: null display, layout shifts, nav bugs"
  echo "3. Then MEDIUM: animations, platform adaptation"
  echo "4. Finally POLISH: icon caching, GoogleFonts, notifications"
  echo ""
  echo "See references/ directory for implementation patterns."

} | tee "$RESULTS_FILE"

echo ""
echo "📄 Report saved to: $RESULTS_FILE"
