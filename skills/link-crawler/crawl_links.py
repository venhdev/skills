#!/usr/bin/env python3
"""
Link Crawler — Crawls root URLs to a configurable depth, checks each link
is active (HTTP 200 + non-trivial content), and outputs grouped results as JSON.

Two run modes:

  (1) Single URL — no config file needed:
      python crawl_links.py --url https://example.com [--depth 2] [--topic "My Topic"]

  (2) Config file — multiple topics at once:
      python crawl_links.py --input crawl_config.json [--timeout 10]

config.json format:
{
    "topics": [
        {
            "topic": "Python Docs",
            "root_links": ["https://docs.python.org/3/"],
            "depth": 1
        }
    ],
    "filter_groups": null  // null = all, or ["group1", "keyword"]
}

Output: JSON to stdout with grouped active/dead link lists.
Stderr: structured progress logs (INFO level).
With --investigate: live per-URL event stream + tree view on stderr.
"""

import argparse
import json
import logging
import os
import re
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
from html.parser import HTMLParser
from threading import Lock
from typing import Optional

# ---------------------------------------------------------------------------
# Logging setup
# ---------------------------------------------------------------------------
logging.basicConfig(
    level=logging.INFO,
    format="[%(levelname)s] %(message)s",
    stream=sys.stderr,
)
logger = logging.getLogger("crawl")

# ---------------------------------------------------------------------------
# Real-time investigation logger (separate stream, no buffering)
# ---------------------------------------------------------------------------

class InvestigatorLogger:
    """
    Real-time per-URL event log with live tree view.
    All output goes directly to stderr (unbuffered) so the user sees
    every result the moment it arrives — no batch delays.
    """

    def __init__(self, enabled: bool, tree_enabled: bool) -> None:
        self.enabled = enabled
        self.tree_enabled = tree_enabled
        self._lock = Lock()
        self._url_timestamps: dict[str, float] = {}
        self._start_time = time.monotonic()
        self._last_tree_lines = 0

    def _elapsed_ms(self) -> str:
        """Wall-clock elapsed time since crawl start, in ms."""
        elapsed = (time.monotonic() - self._start_time) * 1000
        return f"{int(elapsed):>6}ms"

    def _short_url(self, url: str, max_len: int = 55) -> str:
        """Truncate URL for readability in tree view."""
        if len(url) <= max_len:
            return url
        # Keep domain + first path segment
        parsed = urllib.parse.urlparse(url)
        domain = parsed.netloc
        path = parsed.path.strip("/")
        segs = path.split("/")
        short = "/".join(segs[:2]) if len(segs) > 2 else path
        result = f"{domain}/{short}"
        return (result[: max_len - 3] + "...") if len(result) > max_len else result

    def _print(self, msg: str) -> None:
        """Unbuffered write to stderr."""
        print(msg, flush=True)

    def url_discovered(self, url: str, depth: int, parent_url: Optional[str] = None) -> None:
        """Called when a new URL is added to the queue."""
        if not self.enabled:
            return
        with self._lock:
            ts = self._elapsed_ms()
            indent = "  " * depth
            self._print(
                f"[DISCOVER] {ts}  {indent}{url}"
            )

    def url_checked(
        self,
        url: str,
        depth: int,
        status: int,
        is_active: bool,
        new_links: int = 0,
        content_len: int = 0,
    ) -> None:
        """Called the instant a URL result arrives (real-time, not batched)."""
        if not self.enabled:
            return
        with self._lock:
            ts = self._elapsed_ms()
            if is_active:
                tag = "[OK]"
                extra = f" | +{new_links} links" if new_links > 0 else ""
                if content_len > 0:
                    extra += f" | {content_len:,} chars"
            else:
                tag = "[DEAD]"
                extra = f" | HTTP {status}"
            indent = "  " * depth
            marker = "✓" if is_active else "✗"
            self._print(
                f"{tag} {ts}  {indent}{marker} {url}{extra}"
            )

            if self.tree_enabled:
                self._print_tree_node(url, depth, is_active, status)

    def _print_tree_node(
        self, url: str, depth: int, is_active: bool, status: int
    ) -> None:
        """Print a single line of the live crawl tree."""
        branch = "│   " if depth > 0 else ""
        prefix = "├── " if depth > 0 else ""
        marker = "✓" if is_active else "✗"
        status_str = f"[{status}]" if not is_active else ""
        self._print(
            f"  {branch}{prefix}{marker} {self._short_url(url)} {status_str}"
        )

    def batch_started(self, batch_num: int, batch_size: int, total_queued: int) -> None:
        """Called when a new batch of URLs is dispatched."""
        if not self.enabled:
            return
        with self._lock:
            ts = self._elapsed_ms()
            self._print(
                f"[BATCH]   {ts}  Starting batch #{batch_num} "
                f"({batch_size} URLs)  queued={total_queued}"
            )

    def batch_complete(
        self, batch_num: int, ok_count: int, dead_count: int, elapsed_s: float
    ) -> None:
        """Called when a batch finishes."""
        if not self.enabled:
            return
        with self._lock:
            ts = self._elapsed_ms()
            self._print(
                f"[BATCH]   {ts}  Batch #{batch_num} done "
                f"| ✓{ok_count}  ✗{dead_count}  | {elapsed_s:.1f}s"
            )

    def crawl_started(self, root_url: str, max_depth: int) -> None:
        """Called when crawling begins for a root URL."""
        if not self.enabled:
            return
        with self._lock:
            self._start_time = time.monotonic()
            self._print("")
            self._print("╔══════════════════════════════════════════════════════════╗")
            self._print(f"║  🔍 INVESTIGATION START  root={root_url}  depth={max_depth}  ║")
            self._print("╠══════════════════════════════════════════════════════════╣")
            self._print("│  Legend: [DISCOVER] new URL queued  [OK] alive  [DEAD] unreachable  │")
            self._print("╠══════════════════════════════════════════════════════════╣")

    def crawl_complete(
        self, root_url: str, visited: int, active: int, dead: int, elapsed_s: float
    ) -> None:
        """Called when crawling finishes for a root URL."""
        if not self.enabled:
            return
        with self._lock:
            self._print("")
            self._print("╠══════════════════════════════════════════════════════════╣")
            self._print(f"║  ✅ INVESTIGATION COMPLETE  {root_url}         ║")
            self._print(f"║  Visited: {visited}  ✓ alive: {active}  ✗ dead: {dead}  | {elapsed_s:.1f}s  ║")
            self._print("╚══════════════════════════════════════════════════════════╝")
            self._print("")

    def separator(self, msg: str) -> None:
        """Print a visual separator between sections."""
        if not self.enabled:
            return
        self._print(f"\n──── {msg} ────\n")


investigator = InvestigatorLogger(enabled=False, tree_enabled=False)


# ---------------------------------------------------------------------------
# HTML parsing
# ---------------------------------------------------------------------------
class LinkExtractor(HTMLParser):
    """Extract href links from HTML content."""

    def __init__(self) -> None:
        super().__init__()
        self.links: list[str] = []

    def handle_starttag(
        self, tag: str, attrs: list[tuple[str, Optional[str]]]
    ) -> None:
        if tag == "a":
            for attr_name, attr_val in attrs:
                if attr_name == "href" and attr_val:
                    self.links.append(attr_val)


# ---------------------------------------------------------------------------
# Content validation
# ---------------------------------------------------------------------------
_ERROR_PATTERNS: list[re.Pattern[str]] = [
    re.compile(r"<title>\s*(404|403|500|502|503|not found|error|page not found|access denied)", re.IGNORECASE),
    re.compile(r"<h1>\s*(404|not found|error|oops|page not found|forbidden)", re.IGNORECASE),
    re.compile(r"the page you.*(looking for|requested).*(not|cannot|doesn.t)", re.IGNORECASE),
    re.compile(r"this page (doesn.t|does not) exist", re.IGNORECASE),
    re.compile(r"nothing (was )?found here", re.IGNORECASE),
    re.compile(r"under construction", re.IGNORECASE),
    re.compile(r"coming soon", re.IGNORECASE),
]


def is_real_content(html_text: str) -> bool:
    """Check if a page has real content (not an empty/error/placeholder page)."""
    if not html_text or len(html_text.strip()) < 256:
        return False

    lower = html_text.lower()

    for pat in _ERROR_PATTERNS:
        if pat.search(lower):
            return False

    # Strip HTML tags and check remaining text length
    text_only = re.sub(r"<[^>]+>", " ", lower)
    text_only = re.sub(r"\s+", " ", text_only).strip()
    return len(text_only) >= 100


# ---------------------------------------------------------------------------
# URL utilities
# ---------------------------------------------------------------------------
def normalize_url(url: str, base_url: str) -> Optional[str]:
    """Resolve relative URLs and normalize. Returns None for non-HTTP URLs."""
    try:
        resolved = urllib.parse.urljoin(base_url, url)
        parsed = urllib.parse.urlparse(resolved)
        if parsed.scheme not in ("http", "https"):
            return None
        return urllib.parse.urlunparse(parsed._replace(fragment=""))
    except Exception:
        return None


def extract_links(html: str, base_url: str) -> list[str]:
    """Extract and normalize all links from HTML."""
    parser = LinkExtractor()
    try:
        parser.feed(html)
    except Exception:
        pass

    results: list[str] = []
    seen: set[str] = set()
    for raw in parser.links:
        norm = normalize_url(raw, base_url)
        if norm and norm not in seen:
            seen.add(norm)
            results.append(norm)
    return results


# ---------------------------------------------------------------------------
# HTTP fetching
# ---------------------------------------------------------------------------
def fetch_page(url: str, timeout: int) -> tuple[int, str]:
    """Fetch a URL. Returns (status_code, body_text). Returns (-1, "") on error."""
    try:
        req = urllib.request.Request(url, headers={
            "User-Agent": "Mozilla/5.0 (compatible; LinkCrawler/1.0)"
        })
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            status = resp.getcode()
            content_type = resp.headers.get("Content-Type", "")
            if "text/html" not in content_type and "text/plain" not in content_type:
                return (status, "<non-html-resource>")
            body = resp.read(500_000).decode("utf-8", errors="replace")
            return (status, body)
    except urllib.error.HTTPError as e:
        return (e.code, "")
    except Exception:
        return (-1, "")


# ---------------------------------------------------------------------------
# Crawler
# ---------------------------------------------------------------------------
def crawl(
    root_url: str,
    max_depth: int,
    timeout: int,
    max_links: int = 500,
) -> dict:
    """
    Crawl starting from root_url up to max_depth levels.
    Returns dict with 'active' and 'dead' link lists.
    Emits real-time investigation events via investigator.log() as each
    result arrives — no batching, no delays.
    """
    visited: dict[str, dict] = {}  # url -> {"status": int, "active": bool}
    to_visit: list[tuple[str, int]] = [(root_url, 0)]
    queued: set[str] = {root_url}

    batch_num = 0
    batch_size = 20
    max_workers = 10

    crawl_start = time.monotonic()

    investigator.crawl_started(root_url, max_depth)

    logger.info("Starting crawl  root=%s  depth=%d  max_links=%d", root_url, max_depth, max_links)

    # ThreadPoolExecutor created once outside the loop (was recreated every batch)
    with ThreadPoolExecutor(max_workers=max_workers) as pool:
        while to_visit and len(visited) < max_links:
            batch_num += 1
            batch = to_visit[:batch_size]
            to_visit = to_visit[batch_size:]

            investigator.batch_started(batch_num, len(batch), len(queued))

            batch_start = time.monotonic()
            futures = {
                pool.submit(fetch_page, url, timeout): (url, depth)
                for url, depth in batch
            }

            for fut in as_completed(futures):
                url, depth = futures[fut]
                try:
                    status, body = fut.result()
                except Exception:
                    status, body = -1, ""

                is_active = (
                    status == 200
                    and (body == "<non-html-resource>" or is_real_content(body))
                )
                visited[url] = {"status": status, "active": is_active}

                # ── Real-time investigation event: fires the moment result arrives ──
                new_links_count = 0
                content_len = 0
                if is_active and depth < max_depth and body != "<non-html-resource>":
                    children = extract_links(body, url)
                    new_links_count = sum(
                        1 for c in children if c not in queued and len(queued) < max_links
                    )
                    content_len = len(body)

                investigator.url_checked(
                    url=url,
                    depth=depth,
                    status=status,
                    is_active=is_active,
                    new_links=new_links_count,
                    content_len=content_len,
                )

                # Queue new links
                if is_active and depth < max_depth and body != "<non-html-resource>":
                    children = extract_links(body, url)
                    logger.debug(
                        "  [links] extracted %d links from %s", len(children), url
                    )
                    new_count = 0
                    for child in children:
                        if child not in queued and len(queued) < max_links:
                            queued.add(child)
                            to_visit.append((child, depth + 1))
                            new_count += 1
                            investigator.url_discovered(child, depth + 1, parent_url=url)

                    if new_count:
                        logger.info(
                            "  [queue] +%d new URLs queued  (depth=%d)  total_queued=%d",
                            new_count, depth + 1, len(queued),
                        )

            batch_elapsed = time.monotonic() - batch_start
            ok_count = sum(1 for u in visited if visited[u]["active"])
            investigator.batch_complete(
                batch_num, ok_count, len(visited) - ok_count, batch_elapsed
            )

            if to_visit:
                time.sleep(0.2)

    crawl_elapsed = time.monotonic() - crawl_start
    active = [url for url, info in visited.items() if info["active"]]
    dead = [
        {"url": url, "status": info["status"]}
        for url, info in visited.items()
        if not info["active"]
    ]

    investigator.crawl_complete(
        root_url, len(visited), len(active), len(dead), crawl_elapsed
    )

    logger.info(
        "Crawl complete  visited=%d  active=%d  dead=%d  queued_remaining=%d",
        len(visited), len(active), len(dead), len(to_visit),
    )

    return {"active": active, "dead": dead}


# ---------------------------------------------------------------------------
# Deduplication helpers
# ---------------------------------------------------------------------------
def deduplicate_urls(urls: list[str]) -> list[str]:
    seen: set[str] = set()
    result: list[str] = []
    for u in urls:
        if u not in seen:
            seen.add(u)
            result.append(u)
    return result


def deduplicate_dead(dead: list[dict]) -> list[dict]:
    """Deduplicate dead entries by URL, keeping the entry with the worst status."""
    best: dict[str, dict] = {}
    for entry in dead:
        url = entry["url"]
        status = entry["status"]
        if url not in best or status < best[url]["status"]:
            best[url] = entry
    return list(best.values())


# ---------------------------------------------------------------------------
# Config builder (single-URL shortcut → internal config dict)
# ---------------------------------------------------------------------------
def build_config(args: argparse.Namespace) -> dict:
    """
    Build internal config dict from CLI args.

    Mode A — single URL (--url):   build one topic from --url / --depth / --topic
    Mode B — config file (--input): load topics from JSON
    """
    if args.url:
        return {
            "topics": [
                {
                    "topic": args.topic or _url_to_topic(args.url),
                    "root_links": [args.url],
                    "depth": args.depth,
                }
            ],
            "filter_groups": None,
        }

    with open(args.input) as f:
        return json.load(f)


def _url_to_topic(url: str) -> str:
    """Derive a topic label from a URL for single-URL mode."""
    parsed = urllib.parse.urlparse(url)
    path = parsed.path.strip("/").replace("/", " › ")
    label = path if path else parsed.netloc
    return label if label else "Crawl Result"


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------
def main() -> None:
    parser = argparse.ArgumentParser(
        description="Crawl links and check activity",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Run modes (mutually exclusive):

  Single URL (no config file):
    python crawl_links.py --url https://example.com [--depth 2] [--topic "My Topic"]

  Config file (multiple topics):
    python crawl_links.py --input crawl_config.json

Output flags:
  --output FILE   Write JSON results to FILE instead of stdout
  --clean         Delete --output FILE before writing (fresh run)
  --append        Append JSON results to --output FILE instead of overwriting
                  (ignored if --output is not given)

  Tip: combine with shell redirect for full control:
    python crawl_links.py --url https://example.com > results.json 2>&1
    python crawl_links.py --url https://example.com --investigate --output results.json
        """,
    )

    # Run mode (mutually exclusive)
    mode = parser.add_argument_group("run mode (pick one)")
    mode_ex = mode.add_mutually_exclusive_group(required=True)
    mode_ex.add_argument(
        "--input", metavar="FILE",
        help="Path to a config JSON file (multi-topic mode)",
    )
    mode_ex.add_argument(
        "--url", metavar="URL",
        help="Crawl a single URL directly (single-topic mode, no config file needed)",
    )

    # Single-URL options
    single = parser.add_argument_group("single-URL options (used with --url)")
    single.add_argument(
        "--depth", type=int, default=1,
        help="Crawl depth for --url mode (default: 1, max recommended: 3)",
    )
    single.add_argument(
        "--topic", metavar="NAME",
        help="Topic name used as the group key in JSON output (default: derived from URL path)",
    )

    # Output options
    output_grp = parser.add_argument_group("output options")
    output_grp.add_argument(
        "--output", "-o", metavar="FILE",
        help="Write JSON results to FILE instead of stdout",
    )
    output_grp.add_argument(
        "--clean", action="store_true",
        help="Delete --output FILE before writing (fresh run, no stale data)",
    )
    output_grp.add_argument(
        "--append", action="store_true",
        help="Append results to --output FILE instead of overwriting "
             "(each append is a separate top-level entry with topic+timestamp)",
    )

    # General options
    parser.add_argument(
        "--timeout", type=int, default=10,
        help="HTTP request timeout in seconds (default: 10)",
    )
    parser.add_argument(
        "--verbose", "-v", action="store_true",
        help="Enable DEBUG-level logging (extracted link counts, queue sizes)",
    )
    parser.add_argument(
        "--investigate", action="store_true",
        help="Enable real-time investigation log: live per-URL events "
             "printed directly to stderr (no buffering)",
    )
    parser.add_argument(
        "--tree", action="store_true",
        help="Include a live crawl-tree view in the investigation log "
             "(implies --investigate)",
    )
    args = parser.parse_args()

    if args.verbose:
        logging.getLogger("crawl").setLevel(logging.DEBUG)

    global investigator
    investigator = InvestigatorLogger(enabled=args.investigate, tree_enabled=args.tree)

    # ── Output file setup ──
    out_file: Optional[str] = None
    out_mode = "w"
    if args.output:
        out_file = args.output
        if args.clean:
            if os.path.exists(out_file):
                os.remove(out_file)
                logger.info("Removed existing output file: %s", out_file)
        if args.append:
            out_mode = "a"

    # ── Config ──
    if args.url:
        logger.info("=== Link Crawler started (single-URL mode) ===")
        logger.info("URL: %s  depth=%d  timeout=%ds", args.url, args.depth, args.timeout)
    else:
        logger.info("=== Link Crawler started ===")
        logger.info("Config: %s  timeout=%ds", args.input, args.timeout)

    start_time = time.monotonic()
    config = build_config(args)

    results: dict = {}
    total_active = 0
    total_dead = 0

    topic_count = len(config["topics"])
    for idx, topic_cfg in enumerate(config["topics"], 1):
        topic = topic_cfg["topic"]
        depth = topic_cfg.get("depth", 1)

        investigator.separator(f"Topic {idx}/{topic_count}: {topic}")
        logger.info("")
        logger.info(">>> Topic: %s  (depth=%d)", topic, depth)

        all_active: list[str] = []
        all_dead: list[dict] = []

        for root_url in topic_cfg["root_links"]:
            logger.info("[crawling] %s -> %s  (depth=%d)", topic, root_url, depth)
            data = crawl(root_url, max_depth=depth, timeout=args.timeout)
            all_active.extend(data["active"])
            all_dead.extend(data["dead"])

        # Deduplicate results for this topic
        deduped_active = deduplicate_urls(all_active)
        deduped_dead = deduplicate_dead(all_dead)

        logger.info(
            "Topic '%s' done: %d active  %d dead  (deduped from %d/%d raw)",
            topic, len(deduped_active), len(deduped_dead),
            len(set(all_active)), len(set(e["url"] for e in all_dead)),
        )

        results[topic] = {
            "active_count": len(deduped_active),
            "dead_count": len(deduped_dead),
            "active_links": deduped_active,
            "dead_links": deduped_dead,
        }
        total_active += len(deduped_active)
        total_dead += len(deduped_dead)

    # Apply group filter if specified
    filter_groups = config.get("filter_groups")
    if filter_groups:
        logger.info("Applying filter_groups: %s", filter_groups)
        filtered = {}
        for key in results:
            if any(fg.lower() in key.lower() for fg in filter_groups):
                filtered[key] = results[key]
        results = filtered if filtered else results

    elapsed = time.monotonic() - start_time

    logger.info("")
    logger.info("=== Crawl complete ===")
    logger.info("  Total visited : %d", total_active + total_dead)
    logger.info("  Active links  : %d", total_active)
    logger.info("  Dead links    : %d", total_dead)
    logger.info("  Elapsed time  : %.1fs", elapsed)

    # ── Write output ──
    if out_file:
        if args.append:
            # Wrap each run in a timestamped envelope so multiple appends are distinguishable
            envelope = {
                "run_at": datetime.now().isoformat(timespec="seconds"),
                "elapsed_s": round(elapsed, 2),
                "topics": results,
            }
            with open(out_file, out_mode) as f:
                if out_mode == "a":
                    f.write("\n")
                json.dump(envelope, f, indent=2)
            logger.info("Appended results to: %s", out_file)
        else:
            with open(out_file, "w") as f:
                json.dump(results, f, indent=2)
            logger.info("Saved results to: %s", out_file)
    else:
        json.dump(results, sys.stdout, indent=2)


if __name__ == "__main__":
    main()