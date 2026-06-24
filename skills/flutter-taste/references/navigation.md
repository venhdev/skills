# Navigation & Platform Patterns

> **Key principle:** Navigation should feel natural to the platform. iOS → smooth sheets, Android → adaptive dialogs. Back button and tab switching should behave as users expect.

## Pattern 1: Bottom Navigation — Scroll to Top (or Focus Search) on Retap

> Both Apple's HIG and Material spec this: tapping the active tab should scroll that page to top. If it's already at top, do a second thing — typically focus a search field. Flutter's `BottomNavigationBar` does neither for you.

### ❌ Anti-Pattern: Tapping active tab does nothing
```dart
BottomNavigationBar(
  currentIndex: _currentIndex,
  onTap: (int index) => setState(() => _currentIndex = index),
  items: const [/* ... */],
)
// Retap: nothing happens — users tap anyway, and it feels broken.
```

### ✅ Pattern: Scroll to top, or focus search if already at top
Keep long-lived `ScrollController` + `FocusNode` per tab — lifetime of the nav bar, not the tab page:

```dart
class TabKeys {
  TabKeys._();
  static final homeScroll = ScrollController();
  static final homeSearch = FocusNode();
  // ...one pair per tab...
}

void onTabReselected(int index) {
  final scroller = TabKeys.homeScroll; // pick by index
  final atTop = !scroller.hasClients || scroller.offset <= 0;

  if (!atTop) {
    scroller.animateTo(0,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOutCubic);
  } else {
    TabKeys.homeSearch.requestFocus(); // second action
  }
}

// In widget:
BottomNavigationBar(
  currentIndex: _currentIndex,
  onTap: (int index) {
    if (index == _currentIndex) {
      onTabReselected(index);
      return;
    }
    setState(() => _currentIndex = index);
  },
  items: const [/* ... */],
)
```

Tab page wires them in:
```dart
ListView(
  controller: TabKeys.homeScroll,
  children: [
    TextField(focusNode: TabKeys.homeSearch),
    // ...
  ],
)
```

**Lifecycle:** Controllers/nodes live as long as the bottom nav is on screen — never dispose them while in use. Dispose only from a root widget that owns the nav.

---

## Pattern 2: Adaptive Bottom Sheet (iOS vs Android)

### ❌ Anti-Pattern: Same sheet on both platforms
```dart
// Shows Material sheet on both iOS and Android (wrong feel on iOS)
showModalBottomSheet(
  context: context,
  builder: (context) => BottomSheetContent(),
)
```

### ✅ Pattern: Platform-aware sheet
```dart
void _showAdaptiveSheet(BuildContext context) {
  if (Theme.of(context).platform == TargetPlatform.iOS) {
    // iOS: Cupertino-style modal
    showCupertinoModalPopup(
      context: context,
      builder: (context) => CupertinoActionSheetContent(),
    );
  } else {
    // Android: Material bottom sheet
    showModalBottomSheet(
      context: context,
      builder: (context) => BottomSheetContent(),
    );
  }
}

// Or use built-in showAdaptiveBottomSheet if available (Flutter 3.10+)
showAdaptiveBottomSheet(
  context: context,
  builder: (context) => BottomSheetContent(),
)
```

### iOS Sheet Content
```dart
class CupertinoActionSheetContent extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return CupertinoActionSheetAction(
      child: Text('Delete'),
      isDestructiveAction: true,
      onPressed: () {
        Navigator.pop(context);
        _delete();
      },
    );
  }
}
```

---

## Pattern 3: Android Back Button with Custom Modals

### ❌ Anti-Pattern: Back button doesn't close modal
```dart
// Custom modal ignored Android back button
showDialog(
  context: context,
  builder: (context) => Dialog(
    child: Scaffold(
      appBar: AppBar(title: Text('Modal')),
      body: Content(),
    ),
  ),
)
// Android back button: nothing happens (frustrating)
```

### ✅ Pattern: Handle back button (Flutter 3.12+)
```dart
showDialog(
  context: context,
  builder: (context) => PopScope(
    canPop: true,  // Allow back button to close
    onPopInvoked: (didPop) {
      if (didPop) {
        Navigator.pop(context);
      }
    },
    child: Dialog(
      child: Scaffold(
        appBar: AppBar(title: Text('Modal')),
        body: Content(),
      ),
    ),
  ),
)
```

### Legacy (Flutter < 3.12): Use WillPopScope
```dart
showDialog(
  context: context,
  builder: (context) => WillPopScope(
    onWillPop: () async => true,  // Allow back to pop
    child: Dialog(
      child: Scaffold(
        appBar: AppBar(title: Text('Modal')),
        body: Content(),
      ),
    ),
  ),
)
```

> **Android 13+ root cause:** If back (and predictive-back edge swipe) still doesn't work, the app likely lacks the opt-in flag. Android 13 introduced `OnBackInvokedCallback`; without it, predictive back is disabled. Add to `AndroidManifest.xml`:
> ```xml
> <application
>     ...
>     android:enableOnBackInvokedCallback="true">
> ```
> Note: this flag breaks `WillPopScope` — migrate to `PopScope` first. MIUI/Xiaomi devices are the most common victims.

---

## Pattern 4: Custom Page Route with Spring Physics

### ✅ Pattern: Custom transition
```dart
class SpringPageRoute<T> extends PageRoute<T> {
  final WidgetBuilder builder;

  SpringPageRoute({required this.builder, RouteSettings? settings})
      : super(settings: settings);

  @override
  Duration get transitionDuration => Duration(milliseconds: 400);

  @override
  Color? get barrierColor => null;

  @override
  String? get barrierLabel => null;

  @override
  bool get maintainState => true;

  @override
  bool get fullscreenDialog => false;

  @override
  Widget buildPage(BuildContext context, Animation<double> animation,
      Animation<double> secondaryAnimation) {
    return builder(context);
  }

  @override
  Widget buildTransitions(BuildContext context, Animation<double> animation,
      Animation<double> secondaryAnimation, Widget child) {
    // Slide from right with spring curve
    final curveAnimation =
        CurvedAnimation(parent: animation, curve: Curves.elasticOut);

    return SlideTransition(
      position: Tween<Offset>(
        begin: Offset(1, 0),
        end: Offset.zero,
      ).animate(curveAnimation),
      child: FadeTransition(
        opacity: animation,
        child: child,
      ),
    );
  }
}

// Usage
Navigator.push(context, SpringPageRoute(builder: (_) => NextPage()))
```

---

## Pattern 5: Date Picker Adaptation

### ❌ Anti-Pattern: Material picker on iOS
```dart
// Shows ugly Material picker on iOS
final date = await showDatePicker(
  context: context,
  initialDate: DateTime.now(),
  firstDate: DateTime(2020),
  lastDate: DateTime(2030),
);
```

### ✅ Pattern: Platform-aware date picker
```dart
Future<DateTime?> showAdaptiveDatePicker(BuildContext context) async {
  if (Theme.of(context).platform == TargetPlatform.iOS) {
    // iOS: Cupertino-style picker
    DateTime selectedDate = DateTime.now();
    
    return await showCupertinoModalPopup<DateTime>(
      context: context,
      builder: (context) => Container(
        height: 250,
        color: CupertinoColors.systemBackground,
        child: CupertinoDatePicker(
          initialDateTime: DateTime.now(),
          mode: CupertinoDatePickerMode.date,
          onDateTimeChanged: (DateTime newDateTime) {
            selectedDate = newDateTime;
          },
        ),
      ),
    );
  } else {
    // Android: Material picker
    return showDatePicker(
      context: context,
      initialDate: DateTime.now(),
      firstDate: DateTime(2020),
      lastDate: DateTime(2030),
    );
  }
}

// Usage
final date = await showAdaptiveDatePicker(context);
```

---

## Pattern 6: iOS 26 Sheet Upgrade (Adaptive Sheet Route)

> **iOS-only polish.** On iOS 26+, the default `CupertinoSheetRoute` still renders the dated iOS 13 fully-rounded look. Use the `stupid_simple_sheet` package to render the new iOS 26 style with a frosted-glass barrier.

### ❌ Anti-Pattern: Default `CupertinoSheetRoute`
```dart
// Shows the iOS 13 sheet style on iOS 26 — looks dated
Navigator.of(context).push(
  CupertinoSheetRoute(builder: (_) => BarcodePage()),
);
```

### ✅ Pattern: Adaptive sheet route via extension
```dart
import 'package:flutter/foundation.dart';
import 'package:stupid_simple_sheet/stupid_simple_sheet.dart';

extension WidgetX on Widget {
  Route<T> asAdaptiveSheetRoute<T>() {
    final bool isIos = defaultTargetPlatform == TargetPlatform.iOS;
    if (isIos) {
      return StupidSimpleGlassSheetRoute(
        child: this,
        blurBehindBarrier: false,
      );
    }
    return StupidSimpleSheetRoute(child: this);
  }
}

// Usage
Navigator.of(context).push(BarcodePage().asAdaptiveSheetRoute());
```

### Platform notes
- iOS: `StupidSimpleGlassSheetRoute` — fully rounded, matches iOS 26.
- Android: `StupidSimpleSheetRoute` (swap for `MaterialPageRoute` if preferred).
- Modal sheets (bottom-anchored) get a close (X) at top-left, not a back arrow.

---

## Web-Specific Patterns

### Pattern 1: Dynamic Browser Tab Title

> `onGenerateTitle` looks like the fix but only reruns on rebuild, not navigation — it can't track routes. Use the `Title` widget instead.

### ❌ Anti-Pattern: `onGenerateTitle` or `dart:html`
```dart
// Wrong: only fires on rebuild, ignores route changes
MaterialApp(
  onGenerateTitle: (context) => 'MyApp',
)
```
```dart
// Avoid: dart:html is web-only and imperative — easy to forget per route
import 'dart:html' as html;
html.document.title = 'Home';
```

### ✅ Pattern: `Title` widget per route
```dart
@override
Widget build(BuildContext context) {
  return Title(
    title: 'Billing · MyApp',
    color: Theme.of(context).colorScheme.surface, // required, opaque (0xFF); feeds Android task switcher, ignored on web
    child: Scaffold(...),
  );
}
```

### ✅ Pattern: `go_router` — reactive title for all routes
```dart
MaterialApp.router(
  routerConfig: goRouter,
  builder: (context, child) => RouteTitle(child: child!),
)

class RouteTitle extends StatelessWidget {
  final Widget child;
  const RouteTitle({super.key, required this.child});

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: Listenable.merge([
        goRouter.routerDelegate,
        goRouter.routeInformationProvider,
      ]),
      builder: (context, _) {
        final path = goRouter.routeInformationProvider.value.uri.path;
        return Title(
          title: titleForPath(path),
          color: Theme.of(context).colorScheme.surface,
          child: child,
        );
      },
    );
  }
}

String titleForPath(String path) {
  final segments = Uri.parse(path).pathSegments;
  final page = switch (segments) {
    ['billing'] => 'Billing',
    ['project', final String id, 'insights'] => 'Insights · $id',
    ['project', final String id] => id,
    _ => null,
  };
  return page == null ? 'MyApp' : '$page · MyApp';
}
```

### Platform notes
- `Title.color` is required and must be opaque (`0xFF`); it feeds the Android task switcher and is ignored on web.
- Use the `Title` widget, not `onGenerateTitle`, so navigation events trigger updates.

### Pattern 2: Open Graph Meta Tags

In `web/index.html`:
```html
<!-- Open Graph (Facebook, LinkedIn, Slack, WhatsApp, …) -->
<meta property="og:title" content="MyApp">
<meta property="og:description" content="Amazing app for amazing people">
<meta property="og:image" content="https://myapp.com/preview.png">
<meta property="og:url" content="https://myapp.com">
<meta property="og:type" content="website">

<!-- X / Twitter — required: X reads its own tags first and only sometimes falls back to og:image -->
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="MyApp">
<meta name="twitter:description" content="Amazing app for amazing people">
<meta name="twitter:image" content="https://myapp.com/preview.png">
```

- `og:image` and `twitter:image` MUST be **absolute URLs** — relative paths get ignored.
- Image size: **1200×630** — off-ratio images get cropped or dropped.
- For dynamic OG tags, use a backend service or server-side rendering (Flutter web is client-side rendered).

### Pattern 3: Loading Progress Indicator

> Flutter web boots in stages: bootstrap → `main.dart.js` download → engine init → `runApp`. Show a progress bar so users don't stare at a blank page during the 3–5s download. Use `web/flutter_bootstrap.js` (Flutter uses it instead of the auto-generated one when present).

In `web/index.html`, place a static progress bar above the bootstrap script:
```html
<body>
  <div class="progress-container">
    <div class="progress-bar"></div>
  </div>
  <script src="flutter_bootstrap.js" async></script>
</body>

<style>
  body { margin: 0; height: 100vh; display: flex;
         justify-content: center; align-items: center;
         background: #ffffff; }
  .progress-container { width: 120px; height: 8px;
                        background: #FAFAFA; border-radius: 10px;
                        overflow: hidden; }
  .progress-bar { width: 0; height: 100%;
                  background: #B591FF;
                  transition: width 0.4s ease; }
</style>
```

In `web/flutter_bootstrap.js`, drive the bar from the loader callbacks:
```js
{{flutter_js}}
{{flutter_build_config}}

const bar = document.querySelector('.progress-bar');
const setProgress = (pct) => { bar.style.width = pct + '%'; };

setProgress(20); // bootstrap running

_flutter.loader.load({
  onEntrypointLoaded: async function (engineInitializer) {
    setProgress(50); // main.dart.js downloaded
    const appRunner = await engineInitializer.initializeEngine();
    setProgress(80); // engine ready
    await appRunner.runApp();
    // Flutter mounts into <body> and takes over — bar disappears on its own
  },
});
```

Keep the bar payload tiny (plain HTML/CSS) — don't ship a 2MB animated background or you'll trade one loading problem for another.

---

## Testing Checklist

- [ ] Bottom nav: Tapping active tab scrolls list to top (or focuses search if already at top)
- [ ] iOS: Sheet appears as Cupertino modal (slides from bottom, native feel)
- [ ] Android: Sheet appears as Material sheet
- [ ] Date picker: iOS uses wheel, Android uses calendar
- [ ] Back button: Custom modals close on Android back
- [ ] Page transition: Smooth spring physics, no jank
- [ ] Web: Tab title changes per route
- [ ] Web: OG tags display correctly in social previews

---

## Package Recommendations

| Package | Use |
|---------|-----|
| (Built-in) | PopScope, showAdaptiveBottomSheet (Flutter 3.10+) |
| `go_router` | Type-safe routing with platform adaptation |
| `routemaster` | Named routes + deep linking |

