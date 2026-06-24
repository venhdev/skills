# Animations & Spring Physics

> **Key principle:** Use physics-based animations (springs, damping) instead of linear or ease-in-out curves. Users feel the difference.

## Spring Physics Patterns

### Pattern 1: Button Press with Spring Scale

**Concern:** Default tap scales linearly; users want spring rebound.

```dart
class SpringButton extends StatefulWidget {
  final Widget child;
  final VoidCallback onPressed;

  const SpringButton({
    required this.child,
    required this.onPressed,
  });

  @State<SpringButton> createState() => _SpringButtonState();
}

class _SpringButtonState extends State<SpringButton>
    with SingleTickerProviderStateMixin {
  late AnimationController _controller;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      duration: Duration(milliseconds: 300),
      vsync: this,
    );
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _onTapDown(TapDownDetails details) {
    _controller.forward();
  }

  void _onTapUp(TapUpDetails details) {
    _controller.reverse();
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTapDown: _onTapDown,
      onTapUp: _onTapUp,
      onTapCancel: () => _controller.reverse(),
      onTap: onPressed,
      child: AnimatedBuilder(
        animation: _controller,
        builder: (context, child) {
          // elasticOut gives springy feel; value goes 0→1→0.95 (with bounce)
          final springValue = Curves.elasticOut.transform(_controller.value);
          // Invert: press down = scale down
          final scale = 1.0 - (0.1 * springValue);
          return Transform.scale(scale: scale, child: child);
        },
        child: child,
      ),
    );
  }
}
```

**Usage:**
```dart
SpringButton(
  onPressed: () => print('Tapped!'),
  child: Container(
    padding: EdgeInsets.all(16),
    decoration: BoxDecoration(
      color: Colors.blue,
      borderRadius: BorderRadius.circular(8),
    ),
    child: Text('Press Me', style: TextStyle(color: Colors.white)),
  ),
)
```

**Anti-pattern:**
```dart
// ❌ Linear scale (rigid)
GestureDetector(
  onTap: () {},
  child: Transform.scale(scale: 0.9, child: button), // Static scale
)
```

**Modern preset (recommended):** Use the `motor` package's `SingleMotionBuilder` with platform-tuned presets instead of hand-tuning `Curves.elasticOut` or `SpringDescription`. Scale: `0.96` on press, `1.0` on release/cancel.

```dart
// iOS/macOS
SingleMotionBuilder(motion: CupertinoMotion.smooth(), value: _scale, builder: (_, scale, child) => Transform.scale(scale: scale, child: child!))
// Android/Web/other
SingleMotionBuilder(motion: MaterialSpringMotion(), value: _scale, builder: (_, scale, child) => Transform.scale(scale: scale, child: child!))
```

**Platform rule:** `CupertinoMotion` on iOS/macOS, `MaterialSpringMotion` elsewhere. One cross-platform preset is acceptable but platform-tuned feels more native.

---

### Pattern 2: List Scroll with Spring Deceleration

**Concern:** Lists feel stiff; scrolling stops abruptly.

```dart
class SpringScrollPhysics extends ScrollPhysics {
  const SpringScrollPhysics({ScrollPhysics? parent}) : super(parent: parent);

  @override
  SpringScrollPhysics applyTo(ScrollPhysics? ancestor) {
    return SpringScrollPhysics(parent: buildParent(ancestor));
  }

  @override
  Simulation? createBallisticSimulation(ScrollMetrics position, double velocity) {
    // Use spring simulation for natural deceleration
    if ((velocity <= 0.0 && position.pixels <= position.minScrollExtent) ||
        (velocity >= 0.0 && position.pixels >= position.maxScrollExtent)) {
      return null;
    }
    final Simulation simulation = SpringSimulation(
      SpringDescription.withDampingRatio(
        mass: 1.0,
        stiffness: 100.0, // Higher = bouncier
        damping: 0.8,     // 0.5–1.0 range
      ),
      position.pixels,
      position.pixels + velocity.sign * 0.1 * velocity.abs(),
      0.0,
    );
    return simulation;
  }
}
```

**Usage:**
```dart
ListView(
  physics: const SpringScrollPhysics(),
  children: [/* items */],
)
```

**Alternatives:**
- `BouncingScrollPhysics()` (iOS-like)
- `ClampingScrollPhysics()` (Android-like)
- Custom `SpringDescription.withDampingRatio()` for fine control

---

### Pattern 3: Page Transitions with Physics

**Concern:** Page swaps feel instant or uniform; no momentum.

```dart
class PhysicsPageRoute<T> extends MaterialPageRoute<T> {
  PhysicsPageRoute({
    required WidgetBuilder builder,
    RouteSettings? settings,
    bool maintainState = true,
    bool fullscreenDialog = false,
  }) : super(
    builder: builder,
    settings: settings,
    maintainState: maintainState,
    fullscreenDialog: fullscreenDialog,
  );

  @override
  Duration get transitionDuration => Duration(milliseconds: 400);

  @override
  Widget buildTransitions(
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) {
    // Spring curve: eases in with bounce
    final curve = Curves.elasticOut;
    final curvedAnimation = animation.drive(
      Tween<double>(begin: 0, end: 1).chain(
        CurveTween(curve: curve),
      ),
    );

    return SlideTransition(
      position: Tween<Offset>(
        begin: Offset(1, 0),
        end: Offset.zero,
      ).animate(curvedAnimation),
      child: FadeTransition(
        opacity: animation,
        child: child,
      ),
    );
  }
}
```

**Usage:**
```dart
Navigator.push(
  context,
  PhysicsPageRoute(builder: (context) => NextPage()),
)
```

---

## Curve Selection

| Curve | Feel | Use Case |
|-------|------|----------|
| `Curves.elasticOut` | Springy, bouncy | Button presses, confirmations |
| `Curves.easeOutCubic` | Smooth deceleration | Lists, page transitions |
| `Curves.easeInOutCubic` | Smooth both ways | Fade-ins, opacity changes |
| `Curves.fastOutSlowIn` | Material default | Subtle animations |
| `Curves.linear` | ❌ Avoid | Only for progress bars |

---

## Common Pitfalls

**❌ Mixing curves:**
```dart
// Don't jump between curves
AnimationController(duration: Duration(ms: 300)); // Then apply different curve per use
```

**✅ Consistent physics:**
```dart
// Define once, reuse
class AppCurves {
  static const spring = Curves.elasticOut;
  static const smooth = Curves.easeOutCubic;
}
```

---

## Testing on Device

Spring physics feel different on:
- **Emulator** (may stutter)
- **Physical device** (smooth)
- **Low-end device** (may jank if too many simultaneous animations)

**Recommendation:** Always test on real Android + iOS devices. Adjust dampingRatio if jank occurs.

---

## Package Recommendations

| Package | Use | Cost |
|---------|-----|------|
| (Built-in) | Spring physics, basic curves | Free, zero dependency |
| `fluttery` | Parallax, ripple effects | Small, stable |
| `flutter_animate` | Staggered, sequence animations | Medium, feature-rich |

Spring physics is built-in (use `SpringSimulation` directly). No package needed.
