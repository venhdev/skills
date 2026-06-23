# URL-to-Markdown Architecture

## Overview

```
Input: URLs (CLI, file, or programmatic)
  ↓
Environment Check (Node.js, dependencies)
  ↓
Fetch HTML from each URL (with AbortController timeout)
  ↓
Hand off to Defuddle (via dynamic import — defuddle/node is ESM-only)
  ↓
  ├─ Defuddle internal pipeline:
  │   ├─ JSDOM parse
  │   ├─ Pre-cleanup (~30+ bad selectors)
  │   ├─ Lazy-image attribute resolution (data-src → src)
  │   ├─ Readability: score + extract main content
  │   ├─ Schema.org / JSON-LD / OpenGraph metadata extraction
  │   └─ Turndown + GFM: HTML → Markdown (with language hints, title attrs)
  ↓
Post-processing (in our wrapper):
  ├─ Strip raw-HTML <a>/<img> fallbacks (if keepLinks/keepImages is false)
  ├─ Optional customCleanupFn (user-provided)
  ├─ Optional page-structure map (Tier 3 — generatePageMap from heading scan)
  └─ Optional date format conversion (ISO ↔ locale)
  ↓
Save to .md/<filename>.md
  ↓
Output: Report success/failures
```

## Components

### 1. Checker (`scripts/checker.js`)
- Verifies Node.js version (18+)
- Checks for 5 required packages (`defuddle`, `@mozilla/readability`, `turndown`, `turndown-plugin-gfm`, `jsdom`)
- Validates output directory permissions
- Reports missing dependencies + installation instructions

### 2. Converter (`scripts/converter.js`)
- `urlToMarkdown(url, options)` - Single URL conversion
- `batchUrlsToMarkdown(urls, options, concurrency)` - Batch processing
- Wraps `Defuddle(html, url, options)` from `defuddle/node`
- Options: `keepImages`, `keepLinks`, `customCleanupFn`, `pageMap`, `dateFormat`
- Returns: `{ markdown, title, author, date, url, description }`

### 3. Page-Map (`scripts/page-map.js`)
- `generatePageMap(markdown, title)` — pure-JS, copied verbatim from MD-This-Page source
- Parses ATX-style headings from Markdown
- Builds tree, renders ASCII connectors (`├──`, `└──`, `│`)

### 4. CLI (`scripts/cli.js`)
- Command-line argument parsing
- Input file loading (.txt, .csv, .json)
- Output path management
- Progress reporting
- Error handling & summary
- New flags: `--page-map true|false`, `--date-format iso|locale`

### 5. Index (`scripts/index.js`)
- Module exports for library use
- Re-exports `urlToMarkdown`, `batchUrlsToMarkdown`, `checkNodeVersion`, `checkPackages`

## Conversion Pipeline

### Stage 1: Extract (Defuddle → Readability + schema.org)

```javascript
// Input: HTML string + URL
const article = await Defuddle(html, url, {
  separateMarkdown: true,
  removeImages: !keepImages,
  removeExactSelectors: true,
  removePartialSelectors: true,
  removeHiddenElements: true,
  removeLowScoring: true,
  standardize: true,         // footnotes, headings, code blocks
  removeContentPatterns: true,
  useAsync: false
});

// Output:
{
  title: "Article Title",
  author: "Author Name",        // from byline, og:author, or schema.org
  published: "2024-06-15",      // from article:published_time or schema.org datePublished
  description: "...",
  domain: "example.com",
  image: "https://...",
  content: "<article>... HTML ...</article>",
  contentMarkdown: "# Heading\n\n...",
  schemaOrgData: { ... },
  metaTags: [ ... ],
  wordCount: 1234
}
```

**What Defuddle does (in one call):**
1. Parses HTML via JSDOM
2. Pre-cleanup: removes ~30+ bad selectors (ads, social, cookies, popups, ARIA roles, breadcrumb, pagination, comment sections, …)
3. Resolves lazy-image attributes (`data-src`, `data-lazy-src`, `data-original` → `src`)
4. Runs Readability scoring on semantic blocks
5. Extracts highest-scoring subtree as main content
6. Pulls metadata from JSON-LD, schema.org, OpenGraph, `<meta>` tags
7. Runs Turndown with GFM plugin → Markdown
8. Code-block language detection from `class="language-xxx"`
9. Title-attribute preservation in links (`[text](url "title")`)
10. Footnote standardization (`[^1]` markers)
11. Inline-style stripping
12. Whitespace normalization

### Stage 2: Post-process (our wrapper)

```javascript
// Strip raw-HTML fallbacks (defuddle may emit both MD and HTML forms)
if (!keepImages) {
  markdown = markdown.replace(/<img[^>]*>/gi, '');
  markdown = markdown.replace(/<picture[^>]*>[\s\S]*?<\/picture>/gi, '');
}
if (!keepLinks) {
  markdown = markdown.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1');  // [text](url) → text
  markdown = markdown.replace(/<a[^>]*>(.*?)<\/a>/gi, '$1');    // <a>text</a> → text
}

// Custom cleanup hook
if (customCleanupFn) markdown = customCleanupFn(markdown);

// Optional page-structure map (Tier 3)
if (pageMap) {
  const map = generatePageMap(markdown, article.title);
  markdown = map + '\n\n---\n\n' + markdown;
}

// Optional date format
if (dateFormat === 'locale') {
  date = new Date(date).toLocaleDateString();
}
```

**Customization (handled by Defuddle internally):**
- Remove unwanted elements (script, style, nav, ads, social, cookies, …)
- Code-block language detection from `class="language-xxx"`
- Heading style (atx: `#` vs. setext: underlines)
- Code block style (fenced: ``` vs. indented)
- Footnote standardization
- Title-attribute preservation

### Stage 3: Save

CLI writes file(s) to `.md/<auto-desc>.md` (or custom `--output` path), with metadata header prepended in our H1 + italics format:

```markdown
# Title

_Author_

_Date_

---

[markdown body from defuddle]

---

**Source:** https://...
```

## Field Name Mapping (v1 → v2)

| v1 (Readability) | v2 (Defuddle) |
|---|---|
| `result.byline` | `result.author` |
| `result.publishedTime` | `result.date` |
| (not available) | `result.description` |
| `result.title` | `result.title` (unchanged) |
| `result.markdown` | `result.markdown` (unchanged) |
| `result.url` | `result.url` (unchanged) |

## CLI Usage Examples

### Single URL
```bash
node scripts/cli.js https://example.com
# Output: .md/example-com.md
```

### Multiple URLs (inline)
```bash
node scripts/cli.js https://url1 https://url2 https://url3
# Output: .md/00-title1.md, .md/01-title2.md, .md/02-title3.md
```

### From file
```bash
node scripts/cli.js --input urls.txt
# urls.txt contains one URL per line
```

### Options
```bash
# Remove images (smaller files)
--keep-images false

# Remove links (text-only)
--keep-links false

# Remove metadata
--keep-metadata false

# Increase concurrency
--batch-size 5

# Show detailed progress
--verbose

# Custom output
--output my-file.md
```

## Programmatic Usage

### Import as library
```javascript
const { urlToMarkdown, batchUrlsToMarkdown } = require('./scripts/index');

// Single URL
const result = await urlToMarkdown('https://example.com', {
  keepImages: true,
  keepLinks: true,
  keepMetadata: true
});

console.log(result.markdown);  // Markdown content
console.log(result.title);     // Extracted title
console.log(result.author);    // Extracted author
console.log(result.date);      // Publish date
console.log(result.url);       // Source URL

// Multiple URLs
const results = await batchUrlsToMarkdown(
  ['https://url1', 'https://url2'],
  { keepImages: true },
  3  // concurrency
);

for (const [url, result] of results) {
  if (!result.error) {
    console.log(`${url} → ${result.title}`);
  } else {
    console.error(`${url} failed: ${result.error}`);
  }
}
```

## File Structure

```
url-to-markdown-skill/
├── SKILL.md                 # Main skill definition
├── package.json            # NPM config
├── scripts/
│   ├── index.js           # Module exports
│   ├── cli.js             # CLI entry point
│   ├── converter.js       # Core conversion logic
│   └── checker.js         # Environment check
└── references/
    └── ARCHITECTURE.md    # This file
```

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| **defuddle** | ^0.19.0 | High-level orchestrator (wraps everything below) |
| @mozilla/readability | ^0.6.0 | Smart content extraction (via defuddle) |
| turndown | ^7.2.4 | HTML → Markdown conversion (via defuddle) |
| turndown-plugin-gfm | ^1.0.2 | GitHub Flavored Markdown (tables, strikethrough, task lists) |
| jsdom | ^24.0.0 | HTML parsing in Node.js (via defuddle) |

## Performance Characteristics

- **Per URL:** 1-3 seconds (network-dependent)
- **Memory:** ~50-150MB per concurrent process
- **Batch size:** 3 is optimal for most networks
- **Largest file:** HTML content determines markdown size

## Error Handling

| Error | Behavior | Recovery |
|-------|----------|----------|
| HTTP error (404, 503, etc.) | Logged, skipped | Continue with next URL |
| Network timeout | Logged, skipped | Continue with next URL |
| Invalid URL format | Logged, skipped | Continue with next URL |
| Readability failure | Falls back to `<body>` | Content extraction attempted |
| Missing dependencies | Reported, exit | Installation instructions provided |
| Output permission error | Logged, exit | Suggest alternative output path |

## Output Format

### Single URL
```markdown
# Article Title

_Author Name_

_2024-06-15_

---

[Markdown content here]

---
**Source:** https://example.com/article
```

### Multiple URLs
```
.md/
├── 00-SUMMARY.txt           (List of files created)
├── 01-first-article.md
├── 02-second-article.md
└── 03-third-article.md
```

## Customization Points

### Custom Cleanup Function
```javascript
const customCleanup = (markdown) => {
  // Remove URLs
  return markdown.replace(/https?:\/\/\S+/g, '');
};

const result = await urlToMarkdown(url, {
  customCleanupFn: customCleanup
});
```

### Custom Turndown Rules
Edit `scripts/converter.js` to add custom rules:
```javascript
turndownService.addRule('customRule', {
  filter: (node) => node.nodeName === 'CUSTOM',
  replacement: (content) => `**${content}**`
});
```

### Custom Output Naming
Edit `scripts/cli.js` `generateOutputPath()` function to change file naming scheme.

## Testing

```bash
# Check environment
npm run check

# Show help
npm run cli -- --help

# Test single URL
npm run cli -- https://example.com --verbose

# Test multiple URLs
npm run cli -- https://url1 https://url2 --verbose

# Test from file
npm run cli -- --input test-urls.txt --verbose
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Cannot find module" | Run `npm install` |
| "HTTP 429: Too Many Requests" | Decrease `--batch-size` to 1-2 |
| "Empty output" | Website might use JavaScript rendering (JSDOM limitation) |
| "Permission denied" output | Use `--output` to specify writable path |

## Future Improvements

- [ ] Support browser automation (Puppeteer) for JS-rendered content
- [ ] Add caching layer (avoid re-fetching same URL)
- [ ] Support for PDF/DOCX output formats
- [ ] API server wrapper
- [ ] Browser extension integration
