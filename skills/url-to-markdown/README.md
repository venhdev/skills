# URL-to-Markdown Skill

Convert any list of URLs into clean, readable Markdown files.

## Quick Start

### 1. Check Environment

```bash
node scripts/checker.js
```

This will:
- ✅ Verify Node.js version (18+)
- ✅ Check for required packages
- ✅ Report what needs to be installed (no auto-install)

### 2. Install Dependencies (if needed)

If checker reports missing packages:

```bash
npm install @mozilla/readability turndown turndown-plugin-gfm jsdom
# OR
pnpm add @mozilla/readability turndown turndown-plugin-gfm jsdom
```

### 3. Convert URLs

**Single URL:**
```bash
node scripts/cli.js https://example.com
```

**Multiple URLs:**
```bash
node scripts/cli.js https://url1 https://url2 https://url3
```

**From file:**
```bash
node scripts/cli.js --input urls.txt
```

**Custom output:**
```bash
node scripts/cli.js https://example.com --output my-file.md
```

## Options

```
--input FILE              Read URLs from file (one per line)
--output PATH             Output file path (default: .md/<auto>.md)
--keep-images true|false  Include images (default: true)
--keep-links true|false   Include links (default: true)
--keep-metadata true|false Include title/author/date (default: true)
--batch-size N            Concurrent URLs (default: 3)
--verbose                 Show detailed progress
--help                    Show help message
```

## Examples

### Remove images (faster, smaller)
```bash
node scripts/cli.js urls.txt --keep-images false
```

### LLM context prep (keep all, verbose)
```bash
node scripts/cli.js urls.txt --keep-images true --keep-links true --verbose
```

### Text-only extraction
```bash
node scripts/cli.js url --keep-images false --keep-links false --keep-metadata false
```

## Output

**Single URL:** Saves to `.md/<hostname>.md`
```markdown
# Article Title

_Author_

_Publish Date_

---

[Article content]

---
**Source:** https://example.com/article
```

**Multiple URLs:** Saves to `.md/<01-title>.md`, `.md/<02-title>.md`, etc.

## As a Library

Use in your own Node.js code:

```javascript
const { urlToMarkdown, batchUrlsToMarkdown } = require('./scripts/index');

// Single URL
const result = await urlToMarkdown('https://example.com');
console.log(result.markdown);

// Multiple URLs
const results = await batchUrlsToMarkdown(['url1', 'url2', 'url3']);
for (const [url, result] of results) {
  console.log(`${url} → ${result.title}`);
}
```

## File Structure

```
url-to-markdown-skill/
├── SKILL.md                 # Main skill definition
├── README.md               # This file
├── package.json            # Dependencies
├── scripts/
│   ├── index.js           # Module exports (use as library)
│   ├── cli.js             # CLI entry point
│   ├── converter.js       # Conversion logic
│   └── checker.js         # Environment check
└── references/
    └── ARCHITECTURE.md    # Technical deep-dive
```

## Troubleshooting

**"Module not found"**
→ Run `npm install`

**"HTTP 429: Too Many Requests"**
→ Decrease batch-size: `--batch-size 1`

**"Permission denied"**
→ Use `--output` to specify writable path

**"Empty output"**
→ Website may use JavaScript (JSDOM limitation)

## What It Does

1. **Fetches** HTML from each URL
2. **Extracts** main article content (removes ads, nav, sidebars)
3. **Converts** HTML → Markdown
4. **Cleans up** whitespace and formatting
5. **Saves** to `.md/` directory

Uses:
- `@mozilla/readability` for intelligent content extraction
- `turndown` for HTML→Markdown conversion
- `turndown-plugin-gfm` for GitHub Flavored Markdown support
- `jsdom` for DOM parsing in Node.js

## See Also

- `SKILL.md` - Full skill documentation
- `references/ARCHITECTURE.md` - Technical details
- Original project: [MD-This-Page](https://github.com/Ademking/MD-This-Page)
