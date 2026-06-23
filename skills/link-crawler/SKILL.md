---
name: link-crawler
icon: icon.svg
description: Use when the user asks to crawl, harvest, audit, or verify URLs from a website or a list — wants active vs dead links, page inventory, link validation, or a sitemap/link-tree scrape. Also trigger on casual phrasing like "grab all links", "check which URLs work", or "find pages on this site".
---

# Link Crawler

Crawl root URLs to a configurable depth, verify each discovered link is active (HTTP 200 + real content), and present results as a grouped markdown list.

## When to Use

- User asks to find all active pages on a site or section.
- User provides a list of URLs and wants to know which are alive.
- User wants a page inventory / sitemap / link tree scraped and reported.

## When NOT to Use

- Fetching a single page's contents → just use a `curl`/`WebFetch`.
- Downloading a specific file (PDF, image, archive).
- Browsing requires login / authenticated session.
- The user wants to test if a site is up (one URL, no crawl) — just curl it.

## Quick Reference

| Step | What to do | Output |
|------|------------|--------|
| 1 | Get topics + URLs + depth (or extract from user's message) | Inputs |
| 2 | Run `crawl_links.py` (single-URL or config-file mode) | JSON to stdout/file |
| 3 | Format the JSON as a grouped markdown list per topic | Markdown in chat |
| 4 | Offer save / deeper crawl / show dead links | Follow-up question |

The script caps at 500 links per root URL. Max recommended depth: 3.

## Workflow

### 1. Gather Inputs

Ask for **topics + URLs** and **depth** (default 1). Example input from user:

```
Topic: Python Docs
- https://docs.python.org/3/library/
- https://docs.python.org/3/tutorial/

Topic: FastAPI
- https://fastapi.tiangolo.com/
```

If the user already provided topics and links in their message, skip asking.

### 2. Run the Crawler

Two modes — pick by number of topics.

**Single URL** (one-off, no config file):
```bash
python3 <skill-dir>/crawl_links.py --url https://example.com --depth 2 --investigate --output results.json
```

**Config file** (multi-topic — write `crawl_config.json` first):
```json
{"topics": [{"topic": "Name", "root_links": ["https://example.com/start"], "depth": 1}]}
```
```bash
python3 <skill-dir>/crawl_links.py --input <skill-dir>/crawl_config.json --timeout 10
```

Flags: `--output/-o FILE` (write JSON to file), `--investigate` (per-URL stderr log), `--timeout N` (default 10s), `--depth N` (default 1), `--topic NAME` (--url mode only).

### 3. Present Results

Read the JSON output and format as a grouped markdown list:

```
### Topic Name
**X active links found** (Y dead/unreachable)

- https://example.com/page1
- https://example.com/page2
...
```

List only active links by default. If more than 50, show the first 50 and note how many more exist.

### 4. Offer Follow-up

After presenting results, ask if the user wants to save the file, crawl deeper, or see the dead links.

## Common Mistakes

- ❌ Don't auto-install any dependency — the script is stdlib-only.
- ❌ Don't list dead links unless the user asks.
- ❌ Don't skip the active/dead counts in the summary line.
- ❌ Don't run the script from outside its directory — use the `<skill-dir>` placeholder in the examples above (replace with the actual path).

## Edge Cases (actionable)

- Many timeouts → suggest raising `--timeout`.
- Many 403s → tell the user the site may be blocking crawlers.
- Zero links from a root → verify the URL is accessible and HTML.