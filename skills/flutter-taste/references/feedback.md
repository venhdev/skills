# Visual Feedback & Haptics

> **Key principle:** Every interaction should confirm the user's intent. Users feel haptic feedback + visual response subconsciously.

## Pattern 1: Haptic Feedback Types & Timing

### The 3 Main Haptic Types

| Type | Pattern | Use Cases | Strength |
|------|---------|-----------|----------|
| **Selection** | Subtle tick | List item selection, toggle switch | Light |
| **Impact Light** | Short tap | Button press feedback, checkbox | Light |
| **Impact Medium** | Stronger tap | Form validation success, confirm | Medium |
| **Impact Heavy** | Strong thud | Error, destructive action, alert | Heavy |
| **Success** | Specific pattern | Successful operation complete | Medium |
| **Warning** | Double tap pattern | Validation error, wrong input | Medium |

> Flutter's built-in `HapticFeedback` only exposes a few generic types. For semantic types (`success`, `warning`, `error`, `light`, `medium`, `heavy`, `selection`, `rigid`, `soft`) prefer the `haptic_feedback` package's `Haptics.vibrate(HapticsType)`, guarded by `Haptics.canVibrate()`.

### ❌ Anti-Pattern: No haptic feedback
```dart
ElevatedButton(
  onPressed: _toggleLike,
  child: Text('Like'),
  // Silent; user unsure if tap registered
)
```

### ✅ Pattern: Context-appropriate haptics
```dart
import 'package:flutter/services.dart';

// Light selection (list items, checkboxes)
ElevatedButton(
  onPressed: () async {
    await HapticFeedback.selectionClick();
    _toggleCheckbox();
  },
  child: Text('Done'),
)

// Medium impact (form submission, delete)
ElevatedButton(
  onPressed: () async {
    await HapticFeedback.mediumImpact();
    _deleteItem();
  },
  child: Text('Delete'),
)

// Heavy impact (errors, critical actions)
ElevatedButton(
  onPressed: () async {
    if (!_validateForm()) {
      await HapticFeedback.heavyImpact();
      _showError();
    }
  },
  child: Text('Submit'),
)

// Success feedback
Future<void> _confirmAction() async {
  await HapticFeedback.heavyImpact();
  // Or custom pattern
  await HapticFeedback.vibrate(duration: 100);
}
```

### Haptic Timing Examples

```dart
// ✅ Like button with delayed haptic
class LikeButton extends StatefulWidget {
  @override
  State<LikeButton> createState() => _LikeButtonState();
}

class _LikeButtonState extends State<LikeButton> {
  bool _isLiked = false;

  Future<void> _toggleLike() async {
    setState(() => _isLiked = !_isLiked);
    
    if (_isLiked) {
      // Selection click for positive action
      await HapticFeedback.selectionClick();
    } else {
      // Light impact for removal
      await HapticFeedback.lightImpact();
    }
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: _toggleLike,
      child: Icon(
        _isLiked ? Icons.favorite : Icons.favorite_border,
        color: _isLiked ? Colors.red : Colors.grey,
      ),
    );
  }
}
```

---

## Pattern 2: Disable Material Tap Effects (Splash)

### ❌ Anti-Pattern: Material ripple on non-Material buttons
```dart
// ❌ Unwanted splash effect on custom button
GestureDetector(
  onTap: _handleTap,
  child: Container(
    // Material splash appears (looks jarring)
    child: Text('Custom Button'),
  ),
)

// ❌ Material button with unwanted feedback
MaterialButton(
  onPressed: _handleTap,
  // Splashes by default
  child: Text('Button'),
)
```

### ✅ Pattern: Disable splash when unwanted
```dart
// Option 1: Disable splash on Material button
MaterialButton(
  onPressed: _handleTap,
  splashColor: Colors.transparent,
  highlightColor: Colors.transparent,
  child: Text('Button'),
)

// Option 2: Use IconButton with custom configuration
IconButton(
  onPressed: _handleTap,
  icon: Icon(Icons.favorite),
  splashRadius: 24, // Control splash size
  // Or disable entirely
  splashColor: Colors.transparent,
  highlightColor: Colors.transparent,
)

// Option 3: Cupertino alternative (no splash)
CupertinoButton(
  onPressed: _handleTap,
  child: Text('Button'),
  // No splash effect; native iOS feel
)

// Option 4: Custom button without Material wrapper
GestureDetector(
  onTap: _handleTap,
  child: Container(
    padding: EdgeInsets.all(12),
    decoration: BoxDecoration(
      color: Colors.blue,
      borderRadius: BorderRadius.circular(8),
    ),
    child: Text('Custom'),
    // No Material; no splash
  ),
)
```

### Best Practice: Create reusable custom button
```dart
class CustomButton extends StatefulWidget {
  final VoidCallback onPressed;
  final Widget child;
  final bool haptic;

  const CustomButton({
    required this.onPressed,
    required this.child,
    this.haptic = true,
  });

  @override
  State<CustomButton> createState() => _CustomButtonState();
}

class _CustomButtonState extends State<CustomButton> {
  Future<void> _onTap() async {
    if (widget.haptic) {
      await HapticFeedback.selectionClick();
    }
    widget.onPressed();
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: _onTap,
      child: Container(
        padding: EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        decoration: BoxDecoration(
          color: Colors.blue,
          borderRadius: BorderRadius.circular(8),
        ),
        child: widget.child,
      ),
    );
  }
}

// Usage: No splash, haptic feedback on tap
CustomButton(
  onPressed: _handleTap,
  child: Text('Press Me'),
  haptic: true,
)
```

### ✅ Pattern: Disable splash globally via `ThemeData` (preferred)
Toggling splash per-widget works but doesn't scale. Apply once at the theme level — `NoSplash.splashFactory` removes ripple/highlight, `WidgetStateProperty.all(Colors.transparent)` clears overlay on the rest.
```dart
import 'package:flutter/material.dart';

final ButtonStyle noSplash = ButtonStyle(
  overlayColor: WidgetStateProperty.all(Colors.transparent),
  splashFactory: NoSplash.splashFactory,
);

ThemeData(
  // Global tap effects
  splashColor: Colors.transparent,
  splashFactory: NoSplash.splashFactory,
  highlightColor: Colors.transparent,
  hoverColor: Colors.transparent,
  focusColor: Colors.transparent,

  // Per-widget themes
  iconButtonTheme: IconButtonThemeData(style: noSplash),
  floatingActionButtonTheme: FloatingActionButtonThemeData(splashColor: Colors.transparent),
  textButtonTheme: TextButtonThemeData(style: noSplash),
  elevatedButtonTheme: ElevatedButtonThemeData(style: noSplash),
  outlinedButtonTheme: OutlinedButtonThemeData(style: noSplash),
  filledButtonTheme: FilledButtonThemeData(style: noSplash),
  navigationBarTheme: NavigationBarThemeData(overlayColor: WidgetStateProperty.all(Colors.transparent)),
  tabBarTheme: TabBarThemeData(
    overlayColor: WidgetStateProperty.all(Colors.transparent),
    splashFactory: NoSplash.splashFactory,
  ),
  checkboxTheme: CheckboxThemeData(overlayColor: WidgetStateProperty.all(Colors.transparent), splashRadius: 0),
  radioTheme: RadioThemeData(overlayColor: WidgetStateProperty.all(Colors.transparent), splashRadius: 0),
  switchTheme: SwitchThemeData(overlayColor: WidgetStateProperty.all(Colors.transparent)),
  sliderTheme: SliderThemeData(overlayColor: Colors.transparent, overlayShape: SliderComponentShape.noOverlay),
  menuButtonTheme: MenuButtonThemeData(style: noSplash),
  segmentedButtonTheme: SegmentedButtonThemeData(style: noSplash),
  toggleButtonsTheme: ToggleButtonsThemeData(splashColor: Colors.transparent, highlightColor: Colors.transparent),
  searchBarTheme: SearchBarThemeData(overlayColor: WidgetStateProperty.all(Colors.transparent)),
)
```

If a property collides with your existing theme, merge the value into your theme entry — don't duplicate the widget theme.

---

## Pattern 3: Customize Text Selection Color

`MaterialApp` picks the cursor and selection colors for you. If you have a custom design, pick them yourself — match your brand color.

### ❌ Anti-Pattern: Default Material selection (blue highlight)
```dart
Text('This text is selectable')
// Selection color is Material blue (doesn't match design)

TextField(
  decoration: InputDecoration(labelText: 'Enter name'),
  // Default selection color doesn't match brand
)
```

### ✅ Pattern: Set `textSelectionTheme` on `ThemeData`
Applies globally to `TextField`, `SelectableText`, and anything else using text selection.

```dart
ThemeData(
  textSelectionTheme: TextSelectionThemeData(
    selectionColor: AppColors.brand.withValues(alpha: 0.3),
    selectionHandleColor: AppColors.brand,
    cursorColor: AppColors.brand,
  ),
)
```

> Use `withValues(alpha:)` (not deprecated `withOpacity`). Selection sits behind text, so contrast matters — light brand colors usually work as-is, dark ones need opacity to read cleanly. Cursor and handles can stay at full color (they're small).

---

## Testing Checklist

- [ ] Haptic feedback on button press (feel device vibration)
- [ ] Selection feedback matches positive action (light) vs negative (heavy)
- [ ] Form submission triggers medium impact
- [ ] Errors trigger heavy impact
- [ ] Material splash disabled on custom buttons
- [ ] Text selection color matches brand/theme
- [ ] Haptics work on iOS AND Android (test on device, not emulator)
- [ ] No haptic on rapid taps (catches double-taps)

---

## Platform Notes

**iOS:**
- Haptics are smooth, immediate
- Users expect haptic on interactive elements
- Custom patterns less supported (use standard types)

**Android:**
- Haptics vary by device (some old devices don't support)
- Fallback gracefully if unsupported
- More flexible custom patterns

**Testing:**
```dart
// Check if haptics supported before use
Future<void> safeHaptic() async {
  try {
    await HapticFeedback.mediumImpact();
  } catch (e) {
    // Device doesn't support; continue silently
  }
}
```

---

## Package Recommendations

| Package | Use |
|---------|-----|
| (Built-in) | `HapticFeedback` for the 4 generic impacts; TextSelection theming |
| `haptic_feedback` | Typed `HapticsType` enum (`success`/`warning`/`error`/`light`/`medium`/`heavy`/`selection`/`rigid`/`soft`); use `Haptics.canVibrate()` to guard |
| `vibration` | Custom haptic patterns (if needed) |

Prefer `haptic_feedback` over built-in when you need semantic types beyond generic impacts.

---

## Common Pitfalls

### ❌ Haptic on every interaction
```dart
onTap: () async {
  await HapticFeedback.selectionClick();
  // Called on hover, drag, scroll... too much!
}
```

### ✅ Strategic haptic only
```dart
onTap: () async {
  await HapticFeedback.selectionClick();  // On intentional tap only
}
onHover: (hovering) {
  // No haptic here; too noisy
}
```

### ❌ Ignoring unsupported platforms
```dart
await HapticFeedback.heavyImpact();  // Crashes on old Android
```

### ✅ Safe haptic
```dart
try {
  await HapticFeedback.heavyImpact();
} catch (_) {
  // Graceful fallback
}
```

