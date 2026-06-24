# Flutter Taste Skill

> **Small details that build taste in Flutter.** Things users feel but can't explain.

A comprehensive skill for auditing Flutter apps against **29 design polish concerns** from [FlutterPro.design](https://flutterpro.design). Identifies missing polish, anti-patterns, and actionable improvements across 11 design categories.

---

## Quick Start

### 1. Understand the 11 Categories

The skill organizes Flutter design concerns into:

1. **Animations & Spring Physics** — Buttons, scrolls, transitions should feel alive
2. **Visual Feedback & Haptics** — Users need confirmation their actions registered
3. **Layout Stability & Spacing** — No jumpy content, clear visual hierarchy
4. **Text & Number Formatting** — Numbers readable, nulls never visible
5. **Input & Keyboard Handling** — Forms flow smoothly, keyboard doesn't block content
6. **Navigation & State Transition** — Routes and modals feel natural and responsive
7. **Platform Adaptation** — iOS feels iOS, Android feels Android
8. **Web-Specific Polish** — Web apps have proper titles, loaders, social previews
9. **Data & Null-Safety UX** — Graceful handling of loading, error, empty states
10. **Interaction Hints & Discoverability** — Hidden actions revealed, scrollables hint at motion
11. **Performance & Polish Details** — Icons precached, fonts loaded, notifications rescheduled

### 2. Audit Your App

**Option A: Quick Scan (30 min)**
- Identify which categories apply to your app
- Review concern tables in SKILL.md for each category
- Create a checklist of high-impact items

**Option B: Deep Audit (2–3 hours)**
- Run the audit script: `bash scripts/audit.sh /path/to/your/app`
- This greps your codebase and reports findings
- Review each category methodically
- Create prioritized fix list

**Option C: Code Review**
- Paste Flutter code snippet in conversation
- Skill identifies specific anti-patterns
- Provides corrected code with references

### 3. Implement Fixes

Each concern maps to implementation patterns in `references/`:

- **animations.md** — Spring physics, curves, transitions
- **feedback.md** — Haptics, visual feedback, selection colors
- **data.md** — Null handling, image sizing, number/date formatting, loading states
- **forms.md** — Input styling, autofocus, keyboard dismissal
- **navigation.md** — Bottom nav, sheets, back button, web titles, platform adaptation
- **platform.md** — File sharing, platform-native patterns

All patterns include:
- ❌ Anti-pattern (what NOT to do)
- ✅ Correct pattern (copy-paste ready)
- 🔍 Common pitfalls
- 📦 Package recommendations

### 4. Validate & Test

- Run audit script again to verify fixes
- Test on real iOS and Android devices (emulator may not show spring physics)
- Get feedback on feel before launch

---

## Use Cases

### Use Case 1: Pre-Launch Polish Check
"My app is feature-complete. What should I fix before submitting to app stores?"

→ Skill prioritizes concerns by impact (HIGH → MEDIUM → POLISH) and guides through each.

### Use Case 2: Code Review
"I just built a form/list/animation. Is this up to polish standards?"

→ Skill reviews code, identifies specific patterns, provides corrected versions.

### Use Case 3: Team Onboarding
"New dev joined and building features. What design polish checklist should they follow?"

→ Skill generates simplified checklist for code reviews, links to references for patterns.

### Use Case 4: Competitive Analysis
"How does my app compare to [competitor] on design feel?"

→ Skill suggests testing both apps against audit checklist systematically.

### Use Case 5: Fixing Specific Issues
"Users say my app feels rough/janky/unresponsive. Where do I start?"

→ Skill maps complaint to category and provides high-impact fixes.

---

## File Structure

```
flutter-taste/
├── SKILL.md                    # Main skill definition (audit framework, 11 categories, workflows)
├── TEST_CASES.md               # Test cases and usage examples
├── scripts/
│   └── audit.sh                # Bash script to grep codebase and generate report
├── references/                 # Implementation guides (one per category)
│   ├── animations.md           # Spring physics, curves, transitions
│   ├── feedback.md             # Haptics, visual feedback, selection colors
│   ├── data.md                 # Null safety, image sizing, formatting, states
│   ├── forms.md                # Input styling, autofocus, keyboard
│   ├── navigation.md           # Bottom nav, sheets, back button, web
│   └── platform.md             # File sharing, platform-native patterns
└── README.md                   # This file
```

### Core Files

**SKILL.md** (~600 lines)
- Full description of 11 categories
- Audit workflow (step-by-step)
- All 29 concerns with tables
- Common grep patterns
- References to files for detailed patterns

**references/*.md** (~300 lines each)
- Anti-pattern + correct pattern pairs
- Copy-paste ready code snippets
- Platform-specific notes
- Common pitfalls
- Package recommendations

**scripts/audit.sh**
- Greps codebase for each concern
- Generates report with findings
- Provides actionable next steps

---

## How to Use This Skill

### Conversation 1: Initial Audit
```
User: "Audit my Flutter app for design polish issues."

Claude: 
1. Asks what your app does (tabs? forms? images? web?)
2. Maps to relevant categories
3. Provides audit checklist
4. References specific patterns for your features
```

### Conversation 2: Code Review
```
User: [Pastes ListView with product cards]

Claude:
1. Identifies anti-patterns (unsized images, unformatted price, empty text fallback)
2. Shows corrected code with references
3. Links to data.md for NumberFormat and image sizing
```

### Conversation 3: Implementation Help
```
User: "How do I make button presses feel springy?"

Claude:
1. Explains spring physics concept
2. Provides SpringButton widget implementation
3. References animations.md for more patterns
```

### Conversation 4: Full Audit
```
User: "Run full audit on my app"

Claude:
1. Runs audit.sh (if code accessible)
2. Reports findings per category
3. Prioritizes high-impact items (null display, layout shifts)
4. Creates checklist with references for each fix
```

---

## Key Design Insights

### What Users Feel But Can't Explain

1. **Spring Physics** — Buttons that rebound feel responsive; linear scales feel dead
2. **Null Safety** — Seeing "null" or empty strings destroys trust; always provide fallbacks
3. **Layout Stability** — Images popping in = janky feel; reserved space = professional
4. **Platform Feel** — iOS sheets should be Cupertino; dates should be native pickers
5. **Keyboard Flow** — Single-field pages autofocus; scrolling dismisses keyboard
6. **Loading States** — Skeleton loaders prevent surprise jumps; empty states are clear
7. **Animation Curves** — `elasticOut` feels alive; `linear` feels artificial
8. **Formatting** — $1,000,000 reads faster than $1000000; "15 mins ago" > "2024-01-15T14:32:00"
9. **Feedback** — Users need haptics/visual confirmation on interactive elements
10. **Scrollability Hints** — Fade edges or scroll bars hint that content continues

### High-Impact Fixes (Most Noticeable to Users)

1. Remove all null displays (search `?? ''` → replace with meaningful fallback)
2. Reserve space for images (prevent layout shifts)
3. Format numbers for readability (intl package)
4. Handle loading/empty/error states explicitly
5. Add spring physics to buttons
6. Platform-adapt navigation (iOS sheets, Android back button)

---

## FAQ

**Q: Do I need to fix all 29 concerns?**
No. Prioritize high-impact first (null display, layout shifts, platform adaptation), then medium (animations, keyboard), then polish (icon caching). Most apps don't need everything.

**Q: How long does it take to audit?**
- Quick scan (relevant categories): 30 min
- Full audit (all categories): 2–3 hours
- Implementing fixes: Varies, but most are <30 min each

**Q: Should I use third-party packages?**
Some concerns require packages (intl for NumberFormat, timeago for relative dates). Most are built-in (spring physics, adaptive sheets, platform checks). See references for package recommendations.

**Q: Can I test locally?**
Yes! Run `bash scripts/audit.sh /path/to/flutter/app` to grep your codebase and generate a report.

**Q: Does this replace design reviews?**
No. This is a checklist for polish details users feel. Design reviews cover layout, color, typography, information hierarchy—different concerns.

---

## Next Steps

1. **Identify your app's categories** — What features does it have?
2. **Read relevant concern tables** — In SKILL.md, which apply?
3. **Check your code** — Use grep patterns or read concern descriptions
4. **Review patterns** — Links in SKILL.md point to references/
5. **Implement 1–2 fixes** — Start with high-impact
6. **Test on device** — iOS and Android both
7. **Iterate** — Small polish compounds into great feel

---

## References

- **FlutterPro.design** — Original source of all 29 concerns (https://flutterpro.design)
- **Flutter Docs** — Animation, cupertino, platform adaptation guides
- **Material Design 3** — Platform conventions and best practices
- **Human Interface Guidelines (iOS)** — Platform-specific expectations

---

## Contributing

This skill is based on [FlutterPro.design](https://flutterpro.design) by Kamran Bekirov. For new concerns or updates, check the site.

---

## License

This skill is provided as-is for learning and improving Flutter app polish.

---

## Support

If stuck:
1. Search references/ for your concern
2. Copy code pattern and adapt to your context
3. Test on real device (spring physics, animations feel different on emulator)
4. Ask Claude for code review or specific pattern implementation
