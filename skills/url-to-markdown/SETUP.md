# URL-to-Markdown Skill Setup Guide

## Step 1: Environment Check

Before doing anything, verify your environment:

```bash
node scripts/checker.js
```

**Expected output:**
```
📋 URL-to-Markdown Environment Check

Node.js: ✓ Node.js 22.22.2 (required: v18.0.0+)

Dependencies:
  ⚠️  Missing packages:
  • @mozilla/readability
  • turndown
  • turndown-plugin-gfm
  • jsdom

📦 To install, run one of:
  npm install @mozilla/readability turndown turndown-plugin-gfm jsdom
  OR
  pnpm add @mozilla/readability turndown turndown-plugin-gfm jsdom
```

## Step 2: Install Dependencies

Choose one based on your package manager:

### Using npm:
```bash
npm install @mozilla/readability turndown turndown-plugin-gfm jsdom
```

### Using pnpm (preferred for VenhDev):
```bash
pnpm add @mozilla/readability turndown turndown-plugin-gfm jsdom
```

**This will:**
- Create `node_modules/` directory
- Download dependencies (~150MB)
- Take 30-60 seconds

## Step 3: Verify Installation

```bash
node scripts/checker.js
```

**Expected output:**
```
✓ Node.js 22.22.2 (required: v18.0.0+)

Dependencies:
  ✓ @mozilla/readability (v0.6.0)
  ✓ turndown (v7.2.4)
  ✓ turndown-plugin-gfm (v1.0.2)
  ✓ jsdom (v24.0.0)

Output: ✓ Output directory: /path/to/.md

✅ Environment OK - Ready to convert URLs
```

## Step 4: Try a Test Run

```bash
# Single URL
node scripts/cli.js https://example.com --verbose

# Multiple URLs
node scripts/cli.js https://example.com https://example.org --verbose

# From file
node scripts/cli.js --input test-urls.txt --verbose
```

## Step 5: Check Output

Look in the `.md/` directory:

```bash
ls -la .md/
cat .md/*.md
```

## Troubleshooting

### "Cannot find module '@mozilla/readability'"

**Cause:** Dependencies not installed

**Fix:**
```bash
npm install  # or pnpm add (auto-reads package.json)
node scripts/checker.js
```

### "EACCES: permission denied"

**Cause:** Missing write permissions on `.md/` directory

**Fix:**
```bash
# Specify writable output location
node scripts/cli.js https://example.com --output ~/Documents/result.md
```

### "HTTP 429: Too Many Requests"

**Cause:** Fetching too many URLs concurrently from the same host

**Fix:**
```bash
# Reduce batch size to 1-2
node scripts/cli.js urls.txt --batch-size 1
```

### "Cannot GET https://example.com"

**Cause:** Website might be down or not accessible

**Fix:**
```bash
# Test with a working URL
node scripts/cli.js https://en.wikipedia.org/wiki/Main_Page --verbose
```

## Usage After Setup

### CLI

```bash
# Help
node scripts/cli.js --help

# Single URL with all options
node scripts/cli.js https://example.com \
  --keep-images true \
  --keep-links true \
  --keep-metadata true \
  --output article.md \
  --verbose

# Batch from file
node scripts/cli.js --input urls.txt --batch-size 3 --verbose
```

### As a Library

In your Node.js code:

```javascript
const { urlToMarkdown } = require('./scripts/index');

const result = await urlToMarkdown('https://example.com');
console.log(result.markdown);
```

## File Locations

After setup:
- **Dependencies:** `node_modules/` (~150MB)
- **Configuration:** `package.json`
- **Scripts:** `scripts/`
- **Output:** `.md/` (created automatically)
- **Logs:** None (reports to stdout)

## Optional: Add to PATH

Make CLI accessible from anywhere:

```bash
# Create symlink or add to PATH
ln -s /path/to/url-to-markdown-skill/scripts/cli.js /usr/local/bin/url-to-markdown

# Or use npm global:
npm install -g .  # Makes 'url-to-markdown' command available
```

## Next Steps

1. ✅ Check environment
2. ✅ Install dependencies
3. ✅ Verify installation
4. ✅ Try a test run
5. 📚 Read SKILL.md for full documentation
6. 🔧 Customize as needed (edit scripts/)
7. 🚀 Use in your workflow

---

**Questions?** Check `SKILL.md` or `references/ARCHITECTURE.md`
