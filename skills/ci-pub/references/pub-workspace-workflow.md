# Dart Pub Workspace Publication & CI Runbook

Operational reference for releasing multi-package Dart Pub workspaces (Dart 3.6+) to pub.dev.

---

## 1. Topological Sorting & Dependency Ordering

When releasing multiple workspace member packages, packages must be published in **topological dependency order**:
- **Rule**: Leaf dependencies (e.g. `core`, `logging`, `primitives`) MUST be published and verified on pub.dev before dependent consumers (e.g. `ui`, `feature`, `cli`).
- **Path Dependency Ban**: Member packages must declare intra-workspace dependencies as standard hosted version constraints (e.g. `core: ^1.2.0`), never local `path:` dependencies. Local path dependencies cause `dart pub publish` to abort.

---

## 2. Pub.dev CDN Caching & Propagation Window

### The 120-Second Cache Window
- Pub.dev metadata endpoints (`https://pub.dev/api/packages/<pkg>`) are served via Google Cloud Storage behind Cloud CDN with header `cache-control: public, max-age=120`.
- Even after a package upload completes, CDN edge nodes can return cached data for up to **120 seconds**.
- **The Pitfall**: In a pub workspace, `dart pub -C <downstream> publish` passes local client-side validation because siblings are present locally. However, if uploaded before the upstream dependency propagates, external consumers and the automated **Pana** scoring bot will fail to resolve dependencies, causing **Pana to score 0 points**.
- **Publishing is Forever**: Releases on pub.dev are immutable and cannot be deleted or unpublished.

### Active Polling Verification Pattern
Before publishing a dependent package, poll the pub.dev API to confirm the upstream version is live:

```bash
#!/usr/bin/env bash
verify_pub_dev_version() {
  local PKG="$1"
  local EXPECTED_VER="$2"
  local MAX_RETRIES=18 # 18 * 10s = 180s (3 minutes)

  echo "Waiting for $PKG v$EXPECTED_VER to propagate on pub.dev..."
  for ((i=1; i<=MAX_RETRIES; i++)); do
    LIVE_VER=$(curl -s "https://pub.dev/api/packages/$PKG" | grep -o '"version":"[^"]*"' | head -1 | cut -d'"' -f4 || true)
    if [ "$LIVE_VER" = "$EXPECTED_VER" ]; then
      echo "Verified: $PKG v$EXPECTED_VER is active on pub.dev."
      return 0
    fi
    sleep 10
  done
  echo "Timeout waiting for $PKG v$EXPECTED_VER propagation."
  return 1
}
```

---

## 3. Git Tagging & Release Conventions

### Scoped Tag Pattern
- **Standard**: `<package>-v<version>` (e.g., `package_name-v1.0.0`).
- **Compatibility**: Matches official `flutter/packages` standards, Melos conventions, and `pub.dev` automated publishing OIDC requirements (`tag-pattern: <package>-v{{version}}`).

### Single Atomic Release Commit
- All `pubspec.yaml` and `CHANGELOG.md` updates across the release batch belong to a single commit:
  ```bash
  git commit -m "chore(release): publish packages [package_a 0.1.1, package_b 0.2.0]"
  ```
- All release tags for the batch point to this commit SHA.

---

## 4. GitHub Actions OIDC Workflow

Automated tag-triggered publishing using native workspace resolution and dynamic package targeting is defined in the companion template:
- Seed template: `references/workflow-workspace.yml` (and `references/workflow-single.yml` for single package).
