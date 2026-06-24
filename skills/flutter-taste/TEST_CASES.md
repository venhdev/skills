# Flutter Taste Skill — Test Cases

## Test Case 1: Quick Audit of E-Commerce App

**User Query:**
"Audit my Flutter e-commerce app for design polish. I have product cards with images, a shopping cart bottom sheet, and price displays."

**Expected Skill Output:**
1. Identifies relevant categories: Layout (Category 3️⃣), Text (Category 4️⃣), Navigation (Category 6️⃣)
2. Provides specific audit concerns:
   - Product images: Reserve space to prevent layout shift (AspectRatio + SizedBox)
   - Prices: Format with NumberFormat for readability (\$1,234.56 vs \$1234.56)
   - Bottom sheet: Check if iOS 13 style (should use showAdaptiveBottomSheet)
3. Suggests grep commands to scan code
4. References implementation patterns in references/layout.md and references/data.md

**Follow-up Actions:**
- User implements fixes based on patterns
- Runs audit script: `bash scripts/audit.sh /path/to/flutter_app`
- Reports findings and iterates

---

## Test Case 2: New Developer Building Social App

**User Query:**
"I'm building a social media app with posts, comments, infinite scroll, and like animations. What design details am I missing?"

**Expected Skill Output:**
1. Maps to categories: Animations (1️⃣), Feedback (2️⃣), Layout (3️⃣), Data (9️⃣), Interaction (1️⃣0️⃣)
2. Provides checklist:
   - Like button: Spring physics animation (reference: animations.md)
   - Post images: Skeleton loader + space reservation (reference: data.md)
   - No internet: Show graceful error state (reference: data.md)
   - Null author/caption: Use fallbacks, never show "null" (reference: data.md)
   - Infinite scroll: Hint scrollability with fade edges (reference: interactive.md—not created yet, but mentioned in SKILL.md)
   - Comments textarea: Dismiss keyboard on scroll (reference: forms.md)
3. Suggests command: `grep -r "Image\." lib/ | grep -v "width:"` to find unsized images

**Follow-up Actions:**
- Developer implements each concern incrementally
- Uses audit.sh to validate
- Tests on real devices for spring physics feel

---

## Test Case 3: Improving App Before Launch

**User Query:**
"My app is feature-complete but feels rough. Can you audit for all 29 design concerns and prioritize what to fix?"

**Expected Skill Output:**
1. Full audit across all 11 categories
2. Priority breakdown:
   - **HIGH (fix before launch):**
     - Category 9️⃣: No null display bugs
     - Category 3️⃣: No layout shifts on images
     - Category 6️⃣: Platform-aware navigation
   - **MEDIUM (nice to have):**
     - Category 1️⃣: Spring animations on buttons
     - Category 5️⃣: Keyboard flows smoothly
     - Category 7️⃣: iOS/Android platform adaptation
   - **POLISH (post-launch):**
     - Category 1️⃣1️⃣: Icon precaching, GoogleFonts caching
3. Guides through each using references/

**Follow-up Actions:**
- User runs `bash scripts/audit.sh`
- Addresses high-impact issues first
- Tests on both iOS and Android before app store submission

---

## Test Case 4: Code Review Conversation

**User Query:**
"I just wrote a ListView with product cards and prices. Can you check if I'm missing any design polish?"

**User provides code snippet:**
```dart
ListView.builder(
  itemCount: products.length,
  itemBuilder: (context, index) {
    final product = products[index];
    return Card(
      child: Column(
        children: [
          Image.network(product.imageUrl),  // ❌ No sizing
          Text(product.name ?? ''),          // ❌ Empty fallback
          Text('\$${product.price}'),        // ❌ Unformatted
        ],
      ),
    );
  },
)
```

**Expected Skill Output:**
1. Identifies 3 concerns:
   - **Category 3️⃣ (Layout):** Image has no explicit dimensions → layout shift
   - **Category 4️⃣ (Text):** Price not formatted → hard to read large numbers
   - **Category 9️⃣ (Data):** Name fallback is empty string → confusing
2. Provides corrected code with references:
   ```dart
   ListView.builder(
     itemCount: products.length,
     itemBuilder: (context, index) {
       final product = products[index];
       return Card(
         child: Column(
           children: [
             AspectRatio(  // ✅ Reserved space
               aspectRatio: 1.0,
               child: Image.network(product.imageUrl, fit: BoxFit.cover),
             ),
             Text(product.name ?? 'Unnamed Product'),  // ✅ Safe fallback
             Text(priceFormat.format(product.price)), // ✅ Formatted
           ],
         ),
       );
     },
   )
   ```
3. Points to references/data.md for NumberFormat and references/layout.md for image sizing

**Follow-up Actions:**
- User implements pattern
- Continues conversation for other concerns (animations, feedback, etc.)

---

## Test Case 5: Web App Audit

**User Query:**
"I built a Flutter web app. What web-specific polish am I missing?"

**Expected Skill Output:**
1. Identifies Category 8️⃣ (Web-specific):
   - Browser tab title: Does it change per route? (should use SystemChrome)
   - Loading progress: Does blank screen show progress? (customize index.html)
   - Open Graph: Can users preview link on social media? (add meta tags)
2. Checks web/index.html for:
   - Custom `<title>` (not just "flutter_web_app")
   - Loading spinner or progress indicator
   - `<meta property="og:..."` tags
3. Provides index.html template snippet
4. Points to references for implementation

**Follow-up Actions:**
- User updates web/index.html
- Sets up route-aware title updates
- Tests in browser and on social media share previews

---

## Test Case 6: Platform Adaptation Issues

**User Query:**
"My app looks wrong on iOS. Date picker and bottom sheet feel Android-ish. How do I fix?"

**Expected Skill Output:**
1. Identifies Category 7️⃣ (Platform Adaptation):
   - Date picker: Should use CupertinoDatePicker on iOS (not Material DatePicker)
   - Bottom sheet: Should use CupertinoModalPopupRoute or adaptive version
2. Provides patterns for both
3. Grep commands to find hardcoded Material components
4. Points to references/navigation.md (or new references/platform.md)

**Follow-up Actions:**
- User wraps widgets with Platform.isIOS checks
- Tests on both iOS and Android devices
- Iterates on feel

---

## How Skill Gets Triggered

The skill should trigger when user:
- **Explicitly asks:** "Audit my Flutter app," "Check for design polish," "What am I missing?"
- **Provides code snippet:** "Is this form good?" (skill spots keyboard handling, input styling)
- **Mentions specific features:** "I have a bottom sheet," "Product images keep shifting" (skill maps to categories)
- **Wants improvement:** "My app feels rough," "Before launch checklist" (skill provides prioritized audit)

---

## Expected Behavior

1. **Narrow scope:** If user asks about single component (button), skill focuses on relevant categories (1️⃣–2️⃣)
2. **Broad scope:** If user asks for full audit, skill covers all 11 categories
3. **Code-aware:** When given snippets, skill identifies specific anti-patterns and provides corrected code
4. **Reference-linked:** All advice includes pointers to references/ directory for detailed patterns
5. **Actionable:** Outputs grep commands, checklist items, and copy-paste-ready code

---

## Variations & Extensions

### Variant A: Competitive Audit
"How does my app compare to [competitor app] on design polish?"
→ Skill could suggest testing both apps against audit checklist

### Variant B: Team Review Checklist
"Create a design polish checklist my team can use for code reviews"
→ Skill provides simplified checklist (high-impact items only)

### Variant C: Specific Feature Deep Dive
"I'm building a complex form with 10 fields. What design details matter?"
→ Skill focuses heavily on Category 5️⃣ (forms) with detailed patterns

---

## Success Metrics

- ✅ User runs audit script and finds specific issues
- ✅ User implements 1+ patterns from references/
- ✅ User reports improved feel after fixes
- ✅ App receives positive feedback on polish before/after audit
