/**
 * URL-to-Markdown Module Exports
 * Use as a library in your own code
 */

const { urlToMarkdown, batchUrlsToMarkdown } = require('./converter');
const { checkNodeVersion, checkPackages } = require('./checker');

module.exports = {
  urlToMarkdown,
  batchUrlsToMarkdown,
  checkNodeVersion,
  checkPackages
};

/**
 * Usage as library:
 *
 * const { urlToMarkdown, batchUrlsToMarkdown } = require('./scripts/index.js');
 *
 * // Single URL
 * const result = await urlToMarkdown('https://example.com', {
 *   keepImages: true,
 *   keepLinks: true,
 *   keepMetadata: true
 * });
 * console.log(result.markdown);
 *
 * // Multiple URLs
 * const results = await batchUrlsToMarkdown(
 *   ['url1', 'url2', 'url3'],
 *   { keepImages: true },
 *   3  // concurrency
 * );
 *
 * for (const [url, result] of results) {
 *   console.log(`${url} -> ${result.title}`);
 * }
 */
