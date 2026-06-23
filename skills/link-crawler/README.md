# 🔗 Link Crawler

![Python](https://img.shields.io/badge/Python-3.10+-4ade80?style=flat-square)
![Standard Lib](https://img.shields.io/badge/stdlib-only-4ade80?style=flat-square)
![No Deps](https://img.shields.io/badge/no%20deps-yes-4ade80?style=flat-square)
![Depth 1–3](https://img.shields.io/badge/depth-1%E2%80%933-4ade80?style=flat-square)

Crawl root URLs to a configurable depth, verify each discovered link is active (HTTP 200 + real page content), and output results as grouped JSON.

> Icon: `icon.svg`

---

## Quick Start

```bash
python3 crawl_links.py --input crawl_config.json --timeout 10 > results.json
python3 crawl_links.py --url https://example.com --depth 2 --investigate
```

The first crawls multi-topic from a config file; the second is a one-off against a single URL with per-URL stderr logging.

---

## Two Run Modes

| Mode | Command | When to use |
|------|---------|-------------|
| **Config file** | `--input crawl_config.json` | Multiple topics at once |
| **Single URL** | `--url https://example.com` | One-off crawl, no config file needed |

The topic label in the JSON output defaults to the URL itself when `--topic` is not given.

---

## CLI Flags

### Run Mode (pick one, required)

| Flag | Description |
|------|-------------|
| `--input <file>` | Path to a config JSON file (multi-topic mode) |
| `--url <url>` | Crawl a single URL directly (single-topic mode, no config file) |

### Single-URL Options (used with `--url`)

| Flag | Description |
|------|-------------|
| `--depth <n>` | Crawl depth (default: `1`, max recommended: `3`) |
| `--topic <name>` | Topic label for JSON output key (default: the URL itself) |

### Output Options

| Flag | Description |
|------|-------------|
| `--output`, `-o <file>` | Write JSON results to FILE instead of stdout |
| `--investigate` | Stream per-URL events to stderr in real time |

### General Options

| Flag | Description |
|------|-------------|
| `--timeout <sec>` | HTTP request timeout in seconds (default: `10`) |

---

## Config File Format

Create a `crawl_config.json`:

```json
{
    "topics": [
        {
            "topic": "My Topic",
            "root_links": ["https://example.com/start"],
            "depth": 2
        }
    ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `topic` | string | Label for this group of links in the output |
| `root_links` | array of string | One or more starting URLs to crawl from |
| `depth` | integer | How many link hops deep to follow (default: `1`, max recommended: `3`) |

### Depth Reference

| Depth | What is crawled |
|-------|----------------|
| `1` | Only links found directly on the root page(s) |
| `2` | Root page(s) + all pages linked from them |
| `3` | Root → level 1 → level 2 pages (recommended max) |

> **Warning:** Depth 4+ can produce thousands of URLs and take a very long time. The script caps at 500 links per root URL to prevent runaway crawls.

---

## Output Format

```json
{
  "My Topic": {
    "active_count": 42,
    "dead_count": 3,
    "active_links": [
      "https://example.com/page1",
      "https://example.com/page2"
    ],
    "dead_links": [
      {"url": "https://example.com/broken", "status": 404}
    ]
  }
}
```

Progress and investigation logs go to **stderr**.

---

## Investigation Log (`--investigate`)

The investigation log streams one line per URL directly to stderr the moment each result arrives — no batching, no buffering:

```
[OK] depth=0 https://docs.nestjs.com/cli/overview +14
[OK] depth=1 https://docs.nestjs.com/cli/workspaces
[DEAD] depth=1 https://docs.nestjs.com/missing HTTP 404
```

**Format:** `[OK|DEAD] depth=<N> <url> [+<new>] [HTTP <status>]`

- `[OK]` — URL returned HTTP 200 with real content
- `[DEAD]` — URL is unreachable, returned an error, or is a placeholder page
- `+N` — number of new links queued from this page (only on `[OK]` for crawlable pages)
- `HTTP N` — the HTTP status code (only on `[DEAD]`)

---

## Architecture

```
crawl_links.py
├── log_event()        — Per-URL stderr event log (one line per URL)
├── LinkExtractor      — HTML link extraction (stdlib HTMLParser)
├── fetch()            — HTTP GET with timeout, follows same-host redirects
├── is_real_content()  — Content quality check (length + error-pattern regex)
├── extract_links()    — HTML parsing + URL normalization
├── crawl()            — BFS crawler with ThreadPoolExecutor
└── main()             — CLI, config loading, output
```

### Concurrency

- **10 concurrent threads** per batch
- **Batch size:** 20 URLs per dispatch
- **Max links per root:** 500 (prevents runaway crawls)

### Content Validation

A URL is marked **active** only when ALL of these are true:
1. HTTP status is `200`
2. Response `Content-Type` is `text/html`, `text/plain`, or `application/xhtml+xml`
3. Raw HTML body is at least 256 bytes
4. No error/placeholder patterns match (404, "not found", "under construction", etc.)
5. After stripping HTML tags, at least 100 characters of readable text remain

---

## Tips

- **Start with depth 1** to get a quick feel for link density, then increase.
- Use **`--timeout 30`** for sites with slow responses.
- Use **`--investigate`** when debugging crawl coverage.
- Post-process the JSON with `jq` (e.g., `jq '.[].active_links | length'`).
- Capture results and logs together with shell redirect: `... > results.json 2>&1`.

---

## Requirements

- Python 3.10+
- Standard library only (`urllib`, `html.parser`, `concurrent.futures`)
- No external dependencies