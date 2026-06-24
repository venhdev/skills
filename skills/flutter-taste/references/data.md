# Data & Null-Safety UX

> **Key principle:** Never show raw `null` to users. Always provide meaningful fallbacks and graceful state handling (loading, error, empty).

## Pattern 1: Safe Text Display

### ❌ Anti-Pattern: Null display
```dart
Text(user?.name)  // Shows "null" if name is null
Text(user?.name ?? '')  // Shows empty string (confusing)
Text('${data?.title}')  // Shows "null" string
```

### ✅ Pattern: Meaningful fallback
```dart
Text(user?.name ?? 'Guest User')
Text(user?.name ?? 'Unnamed Item')
Text(user?.name ?? 'No title')

// Conditional for completely missing data
Text(product?.price != null ? '\$${product!.price}' : 'Price unavailable')
```

### Best Practice: Extract to helper
```dart
class DisplayStrings {
  static String userName(User? user) => user?.name ?? 'Guest';
  static String productTitle(Product? product) => product?.title ?? 'Unknown';
  static String priceString(double? price) => 
    price != null ? '\$${price.toStringAsFixed(2)}' : 'N/A';
}

// Usage
Text(DisplayStrings.userName(user))
Text(DisplayStrings.priceString(product.price))
```

### Best Practice: `String?` extension (handles literal `"null"` string)
```dart
extension StringX on String? {
  bool get isUsable =>
      this != null && this!.isNotEmpty && this != 'null';
  String orPlaceholder([String placeholder = '-']) =>
      isUsable ? this! : placeholder;
}

// Usage
Text(user.name.orPlaceholder())                 // null → "-"
Text(user.name.orPlaceholder('N/A'))            // null → "N/A"
Text(order.totalPrice.toString().orPlaceholder())
// Wrap in () when receiver is null-aware — keeps orPlaceholder reachable:
Text((state.cartInfo?.totalPrice?.toString()).orPlaceholder())
```

> Why the extension: `?? '-'` misses the literal string `"null"` that some backends serialize, and `?? ''` still leaves a blank spot. `isUsable` treats all three (null, empty, `"null"`) as missing.

---

## Pattern 2: Image Loading with Reserved Space

### ❌ Anti-Pattern: Layout shift
```dart
// Image pops in; layout jumps
Image.network(imageUrl, fit: BoxFit.cover)

// Size unknown; default 0×0
Image.asset('assets/avatar.png')
```

### ✅ Pattern: Fixed dimensions
```dart
// Reserve space; no shift
SizedBox(
  width: 200,
  height: 200,
  child: Image.network(
    imageUrl,
    fit: BoxFit.cover,
  ),
)

// Or with aspect ratio (if width is flexible)
AspectRatio(
  aspectRatio: 16 / 9,
  child: Image.network(imageUrl, fit: BoxFit.cover),
)
```

### Pattern: Fade-in on load (no jarring pop-in)
```dart
// Default: image pops in once decoded — feels abrupt
Image.network(imageUrl)

// Better: cross-fade placeholder → image via image_fade package
ImageFade(
  image: NetworkImage(imageUrl),
  // or: image: AssetImage('assets/avatar.png')
  // or: image: CachedNetworkImageProvider(imageUrl)
  width: 200,
  height: 200,
  fit: BoxFit.cover,
  duration: const Duration(milliseconds: 300),
  placeholder: Container(color: Colors.grey.shade100),
  errorBuilder: (context, _) => Container(
    color: Colors.grey.shade100,
    child: Icon(Icons.image_not_supported,
        color: Colors.grey.shade400, size: 20),
  ),
)
```
Key points:
- Wrap in `SizedBox` / `AspectRatio` to reserve space (Pattern above).
- Keep placeholder subtle: solid light grey or a shimmer — **avoid spinners**, they're distracting.
- Same for errors: small icon, no technical messages.
- `image` accepts any `ImageProvider` (`NetworkImage`, `AssetImage`, `CachedNetworkImageProvider`, `ExtendedNetworkImageProvider`, …).

---

### Pattern: Image with skeleton loader
```dart
class ImageWithSkeleton extends StatelessWidget {
  final String imageUrl;
  final double width;
  final double height;

  const ImageWithSkeleton({
    required this.imageUrl,
    required this.width,
    required this.height,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: width,
      height: height,
      child: Image.network(
        imageUrl,
        fit: BoxFit.cover,
        loadingBuilder: (context, child, progress) {
          if (progress == null) return child;
          return Center(
            child: CircularProgressIndicator(
              value: progress.expectedTotalBytes != null
                  ? progress.cumulativeBytesLoaded / progress.expectedTotalBytes!
                  : null,
            ),
          );
        },
        errorBuilder: (context, error, stackTrace) {
          return Container(
            color: Colors.grey[300],
            child: Icon(Icons.image_not_supported, color: Colors.grey[600]),
          );
        },
      ),
    );
  }
}
```

---

## Pattern 3: Empty & Error States

### ❌ Anti-Pattern: Blank screen
```dart
// Confusing: is it loading or empty?
if (items.isEmpty) {
  return SizedBox(); // Blank!
}
ListView(children: items)
```

### ✅ Pattern: Clear state indication
```dart
Widget _buildContent() {
  if (isLoading) {
    return Center(child: CircularProgressIndicator());
  }
  if (hasError) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.error_outline, size: 48, color: Colors.red),
          SizedBox(height: 16),
          Text('Something went wrong'),
          SizedBox(height: 8),
          ElevatedButton(
            onPressed: _retry,
            child: Text('Retry'),
          ),
        ],
      ),
    );
  }
  if (items.isEmpty) {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.inbox, size: 48, color: Colors.grey),
          SizedBox(height: 16),
          Text('No items yet', style: Theme.of(context).textTheme.titleMedium),
          SizedBox(height: 8),
          Text('Create your first item to get started',
              style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
    );
  }
  return ListView(children: items);
}
```

### Extract to state enum (recommended)
```dart
enum LoadingState { loading, error, empty, success }

class _PageState extends State<Page> {
  late LoadingState _state = LoadingState.loading;
  List<Item> items = [];
  String? errorMessage;

  @override
  void initState() {
    super.initState();
    _loadItems();
  }

  Future<void> _loadItems() async {
    setState(() => _state = LoadingState.loading);
    try {
      final loaded = await repository.fetchItems();
      setState(() {
        items = loaded;
        _state = loaded.isEmpty ? LoadingState.empty : LoadingState.success;
      });
    } catch (e) {
      setState(() {
        _state = LoadingState.error;
        errorMessage = e.toString();
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    switch (_state) {
      case LoadingState.loading:
        return Center(child: CircularProgressIndicator());
      case LoadingState.error:
        return _buildErrorState();
      case LoadingState.empty:
        return _buildEmptyState();
      case LoadingState.success:
        return ListView(children: items.map(_buildItem).toList());
    }
  }
}
```

---

## Pattern 4: Number Formatting

### ❌ Anti-Pattern: Raw numbers
```dart
Text('\$${price}')  // $1000000 (unreadable)
Text('${views} views')  // 1250000 views (hard to scan)
```

### ✅ Pattern: Formatted with intl
```dart
import 'package:intl/intl.dart';

// Currency
final currencyFormat = NumberFormat.currency(symbol: '\$');
Text(currencyFormat.format(1000000)) // $1,000,000.00

// Compact notation
final compactFormat = NumberFormat.compact();
Text(compactFormat.format(1250000)) // 1.3M

// Decimal places
final priceFormat = NumberFormat('#0.00');
Text(priceFormat.format(9.5)) // 9.50

// Group separator (for non-US locales)
final localFormat = NumberFormat('#,##0.00', 'en_US');
Text(localFormat.format(1234.5)) // 1,234.50
```

### Pattern: Locale-aware decimal grouping
```dart
// Picks user's region by default: en_US → "1,234,567", fr_FR → "1 234 567"
final count = NumberFormat.decimalPatternDigits().format(1234567);
// Force a locale:
NumberFormat.decimalPatternDigits(locale: 'de_DE').format(1234567); // 1.234.567
// Control fraction digits:
NumberFormat.decimalPatternDigits(decimalDigits: 2).format(9.5); // 9.50
```

### Pattern: `num` extension (recommended, covers 99% of cases)
```dart
extension NumX on num {
  String humanizedCount({int? decimalDigits}) =>
      NumberFormat.decimalPatternDigits(decimalDigits: decimalDigits).format(this);
  String humanizedCurrency(String code) =>
      NumberFormat.simpleCurrency(name: code).format(this);
  String humanizedCompact() => NumberFormat.compact().format(this);
  String humanizedCompactLong() => NumberFormat.compactLong().format(this);
}

// Usage
Text(price.humanizedCurrency('USD'))   // $1,234.50
Text(views.humanizedCompact())         // 1.23M
Text(views.humanizedCompactLong())     // 1.23 million
```

### Pattern: Tabular figures (for numbers that change)
```dart
// Problem: Text width changes as number updates (0 vs 9)
Text('12345')  // Width changes to 12

// Solution 1: Monospace font
Text('12345', style: TextStyle(fontFamily: 'RobotoMono'))

// Solution 2: Tabular figures
Text(
  '${value}',
  style: TextStyle(fontFeatures: [FontFeature.tabularFigures()]),
)

// For Google Fonts
Text(
  '${value}',
  style: GoogleFonts.roboto(fontFeatures: [FontFeature.tabularFigures()]),
)
```

---

## Pattern 5: Date & Time Formatting

### ❌ Anti-Pattern: Raw DateTime
```dart
Text(createdAt.toString())  // 2024-01-15 14:32:00.000
```

### ✅ Pattern: Human-readable
```dart
import 'package:intl/intl.dart';

final dateFormat = DateFormat('MMM d, yyyy');
Text(dateFormat.format(createdAt)) // Jan 15, 2024

// With time
final dateTimeFormat = DateFormat('MMM d, yyyy • h:mm a');
Text(dateTimeFormat.format(createdAt)) // Jan 15, 2024 • 2:32 PM

// Relative time (e.g., "2 hours ago")
import 'package:timeago/timeago.dart' as timeago;
Text(timeago.format(createdAt)) // about 2 hours ago
```

---

## Pattern 6: List Item Null Safety

### ❌ Anti-Pattern
```dart
ListView.builder(
  itemCount: users.length,
  itemBuilder: (context, index) {
    final user = users[index];
    return Text(user.name);  // Crash if null
  },
)
```

### ✅ Pattern
```dart
ListView.builder(
  itemCount: users.length,
  itemBuilder: (context, index) {
    final user = users[index];
    if (user == null) {
      return ListTile(
        title: Text('User unavailable'),
        leading: Icon(Icons.person_off),
      );
    }
    return ListTile(
      title: Text(user.name ?? 'Unnamed'),
      subtitle: Text(user.email ?? 'No email'),
    );
  },
)
```

---

## Common Pitfalls

### ❌ Empty vs. null confusion
```dart
Text(value ?? '')  // Hard to distinguish empty from missing
```

### ✅ Clear intent
```dart
Text(value ?? 'No value provided')  // Clear fallback
Text(value?.isEmpty ?? true ? 'Empty' : value!)  // Explicit
```

---

## Testing Checklist

- [ ] Does `Text()` ever show "null"? (Search codebase for `?.toString()` on displays)
- [ ] Do images have explicit dimensions? (Check `Image.network`, `Image.asset`)
- [ ] Are loading, error, empty states distinct and user-friendly?
- [ ] Are numbers formatted for human readability? (Thousands separator, currency)
- [ ] Are dates formatted consistently? (Use `DateFormat` with locale)

---

## Package Recommendations

| Package | Use |
|---------|-----|
| `intl` | NumberFormat, DateFormat (standard) |
| `timeago` | Relative dates ("2 hours ago") |
| `skeletons` | Skeleton loaders for image placeholders |
| `image_fade` | Cross-fade image on load (placeholder → image) |

All in `pubspec.yaml`:
```yaml
dependencies:
  intl: ^0.18.0
  timeago: ^3.4.0
  image_fade: ^0.4.0
```
