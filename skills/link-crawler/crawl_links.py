#!/usr/bin/env python3
"""
Link Crawler — Crawls root URLs to a configurable depth, checks each link
is active (HTTP 200 + non-trivial content), and outputs grouped JSON.

Two run modes:

  (1) Single URL:
      python crawl_links.py --url https://example.com [--depth 2] [--topic "My Topic"]

  (2) Config file:
      python crawl_links.py --input crawl_config.json

JSON to stdout. Per-URL events to stderr with --investigate.
"""

import argparse
import json
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from html.parser import HTMLParser

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
MAX_BODY_BYTES = 500_000
MAX_LINKS = 500
BATCH_SIZE = 20
MAX_WORKERS = 10
MAX_REDIRECTS = 5
MIN_BODY_BYTES = 256
MIN_TEXT_CHARS = 100

# ---------------------------------------------------------------------------
# Pre-compiled regexes
# ---------------------------------------------------------------------------
_ERROR_RE = re.compile(
    r"<title>\s*(?:404|403|500|502|503|not found|error|page not found|access denied)"
    r"|<h1>\s*(?:404|not found|error|oops|page not found|forbidden)"
    r"|the page you.*(?:looking for|requested).*(?:not|cannot|doesn.t)"
    r"|this page (?:doesn.t|does not) exist"
    r"|nothing (?:was )?found here"
    r"|under construction"
    r"|coming soon",
    re.IGNORECASE,
)
_TAG_AND_WS_RE = re.compile(r"<[^>]+>|\s+")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
def is_html_content_type(ct: str) -> bool:
    primary = ct.split(";", 1)[0].strip().lower()
    return primary in ("text/html", "text/plain", "application/xhtml+xml")


def log_event(url: str, active: bool, depth: int, new: int, status: int) -> None:
    tag = "[OK]" if active else "[DEAD]"
    extra = f" +{new}" if new else ""
    status_extra = f" HTTP {status}" if not active else ""
    print(f"{tag} depth={depth} {url}{extra}{status_extra}", file=sys.stderr, flush=True)


# ---------------------------------------------------------------------------
# HTML link extraction
# ---------------------------------------------------------------------------
class LinkExtractor(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.links: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag == "a":
            for k, v in attrs:
                if k == "href" and v:
                    self.links.append(v)


# ---------------------------------------------------------------------------
# Content validation
# ---------------------------------------------------------------------------
def is_real_content(html: str) -> bool:
    """True if html has real content (not empty/error/placeholder)."""
    if len(html) < MIN_BODY_BYTES:
        return False
    if _ERROR_RE.search(html):
        return False
    text = _TAG_AND_WS_RE.sub("", html)
    return len(text) >= MIN_TEXT_CHARS


# ---------------------------------------------------------------------------
# URL utilities
# ---------------------------------------------------------------------------
def extract_links(html: str, base: str) -> list[str]:
    """Extract and normalize all links from HTML. Deduped, order-preserved."""
    parser = LinkExtractor()
    parser.feed(html)
    out: list[str] = []
    for href in parser.links:
        joined = urllib.parse.urljoin(base, href)
        if urllib.parse.urlparse(joined).scheme in ("http", "https"):
            out.append(urllib.parse.urldefrag(joined).url)
    return list(dict.fromkeys(out))


# ---------------------------------------------------------------------------
# HTTP fetching
# ---------------------------------------------------------------------------
class SafeRedirectHandler(urllib.request.HTTPRedirectHandler):
    """Refuse redirects beyond MAX_REDIRECTS or to a different hostname."""

    def __init__(self, original_host: str, max_redirects: int) -> None:
        super().__init__()
        self.original_host = original_host
        self.max_redirects = max_redirects
        self.count = 0

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        self.count += 1
        if self.count > self.max_redirects:
            return None  # too many hops
        if urllib.parse.urlparse(newurl).hostname != self.original_host:
            return None  # cross-host refused
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def fetch(url: str, timeout: int) -> tuple[int, str, bool, str]:
    """Fetch URL following redirects.

    Returns (status, body, is_html, final_url). final_url is the URL after any
    redirects that were followed. Status is -1 when a redirect was blocked
    (too many hops or cross-host) or on network/other error.
    """
    original_host = urllib.parse.urlparse(url).hostname
    handler = SafeRedirectHandler(original_host, MAX_REDIRECTS)
    opener = urllib.request.build_opener(handler)
    try:
        req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0 (compatible; LinkCrawler/1.0)"})
        with opener.open(req, timeout=timeout) as r:
            ct = r.headers.get("Content-Type", "")
            is_html = is_html_content_type(ct)
            body = r.read(MAX_BODY_BYTES).decode("utf-8", errors="replace") if is_html else ""
            return r.getcode(), body, is_html, r.url
    except urllib.error.HTTPError as e:
        # If we blocked a redirect, count > 0; surface as -1. Otherwise report the
        # server's actual error code (e.g. 404).
        return (-1, "", False, url) if handler.count > 0 else (e.code, "", False, url)
    except Exception:
        return -1, "", False, url


# ---------------------------------------------------------------------------
# Crawler
# ---------------------------------------------------------------------------
def crawl(root: str, depth: int, timeout: int, investigate: bool) -> tuple[list[str], list[dict]]:
    """BFS crawl from root. Returns (active_urls, dead_entries)."""
    active: list[str] = []
    dead: list[dict] = []
    queue: list[tuple[str, int]] = [(root, 0)]
    seen: set[str] = {root}

    with ThreadPoolExecutor(max_workers=MAX_WORKERS) as pool:
        while queue and len(active) + len(dead) < MAX_LINKS:
            remaining = MAX_LINKS - len(active) - len(dead)
            batch = queue[:min(BATCH_SIZE, remaining)]
            queue = queue[len(batch):]
            futures = {pool.submit(fetch, u, timeout): (u, d) for u, d in batch}

            for fut in as_completed(futures):
                url, d = futures[fut]
                status, body, is_html, final_url = fut.result()
                is_active = status == 200 and (not is_html or is_real_content(body))

                new_count = 0
                if is_active:
                    active.append(url)
                    if is_html and d < depth:
                        for c in extract_links(body, final_url):
                            if c not in seen and len(active) + len(dead) + len(queue) < MAX_LINKS:
                                seen.add(c)
                                queue.append((c, d + 1))
                                new_count += 1
                else:
                    dead.append({"url": url, "status": status})

                if investigate:
                    log_event(url, is_active, d, new_count, status)

    return active, dead


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------
def main() -> None:
    ap = argparse.ArgumentParser(
        description="Crawl links and check activity.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Run modes (mutually exclusive):

  Single URL:
    python crawl_links.py --url https://example.com [--depth 2] [--topic "My Topic"]

  Config file (multi-topic):
    python crawl_links.py --input crawl_config.json

Output:
  --output FILE   Write JSON results to FILE instead of stdout
  --investigate   Stream per-URL events to stderr in real time
        """,
    )
    mode = ap.add_mutually_exclusive_group(required=True)
    mode.add_argument("--input", metavar="FILE", help="Config JSON file (multi-topic)")
    mode.add_argument("--url", metavar="URL", help="Single URL to crawl")

    ap.add_argument("--depth", type=int, default=1, help="Crawl depth (default: 1)")
    ap.add_argument("--topic", metavar="NAME", help="Topic label for JSON key (--url mode)")
    ap.add_argument("--timeout", type=int, default=10, help="HTTP timeout in seconds (default: 10)")
    ap.add_argument("--output", "-o", metavar="FILE", help="Write JSON to FILE")
    ap.add_argument("--investigate", action="store_true", help="Per-URL event log to stderr")
    args = ap.parse_args()

    # Build config
    if args.url:
        config = {"topics": [{"topic": args.topic or args.url, "root_links": [args.url], "depth": args.depth}]}
    else:
        with open(args.input, encoding="utf-8") as f:
            config = json.load(f)

    results: dict = {}
    for tc in config.get("topics", []):
        topic = tc["topic"]
        depth = tc.get("depth", 1)
        active: list[str] = []
        dead: list[dict] = []
        for root in tc.get("root_links", []):
            a, d = crawl(root, depth, args.timeout, args.investigate)
            active.extend(a)
            dead.extend(d)
        results[topic] = {
            "active_count": len(active),
            "dead_count": len(dead),
            "active_links": active,
            "dead_links": dead,
        }

    out = json.dumps(results, indent=2)
    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(out)
    else:
        print(out)


if __name__ == "__main__":
    main()