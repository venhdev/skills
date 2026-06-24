# Forms & Input Handling

> **Key principle:** Forms should require minimal user friction. Autofocus, dismiss keyboard on scroll, handle platform differences.

## Pattern 1: Autofocus on Single-Field Pages

### ❌ Anti-Pattern: User must tap
```dart
class LoginPage extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return TextField(
      decoration: InputDecoration(labelText: 'Email'),
      // User must tap to focus; keyboard doesn't appear
    );
  }
}
```

### ✅ Pattern: Keyboard appears automatically
```dart
class LoginPage extends StatefulWidget {
  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  late FocusNode _emailFocus;

  @override
  void initState() {
    super.initState();
    _emailFocus = FocusNode();
    // Autofocus after frame build
    WidgetsBinding.instance.addPostFrameCallback((_) {
      FocusScope.of(context).requestFocus(_emailFocus);
    });
  }

  @override
  void dispose() {
    _emailFocus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        TextField(
          focusNode: _emailFocus,
          decoration: InputDecoration(labelText: 'Email'),
          keyboardType: TextInputType.emailAddress,
        ),
        // Single field → autofocus in initState
      ],
    );
  }
}
```

### Simpler: Direct autofocus attribute
```dart
TextField(
  autofocus: true,  // Only use if single field on page
  decoration: InputDecoration(labelText: 'Email'),
)
```

### Scope: when to apply
- **Do:** OTP page, phone number on registration, change email — one field, one job.
- **Don't:** A login page with email + Google/Apple sign-in. Keyboard opens over those buttons, hiding options the user might pick instead.
- **Skip** any "tap outside to dismiss keyboard" gesture on these pages — nothing else to reach, and dropping the keyboard only adds a tap to bring it back.

### OTP / PIN fields: keep keyboard on completion
With `pinput`, `closeKeyboardWhenCompleted` defaults to `true`. A wrong code leaves the user retyping with the keyboard already gone. Set it explicitly to `false`:
```dart
Pinput(
  autofocus: true,
  length: 4,
  onCompleted: (pin) => _verify(pin),
  closeKeyboardWhenCompleted: false,
)
```

---

## Pattern 2: Dismiss Keyboard on Scroll

### ❌ Anti-Pattern: Keyboard blocks content
```dart
ListView(
  children: [
    TextField(decoration: InputDecoration(labelText: 'Name')),
    TextField(decoration: InputDecoration(labelText: 'Email')),
    // User scrolls, keyboard persists and blocks the view
  ],
)
```

### ✅ Pattern: Use the built-in `keyboardDismissBehavior`
`ListView`, `SingleChildScrollView`, `CustomScrollView`, and `GridView` all accept `keyboardDismissBehavior`. `onDrag` dismisses as soon as the user starts scrolling — no controller or listener needed.
```dart
ListView(
  keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
  children: [
    TextField(decoration: InputDecoration(labelText: 'Name')),
    TextField(decoration: InputDecoration(labelText: 'Email')),
  ],
)
```

### Scope: when to apply
- **Do:** any form screen where the user finishes a field and scrolls to inspect more content below.
- **Don't:** chat screens. Here scrolling means reading past messages — the user is still mid-thought. Closing the keyboard on drag cuts off what they were typing. Leave it open.
- **Fallback:** if you need to dismiss on tap-outside as well as scroll, wrap with `GestureDetector(onTap: () => FocusManager.instance.primaryFocus?.unfocus(), child: ...)`.

---

## Pattern 2b: Dismiss Keyboard Before Opening a Modal

> **Why this is its own concern:** Tapping a field, typing, then tapping something that opens a modal (picker, dropdown, dialog) appears to close the keyboard when the modal opens. But closing the modal reopens the keyboard and refocuses the original field — even though the user was done typing.

### ❌ Anti-Pattern: Modal opens on top of open keyboard
```dart
CustomDropdown(
  title: 'City',
  value: selectedCity,
  onTap: () {
    // Modal appears over open keyboard
    showModalBottomSheet(context: context, builder: (_) => CityPicker());
  },
)
// After picking + closing: keyboard pops back up, original field re-focuses
```

### ✅ Pattern: Unfocus first
```dart
CustomDropdown(
  title: 'City',
  value: selectedCity,
  onTap: () async {
    // Always dismiss the keyboard first
    FocusManager.instance.primaryFocus?.unfocus();

    // Then open picker, update state, etc.
    showModalBottomSheet(context: context, builder: (_) => CityPicker());
  },
)
```

### Why `FocusManager`, not `FocusScope`
- `FocusScope.of(context)` finds the **nearest** `Focus` widget in the tree — wrong when the focused field is nested inside a different subtree.
- `FocusManager.instance.primaryFocus` holds the **latest** focused node globally — the right call here.

Use `FocusManager` for any "dismiss the active input" intent. Use `FocusScope` only when you want to move focus to a specific node in the current tree.

---

## Pattern 3: TextInput Styling

### ❌ Anti-Pattern: Default doesn't match brand
```dart
TextField(
  decoration: InputDecoration(labelText: 'Email'),
  // Uses default Material blue underline
)
```

### ✅ Pattern: Custom InputDecoration matching design
```dart
TextField(
  decoration: InputDecoration(
    labelText: 'Email',
    labelStyle: TextStyle(color: Colors.grey[600]),
    prefixIcon: Icon(Icons.email),
    border: OutlineInputBorder(
      borderRadius: BorderRadius.circular(8),
      borderSide: BorderSide(color: Colors.grey[300]!),
    ),
    focusedBorder: OutlineInputBorder(
      borderRadius: BorderRadius.circular(8),
      borderSide: BorderSide(color: Colors.blue, width: 2),
    ),
    errorBorder: OutlineInputBorder(
      borderRadius: BorderRadius.circular(8),
      borderSide: BorderSide(color: Colors.red),
    ),
    contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 12),
  ),
  style: TextStyle(fontSize: 16),
)
```

### Reusable theme
```dart
class AppInputDecoration {
  static InputDecoration standard({
    required String label,
    IconData? icon,
    String? hintText,
  }) {
    return InputDecoration(
      labelText: label,
      hintText: hintText,
      prefixIcon: icon != null ? Icon(icon) : null,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(8),
        borderSide: BorderSide(color: AppColors.primary, width: 2),
      ),
      contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    );
  }
}

// Usage
TextField(decoration: AppInputDecoration.standard(label: 'Email', icon: Icons.email))
```

---

## Pattern 4: Form Validation with UX

### ❌ Anti-Pattern: Errors only on submit
```dart
class LoginForm extends StatefulWidget {
  @override
  State<LoginForm> createState() => _LoginFormState();
}

class _LoginFormState extends State<LoginForm> {
  String? _email;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        TextField(
          onChanged: (v) => _email = v,
          decoration: InputDecoration(labelText: 'Email'),
        ),
        ElevatedButton(
          onPressed: _validate,
          child: Text('Login'),
        ),
      ],
    );
  }

  void _validate() {
    if (_email == null || _email!.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Email is required')),
      );
    }
  }
}
```

### ✅ Pattern: Real-time validation with visual feedback
```dart
class LoginForm extends StatefulWidget {
  @override
  State<LoginForm> createState() => _LoginFormState();
}

class _LoginFormState extends State<LoginForm> {
  final _formKey = GlobalKey<FormState>();
  late TextEditingController _emailController;
  String? _emailError;

  @override
  void initState() {
    super.initState();
    _emailController = TextEditingController();
  }

  bool _isValidEmail(String email) {
    const pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$';
    return RegExp(pattern).hasMatch(email);
  }

  void _validateEmail(String value) {
    setState(() {
      if (value.isEmpty) {
        _emailError = 'Email is required';
      } else if (!_isValidEmail(value)) {
        _emailError = 'Enter a valid email';
      } else {
        _emailError = null;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    return Form(
      key: _formKey,
      child: Column(
        children: [
          TextField(
            controller: _emailController,
            onChanged: _validateEmail,
            decoration: InputDecoration(
              labelText: 'Email',
              errorText: _emailError,
              errorBorder: OutlineInputBorder(
                borderSide: BorderSide(color: Colors.red),
              ),
              border: OutlineInputBorder(),
            ),
            keyboardType: TextInputType.emailAddress,
          ),
          SizedBox(height: 16),
          ElevatedButton(
            onPressed: _emailError == null ? _submit : null,
            child: Text('Login'),
          ),
        ],
      ),
    );
  }

  void _submit() {
    print('Email: ${_emailController.text}');
  }

  @override
  void dispose() {
    _emailController.dispose();
    super.dispose();
  }
}
```

---

## Pattern 5: Platform-Specific Keyboard Behavior

### ❌ Anti-Pattern: Ignores platform conventions
```dart
// Android: Return button labeled "Go"
// iOS: Return button labeled "Go"
// Both should adapt
TextField(
  textInputAction: TextInputAction.go,
)
```

### ✅ Pattern: Platform-aware return action
```dart
TextField(
  textInputAction: Platform.isIOS 
    ? TextInputAction.done 
    : TextInputAction.go,
  onSubmitted: (_) => _submit(),
)
```

---

## Pattern 6: Multi-field Form Flow

### ✅ Pattern: Field navigation with keyboard
```dart
class MultiFieldForm extends StatefulWidget {
  @override
  State<MultiFieldForm> createState() => _MultiFieldFormState();
}

class _MultiFieldFormState extends State<MultiFieldForm> {
  late FocusNode _nameFocus;
  late FocusNode _emailFocus;
  late FocusNode _passwordFocus;

  @override
  void initState() {
    super.initState();
    _nameFocus = FocusNode();
    _emailFocus = FocusNode();
    _passwordFocus = FocusNode();
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        TextField(
          focusNode: _nameFocus,
          decoration: InputDecoration(labelText: 'Name'),
          textInputAction: TextInputAction.next,
          onSubmitted: (_) => FocusScope.of(context).requestFocus(_emailFocus),
        ),
        TextField(
          focusNode: _emailFocus,
          decoration: InputDecoration(labelText: 'Email'),
          textInputAction: TextInputAction.next,
          keyboardType: TextInputType.emailAddress,
          onSubmitted: (_) => FocusScope.of(context).requestFocus(_passwordFocus),
        ),
        TextField(
          focusNode: _passwordFocus,
          decoration: InputDecoration(labelText: 'Password'),
          textInputAction: TextInputAction.done,
          obscureText: true,
          onSubmitted: (_) => _submit(),
        ),
        ElevatedButton(
          onPressed: _submit,
          child: Text('Sign Up'),
        ),
      ],
    );
  }

  void _submit() {
    FocusManager.instance.primaryFocus?.unfocus();
    print('Form submitted');
  }

  @override
  void dispose() {
    _nameFocus.dispose();
    _emailFocus.dispose();
    _passwordFocus.dispose();
    super.dispose();
  }
}
```

---

## Testing Checklist

- [ ] Single-field pages autofocus? Test on device (emulator may not show keyboard)
- [ ] Keyboard dismisses when user scrolls form?
- [ ] TextField styling matches brand?
- [ ] Validation shows errors in real-time (not just on submit)?
- [ ] Multi-field forms flow with `.next` action?
- [ ] Text input actions appropriate for platform?
- [ ] No keyboard covers critical content?

---

## Common Pitfalls

### ❌ Forgetting to dispose FocusNode
```dart
FocusNode _focus = FocusNode();
// Memory leak: node not released
```

### ✅ Dispose in cleanup
```dart
@override
void dispose() {
  _focus.dispose();
  super.dispose();
}
```

---

## Package Recommendations

| Package | Use |
|---------|-----|
| (Built-in) | TextField, Form, FocusNode (standard) |
| `flutter_keyboard_visibility` | Listen for keyboard show/hide |
| `keyboard_submitter` | Handle return key across platforms |

