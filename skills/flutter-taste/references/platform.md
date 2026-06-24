# Platform-Specific Patterns

> **Key principle:** Respect platform conventions. Don't ask permission when unnecessary. Use platform-native solutions for common tasks.

## Pattern 1: Save Files Without Permission Handling (Share Instead of Download)

### The Problem: Download Permission Friction

**Android 10+** and **iOS** have stricter file access:
- User expects immediate download
- Instead: Permission dialog appears
- Downloads might fail silently
- Bad UX for simple file sharing

### ❌ Anti-Pattern: Direct file write (requires permissions)
```dart
// Requires WRITE_EXTERNAL_STORAGE permission (annoying)
// Fails silently on Android 10+
// iOS has no downloads folder
Future<void> _downloadFile(String url, String filename) async {
  final response = await http.get(Uri.parse(url));
  final file = File('/sdcard/Download/$filename');
  await file.writeAsBytes(response.bodyBytes);
  // ❌ Permission request, might fail
}
```

### ✅ Pattern: Use share/open instead
```dart
import 'package:share_plus/share_plus.dart';
import 'package:path_provider/path_provider.dart';

// Option 1: Share file directly (platform handles everything)
Future<void> _shareFile(String url, String filename) async {
  final response = await http.get(Uri.parse(url));
  final tempDir = await getTemporaryDirectory();
  final tempFile = File('${tempDir.path}/$filename');
  await tempFile.writeAsBytes(response.bodyBytes);

  await SharePlus.instance.share(
    ShareParams(files: [XFile(tempFile.path)], text: 'Check this out!'),
  );
  // iOS: Share menu appears → user chooses (Mail, Files, AirDrop, etc.)
  // Android: Share menu appears → user chooses (Gmail, Drive, etc.)
  // No permission needed!
}

// Usage
ElevatedButton(
  onPressed: () => _shareFile('https://example.com/file.pdf', 'document.pdf'),
  child: Text('Share File'),
)
```

### Pattern 2: Open file in platform-native app
```dart
import 'package:open_file/open_file.dart';
import 'package:path_provider/path_provider.dart';

Future<void> _openFile(String url, String filename) async {
  final response = await http.get(Uri.parse(url));
  final tempDir = await getTemporaryDirectory();
  final tempFile = File('${tempDir.path}/$filename');
  await tempFile.writeAsBytes(response.bodyBytes);
  
  // Platform opens with native app (PDF reader, etc.)
  await OpenFile.open(tempFile.path);
}

// Usage
ElevatedButton(
  onPressed: () => _openFile('https://example.com/file.pdf', 'document.pdf'),
  child: Text('View PDF'),
)
```

### Pattern 3: Save to app-specific directory (no permission needed)
```dart
import 'package:path_provider/path_provider.dart';

// App-specific documents (no permissions required)
Future<void> _saveToAppDocuments(String filename, List<int> bytes) async {
  final appDir = await getApplicationDocumentsDirectory();
  final file = File('${appDir.path}/$filename');
  await file.writeAsBytes(bytes);
  print('Saved to: ${file.path}');
  // User can access via Files app or in-app
}

// Usage
_saveToAppDocuments('report.pdf', pdfBytes);
```

### Pattern 4: Full download flow (if permission unavoidable)
```dart
import 'package:permission_handler/permission_handler.dart';
import 'package:path_provider/path_provider.dart';

Future<void> _downloadWithPermission(String url, String filename) async {
  // Check/request permission
  final status = await Permission.storage.request();
  
  if (status.isDenied) {
    _showError('Storage permission required');
    return;
  }
  
  if (status.isPermanentlyDenied) {
    _showError('Please enable storage permission in settings');
    openAppSettings();
    return;
  }
  
  // Download to standard Download directory
  final downloadDir = Directory('/storage/emulated/0/Download');
  if (!await downloadDir.exists()) {
    downloadDir.createSync(recursive: true);
  }
  
  try {
    final response = await http.get(Uri.parse(url));
    final file = File('${downloadDir.path}/$filename');
    await file.writeAsBytes(response.bodyBytes);
    
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('Downloaded to Downloads folder')),
    );
  } catch (e) {
    _showError('Download failed: $e');
  }
}
```

### Comparison Table

| Method | Permission | iOS Support | Android | UX | Best For |
|--------|-----------|-----------|---------|----|---------| 
| **Share** | ❌ None | ✅ Yes | ✅ Yes | ⭐⭐⭐ Excellent | Most files |
| **Open** | ❌ None | ✅ Yes | ✅ Yes | ⭐⭐⭐ Excellent | Viewing files |
| **App Docs** | ❌ None | ✅ Yes | ✅ Yes | ⭐⭐ Good | App-specific |
| **Downloads** | ✅ Required | ❌ No | ✅ Android 5 only | ⭐ Poor | Legacy only |

---

## Implementation Decision Tree

```
User wants file?
  │
  ├─ Share with others?
  │   └─ Use SharePlus.instance.share(ShareParams(...)) ← BEST
  │
  ├─ View/edit in app?
  │   └─ Save to app documents
  │
  ├─ Open in native app?
  │   └─ Use OpenFile.open() ← BEST
  │
  └─ Access via Files app?
      └─ Share OR use Downloads (Android only)
```

---

## Testing Checklist

- [ ] File share works without permission request
- [ ] Share menu appears on iOS + Android
- [ ] PDF opens in native reader
- [ ] Image opens in native gallery
- [ ] App-specific files persist between sessions
- [ ] No permissions requested unnecessarily
- [ ] Error handling if download fails

---

## Package Recommendations

| Package | Use | Cost |
|---------|-----|------|
| `share_plus` | Share files/text to other apps | Small, zero-dependency |
| `open_file` | Open file in native app | Small |
| `path_provider` | Get standard directories | Standard, built-in-like |
| `permission_handler` | Only if absolutely need Downloads | Medium, requires careful use |

**Recommendation:** Use `share_plus` + `open_file` for 90% of file handling. Avoid `permission_handler` unless Downloads folder is critical requirement.

---

## iOS Notes

- No "Downloads" folder concept
- Share menu is primary distribution method
- Files app integration available (iOS 11+)
- App-specific documents best option for persistence

## Android Notes

- Downloads folder available (Android 5+)
- Scoped storage (Android 10+) restricts direct access
- Share is permission-free and preferred
- App-specific directories always work

---

## Common Pitfalls

### ❌ Requesting storage permission for Share
```dart
// Unnecessary—share_plus handles it
await Permission.storage.request();
await SharePlus.instance.share(ShareParams(files: [XFile(path)]));
```

### ✅ Share without permission
```dart
// Just share—platform handles everything
await SharePlus.instance.share(ShareParams(files: [XFile(path)]));
```

### ❌ Trying to access Downloads on iOS
```dart
// Doesn't exist on iOS
File('/sdcard/Download/file.pdf')
```

### ✅ Check platform first
```dart
if (Platform.isAndroid) {
  // Use Downloads
} else {
  // Use Share or app documents
}
```

