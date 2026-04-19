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
# Basic crawl (progress to stderr, JSON results to stdout)
python3 crawl_links.py --input crawl_config.json --timeout 10 > results.json

# With real-time investigation log
python3 crawl_links.py --input crawl_config.json --investigate --tree

# Verbose debug (show extracted link counts, queue sizes)
python3 crawl_links.py --input crawl_config.json --verbose
```

---

## Two Run Modes

| Mode | Command | When to use |
|------|---------|-------------|
| **Config file** | `--input crawl_config.json` | Multiple topics at once |
| **Single URL** | `--url https://example.com` | One-off crawl, no config file needed |

### Single-URL Mode (no config file)

```bash
# Simplest form — one URL, depth 1, topic auto-derived from path
python3 crawl_links.py --url https://example.com

# Depth 2, custom topic label
python3 crawl_links.py --url https://example.com --depth 2 --topic "Example Docs"

# With investigation log
python3 crawl_links.py --url https://docs.nestjs.com/cli/overview --depth 3 --investigate --tree

# Save to file instead of stdout
python3 crawl_links.py --url https://example.com --output results.json --clean
```

The topic label is auto-derived from the URL path (e.g. `docs.nestjs.com/cli` → `cli`), so the JSON output key is meaningful without you having to think of a name.

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
| `--topic <name>` | Topic label for JSON output key (default: derived from URL path) |

### Output Options

| Flag | Description |
|------|-------------|
| `--output`, `-o <file>` | Write JSON results to FILE instead of stdout |
| `--clean` | Delete `--output FILE` before writing (fresh run, no stale data) |
| `--append` | Append results to `--output FILE` as a timestamped envelope (run history) |

> **`--clean`** is useful when you want a guaranteed-clean output file before a new crawl — it removes any existing file first.
>
> **`--append`** is useful for running the same crawl repeatedly and keeping a full history in one file (each run is a separate JSON object with `run_at` timestamp).

### General Options

| Flag | Description |
|------|-------------|
| `--timeout <sec>` | HTTP request timeout in seconds (default: `10`) |
| `-v`, `--verbose` | Enable DEBUG-level logging (extracted link counts, queue sizes) |
| `--investigate` | Enable real-time per-URL event stream to stderr (see below) |
| `--tree` | Add a live crawl-tree view to the investigation log (implies `--investigate`) |

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
    ],
    "filter_groups": null
}
```

| Field | Type | Description |
|-------|------|-------------|
| `topic` | string | Label for this group of links in the output |
| `root_links` | array of string | One or more starting URLs to crawl from |
| `depth` | integer | How many link hops deep to follow (default: `1`, max recommended: `3`) |
| `filter_groups` | array or `null` | If non-null, only include topics whose name contains any of these strings (case-insensitive) |

### Depth Reference

| Depth | What is crawled |
|-------|----------------|
| `1` | Only links found directly on the root page(s) |
| `2` | Root page(s) + all pages linked from them |
| `3` | Root → level 1 → level 2 pages (recommended max) |

> **Warning:** Depth 4+ can produce thousands of URLs and take a very long time. The script caps at 500 links per root URL to prevent runaway crawls.

---

## Output Format

### Normal mode (stdout or `--output`)

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

### Append mode (`--append --output FILE`)

Each append writes a separate JSON object on its own line, making it valid JSONL (JSON Lines):

```json
{"run_at": "2026-04-17T14:30:00", "elapsed_s": 8.3, "topics": {...}}
{"run_at": "2026-04-17T15:10:00", "elapsed_s": 7.9, "topics": {...}}
```

Progress and investigation logs go to **stderr** and are printed in real time.

---

## Investigation Log (`--investigate`)

The investigation log streams every event directly to stderr the moment it happens — no batching, no buffering. It shows:

- **`[DISCOVER]`** — when a new URL is added to the crawl queue
- **`[OK]`** — URL returned HTTP 200 with real content (shows `+N links` and char count)
- **`[DEAD]`** — URL is unreachable, returned an error, or is a placeholder page
- **`[BATCH]`** — batch dispatch and completion with timing
- **`[INVESTIGATION START / COMPLETE]`** — section header with summary stats

### With `--tree`

Adds a live crawl-tree showing the URL hierarchy as it's discovered:

```
╔══════════════════════════════════════════════════════════╗
║  🔍 INVESTIGATION START  root=https://docs.nestjs.com  ║
╠══════════════════════════════════════════════════════════╣
│  Legend: [DISCOVER] new URL queued  [OK] alive  [DEAD] unreachable  │
╠══════════════════════════════════════════════════════════╣
[DISCOVER]   0ms    https://docs.nestjs.com/cli/overview
[OK]         312ms    ✓ https://docs.nestjs.com/cli/overview | +14 links | 4,230 chars
  ├── ✓ docs.nestjs.com/cli/workspaces [200]
  ├── ✓ docs.nestjs.com/cli/usages [200]
[DEAD]       450ms    ✗ https://docs.nestjs.com/missing [404] | HTTP 404
[BATCH]      520ms    Batch #1 done | ✓12  ✗1  | 0.5s
...
```

### Example: Normal Run

```
[INFO] === Link Crawler started ===
[INFO] Config: crawl_config.json  timeout=10s
[INFO]
[INFO] >>> Topic: NestJS CLI & Monorepo  (depth=3)
[INFO] [crawling] NestJS CLI & Monorepo -> https://docs.nestjs.com/cli/overview  (depth=3)
[INFO] Starting crawl  root=https://docs.nestjs.com/cli/overview  depth=3  max_links=500
[INFO] [batch 1] processing 20 URLs  (queued=1  visited=0)
[INFO]   [OK]   https://docs.nestjs.com/cli/overview  (depth=0  status=200)
[INFO]   [queue] +14 new URLs queued  (depth=1)  total_queued=15
[INFO] Topic 'NestJS CLI & Monorepo' done: 142 active  7 dead  (deduped from 142/7 raw)
[INFO]
[INFO] === Crawl complete ===
[INFO]   Total visited : 149
[INFO]   Active links  : 142
[INFO]   Dead links    : 7
[INFO]   Elapsed time  : 8.3s
```

---

## Architecture

```
crawl_links.py
├── crawl()             — BFS crawler with ThreadPoolExecutor
│   ├── fetch_page()         — HTTP GET with timeout
│   ├── extract_links()      — HTML parsing + URL normalization
│   └── is_real_content()   — Content quality check
├── InvestigatorLogger      — Real-time event streamer
├── build_config()          — CLI args → internal config dict
└── main()              — CLI, config loading, deduplication
```

### Concurrency

- **10 concurrent threads** per batch
- **Batch size:** 20 URLs per dispatch
- **Polite delay:** 200ms between batches
- **Max links per root:** 500 (prevents runaway crawls)

### Content Validation

A URL is marked **active** only when ALL of these are true:
1. HTTP status is `200`
2. Response `Content-Type` is `text/html` or `text/plain`
3. Raw HTML body is at least 256 bytes
4. No error/placeholder patterns match (404, "not found", "under construction", etc.)
5. After stripping HTML tags, at least 100 characters of readable text remain

Non-HTML resources (PDFs, images, etc.) that return `200` are counted as active but **not** crawled deeper.

---

## Deduplication

- Active links are deduplicated across all root URLs within a topic
- Dead links are deduplicated by URL, keeping the entry with the worst HTTP status code
- Duplicate child links discovered via multiple parents are coalesced automatically

---

## Tips

- **Start with depth 1** to get a quick feel for link density, then increase
- Use **`--timeout 30`** for sites with slow responses
- Use **`--investigate --tree`** when debugging crawl coverage or investigating dead links
- Use **`filter_groups`** in the config to focus on specific sub-sections of a large site
- To capture both results and logs in one command:

```bash
python3 crawl_links.py --input crawl_config.json --investigate > results.json 2>&1
```

- **Single-URL mode** is ideal for quick checks without writing a config file:

```bash
python3 crawl_links.py --url https://example.com --depth 2 --investigate --output results.json --clean
```

---

## Requirements

- Python 3.10+
- Standard library only (`urllib`, `html.parser`, `concurrent.futures`, `logging`)
- No external dependencies
