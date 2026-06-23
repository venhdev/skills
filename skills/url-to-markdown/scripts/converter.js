/**
 * URL-to-Markdown Converter (v2)
 * Delegates extraction + HTML→MD to defuddle (which wraps readability + turndown
 * with lazy-image handling, code-language hints, JSON-LD/schema.org extraction,
 * expanded badSelectors, and other improvements).
 *
 * Pipeline:
 *   1. fetch (with AbortController timeout)
 *   2. Defuddle.parse(html, url, { separateMarkdown: true })
 *   3. optional keepImages/keepLinks post-pass
 *   4. optional customCleanupFn
 *   5. optional page-structure map (Tier 3)
 *   6. optional dateFormat (iso | locale)
 *
 * Breaking change from v1: result field names follow defuddle's schema
 *   - title    (was: title, unchanged)
 *   - author   (was: byline)
 *   - date     (was: publishedTime)
 *   - description  (new)
 */

const { generatePageMap } = require('./page-map');

// Lazy-loaded because defuddle's "./node" subpath is ESM-only
let _defuddle = null;
async function getDefuddle() {
  if (!_defuddle) {
    const mod = await import('defuddle/node');
    _defuddle = mod.Defuddle;
  }
  return _defuddle;
}

const FETCH_TIMEOUT_MS = 10000;
const USER_AGENT = 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36';

async function fetchHtml(url) {
  const abortController = new AbortController();
  const timeoutId = setTimeout(() => abortController.abort(), FETCH_TIMEOUT_MS);
  try {
    const response = await fetch(url, {
      signal: abortController.signal,
      headers: { 'User-Agent': USER_AGENT }
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`);
    }
    return await response.text();
  } finally {
    clearTimeout(timeoutId);
  }
}

/**
 * Convert a single URL to Markdown.
 * @param {string} url
 * @param {Object} options
 *   keepImages       bool   default true
 *   keepLinks        bool   default true
 *   keepMetadata     bool   default true (controls cli.js metadata block, not converter)
 *   customCleanupFn  fn     (markdown) => markdown
 *   pageMap          bool   default false  (prepend ASCII tree of headings)
 *   dateFormat       'iso' | 'locale'   default 'iso'
 * @returns {Promise<{markdown, title, author, date, url, description}>}
 */
async function urlToMarkdown(url, options = {}) {
  const {
    keepImages = true,
    keepLinks = true,
    customCleanupFn,
    pageMap = false,
    dateFormat = 'iso'
  } = options;

  // 1. Fetch HTML
  const html = await fetchHtml(url);

  // 2. Hand off to defuddle
  const Defuddle = await getDefuddle();
  const article = await Defuddle(html, url, {
    separateMarkdown: true,         // populate contentMarkdown; keep content as HTML
    removeImages: !keepImages,      // strip <img> during extraction
    removeExactSelectors: true,
    removePartialSelectors: true,
    removeHiddenElements: true,
    removeLowScoring: true,
    removeSmallImages: true,
    standardize: true,              // footnotes, headings, code blocks
    removeContentPatterns: true,    // boilerplate, read-time, etc.
    useAsync: false                 // don't fetch external resources
  });

  let markdown = article.contentMarkdown || '';

  // 3. Strip raw-HTML <a>/<img> fallback (defuddle may emit both MD and HTML forms)
  if (!keepImages) {
    markdown = markdown.replace(/<img[^>]*>/gi, '');
    markdown = markdown.replace(/<picture[^>]*>[\s\S]*?<\/picture>/gi, '');
  }
  if (!keepLinks) {
    markdown = markdown.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1');
    markdown = markdown.replace(/<a[^>]*>(.*?)<\/a>/gi, '$1');
  }

  // 4. Custom cleanup hook (user-provided)
  if (customCleanupFn && typeof customCleanupFn === 'function') {
    markdown = customCleanupFn(markdown);
  }

  // 5. Optional page-structure map (Tier 3)
  if (pageMap) {
    const map = generatePageMap(markdown, article.title || 'Page Structure');
    if (map) {
      markdown = map + '\n\n---\n\n' + markdown;
    }
  }

  // 6. Date formatting
  let date = article.published || '';
  if (date && dateFormat === 'locale') {
    const d = new Date(date);
    if (!isNaN(d.getTime())) {
      date = d.toLocaleDateString();
    }
  }

  return {
    markdown,
    title: article.title || 'Untitled',
    author: article.author || '',
    date,
    url,
    description: article.description || ''
  };
}

/**
 * Batch-process multiple URLs with bounded concurrency.
 * @param {Array<string>} urls
 * @param {Object} options  (same as urlToMarkdown)
 * @param {number} concurrency  default 3
 * @returns {Promise<Map<string, result>>}
 */
async function batchUrlsToMarkdown(urls, options = {}, concurrency = 3) {
  const { verbose = false } = options;
  const results = new Map();
  const queue = [...urls];
  const processing = [];

  async function worker(workerId) {
    while (queue.length > 0) {
      const url = queue.shift();
      if (!url) break;

      try {
        if (verbose) {
          const idx = urls.indexOf(url) + 1;
          console.log(`[${idx}/${urls.length}] Fetching ${url}...`);
        }

        const result = await urlToMarkdown(url, options);

        if (verbose) {
          const idx = urls.indexOf(url) + 1;
          console.log(`[${idx}/${urls.length}] ✓ ${result.title} (${result.markdown.length} chars)`);
        }

        results.set(url, result);
      } catch (error) {
        if (verbose) {
          const idx = urls.indexOf(url) + 1;
          console.error(`[${idx}/${urls.length}] ✗ ${error.message}`);
        }
        results.set(url, { error: error.message });
      }
    }
  }

  for (let i = 0; i < Math.min(concurrency, urls.length); i++) {
    processing.push(worker(i));
  }

  await Promise.all(processing);
  return results;
}

module.exports = {
  urlToMarkdown,
  batchUrlsToMarkdown
};