---
name: url-to-markdown
description: Use when the user has one or more URLs and wants them converted to clean Markdown files — for documentation, research archival, LLM context preparation, or offline content preservation. Triggers on phrases like "convert these URLs to markdown", "save these articles as MD", "turn this link list into md files", "extract articles from a CSV of URLs".
---

# URL to Markdown

Convert URLs to clean Markdown using [defuddle](https://github.com/kepano/defuddle).

## When to Use

- User provides a list of URLs and asks for Markdown files
- User wants to archive web content for offline reading
- User is preparing LLM context from web sources
- User has a `.txt` / `.csv` / `.json` file of URLs to process in batch
- User wants page-structure maps or stripped output (no images/links)

**Don't use for:** password-protected pages, JS-heavy SPAs (JSDOM can't render), HTML→HTML conversion.

## Quick Reference

| Task | Command |
|---|---|
| Single URL | `node scripts/cli.js https://example.com` |
| From file | `node scripts/cli.js --input urls.txt` |
| Custom output | `node scripts/cli.js <url> --output my.md` |
| Page-structure map | `node scripts/cli.js <url> --page-map true` |
| Locale date format | `node scripts/cli.js <url> --date-format locale` |
| Strip images/links | `node scripts/cli.js <url> --keep-images false --keep-links false` |

Full flag list: `node scripts/cli.js --help`

## Output

```markdown
# Article Title

_Author_

_Date_

---

[Markdown body]

---

**Source:** https://...
```

## Common Mistakes

- **`Module not found`** → run `pnpm add defuddle @mozilla/readability turndown turndown-plugin-gfm jsdom` (or `npm install`)
- **Empty Markdown** → page is JS-rendered; JSDOM can't execute scripts
- **HTTP 429** → reduce `--batch-size` to 1-2
- **Wrong date format** → use `--date-format locale` for human-readable dates
- **Library: `result.byline` is undefined** → field renamed to `result.author` in v2 (see ARCHITECTURE.md § Field Name Mapping)

## See Also

- `README.md` — full quick start, examples, troubleshooting
- `references/ARCHITECTURE.md` — pipeline, dependencies, customization points, v1→v2 field mapping
- `node scripts/cli.js --help` — all CLI flags
