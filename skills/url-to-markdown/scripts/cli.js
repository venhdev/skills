#!/usr/bin/env node

/**
 * URL-to-Markdown CLI
 * Command-line interface with option parsing
 */

const fs = require('fs');
const path = require('path');
const { urlToMarkdown, batchUrlsToMarkdown } = require('./converter');
const { checkNodeVersion, checkPackages, checkOutputDir } = require('./checker');

// Simple argument parser
function parseArgs(args) {
  const options = {
    urls: [],
    input: null,
    output: null,
    keepImages: true,
    keepLinks: true,
    keepMetadata: true,
    pageMap: false,
    dateFormat: 'iso',
    batchSize: 3,
    verbose: false,
    help: false
  };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];

    if (arg === '--help' || arg === '-h') {
      options.help = true;
    } else if (arg === '--input') {
      options.input = args[++i];
    } else if (arg === '--output' || arg === '-o') {
      options.output = args[++i];
    } else if (arg === '--keep-images') {
      options.keepImages = args[++i] !== 'false';
    } else if (arg === '--keep-links') {
      options.keepLinks = args[++i] !== 'false';
    } else if (arg === '--keep-metadata') {
      options.keepMetadata = args[++i] !== 'false';
    } else if (arg === '--page-map') {
      options.pageMap = args[++i] !== 'false';
    } else if (arg === '--date-format') {
      const val = args[++i];
      if (val !== 'iso' && val !== 'locale') {
        console.error(`❌ Invalid --date-format: ${val} (expected 'iso' or 'locale')`);
        process.exit(1);
      }
      options.dateFormat = val;
    } else if (arg === '--batch-size') {
      options.batchSize = Math.max(1, parseInt(args[++i]) || 3);
    } else if (arg === '--verbose' || arg === '-v') {
      options.verbose = true;
    } else if (!arg.startsWith('--')) {
      options.urls.push(arg);
    }
  }

  return options;
}

function printHelp() {
  console.log(`
url-to-markdown - Convert URLs to Markdown

USAGE:
  url-to-markdown [URLs...] [OPTIONS]
  url-to-markdown --input urls.txt [OPTIONS]

EXAMPLES:
  # Single URL
  url-to-markdown https://example.com

  # Multiple URLs
  url-to-markdown url1 url2 url3

  # From file
  url-to-markdown --input urls.txt

  # Custom output
  url-to-markdown https://example.com --output article.md

  # Remove images (faster, smaller files)
  url-to-markdown urls.txt --keep-images false

  # Remove links and metadata
  url-to-markdown url --keep-links false --keep-metadata false

OPTIONS:
  --input FILE              Read URLs from file (one per line)
  --output PATH             Output file path (default: .md/<auto-desc>.md)
  --keep-images true|false  Include image references (default: true)
  --keep-links true|false   Include hyperlinks (default: true)
  --keep-metadata true|false Include title/author/date (default: true)
  --page-map true|false     Prepend ASCII tree of headings (default: false)
  --date-format iso|locale  Date format in metadata (default: iso)
  --batch-size N            Concurrent URLs (default: 3)
  --verbose, -v             Show detailed progress
  --help, -h                Show this help message
`);
}

function generateOutputPath(urls, customOutput) {
  if (customOutput) {
    return path.resolve(customOutput);
  }

  // Auto-generate path based on number of URLs
  const mdDir = path.join(process.cwd(), '.md');
  if (!fs.existsSync(mdDir)) {
    fs.mkdirSync(mdDir, { recursive: true });
  }

  if (urls.length === 1) {
    // Single URL: use title as filename
    const url = new URL(urls[0]);
    const slug = url.hostname.replace(/^www\./, '').replace(/\./g, '-');
    return path.join(mdDir, `${slug}.md`);
  } else {
    // Multiple URLs: numbered files with summary
    return { dir: mdDir, multiple: true };
  }
}

async function loadUrls(inputFile) {
  const content = fs.readFileSync(inputFile, 'utf-8');

  if (inputFile.endsWith('.csv')) {
    // CSV: extract first column
    return content
      .split('\n')
      .filter(line => line.trim())
      .map(line => line.split(',')[0].trim());
  } else if (inputFile.endsWith('.json')) {
    // JSON array
    try {
      const data = JSON.parse(content);
      return Array.isArray(data) ? data : [data];
    } catch {
      console.error('❌ Invalid JSON file');
      process.exit(1);
    }
  } else {
    // Text: one URL per line
    return content
      .split('\n')
      .filter(line => line.trim())
      .map(line => line.trim());
  }
}

async function main() {
  const args = process.argv.slice(2);
  const options = parseArgs(args);

  if (options.help || args.length === 0) {
    printHelp();
    process.exit(0);
  }

  // Check environment
  const nodeCheck = checkNodeVersion();
  if (!nodeCheck.ok) {
    console.error(`❌ ${nodeCheck.message}\n`);
    process.exit(1);
  }

  const pkgCheck = checkPackages();
  if (!pkgCheck.ok && pkgCheck.missing.length > 0) {
    console.error('\n❌ Missing packages:');
    for (const pkg of pkgCheck.missing) {
      console.error(`  • ${pkg}`);
    }
    console.error(`\nRun: npm install ${pkgCheck.missing.join(' ')}\n`);
    process.exit(1);
  }

  // Collect URLs
  let urls = options.urls;
  if (options.input) {
    try {
      const inputUrls = await loadUrls(options.input);
      urls = [...urls, ...inputUrls];
    } catch (e) {
      console.error(`❌ Failed to read input file: ${e.message}\n`);
      process.exit(1);
    }
  }

  if (urls.length === 0) {
    console.error('❌ No URLs provided\n');
    printHelp();
    process.exit(1);
  }

  // Validate URLs
  const validUrls = [];
  for (const url of urls) {
    try {
      new URL(url);
      validUrls.push(url);
    } catch {
      console.error(`⚠️  Invalid URL: ${url}`);
    }
  }

  if (validUrls.length === 0) {
    console.error('❌ No valid URLs\n');
    process.exit(1);
  }

  console.log(`\n📡 Converting ${validUrls.length} URL(s)...\n`);

  // Convert
  try {
    const conversionOptions = {
      keepImages: options.keepImages,
      keepLinks: options.keepLinks,
      keepMetadata: options.keepMetadata,
      pageMap: options.pageMap,
      dateFormat: options.dateFormat,
      verbose: options.verbose
    };

    const results = await batchUrlsToMarkdown(validUrls, conversionOptions, options.batchSize);

    // Generate output
    const outputPath = generateOutputPath(validUrls, options.output);
    const mdDir = typeof outputPath === 'string' ? path.dirname(outputPath) : outputPath.dir;

    // Ensure directory exists
    if (!fs.existsSync(mdDir)) {
      fs.mkdirSync(mdDir, { recursive: true });
    }

    let successCount = 0;
    let failureCount = 0;
    const fileList = [];

    if (typeof outputPath === 'string' && validUrls.length === 1) {
      // Single URL: save directly
      const result = results.get(validUrls[0]);
      if (result && !result.error) {
        let content = '';
        if (options.keepMetadata) {
          content += `# ${result.title}\n\n`;
          if (result.author) content += `_${result.author}_\n\n`;
          if (result.date) content += `_${result.date}_\n\n`;
          content += '---\n\n';
        }
        content += result.markdown;
        content += `\n\n---\n**Source:** ${result.url}\n`;

        fs.writeFileSync(outputPath, content, 'utf-8');
        fileList.push(outputPath);
        successCount++;
      } else {
        failureCount++;
        console.error(`✗ ${validUrls[0]}: ${result?.error || 'Unknown error'}`);
      }
    } else {
      // Multiple URLs: save to numbered files
      let idx = 1;
      for (const [url, result] of results) {
        if (result && !result.error) {
          const sanitizedTitle = result.title
            .toLowerCase()
            .replace(/[^a-z0-9]+/g, '-')
            .replace(/^-|-$/g, '')
            .slice(0, 50) || `article-${idx}`;

          const filename = `${String(idx).padStart(2, '0')}-${sanitizedTitle}.md`;
          const filepath = path.join(mdDir, filename);

          let content = '';
          if (options.keepMetadata) {
            content += `# ${result.title}\n\n`;
            if (result.author) content += `_${result.author}_\n\n`;
            if (result.date) content += `_${result.date}_\n\n`;
            content += '---\n\n';
          }
          content += result.markdown;
          content += `\n\n---\n**Source:** ${result.url}\n`;

          fs.writeFileSync(filepath, content, 'utf-8');
          fileList.push(filepath);
          successCount++;
          idx++;
        } else {
          failureCount++;
          console.error(`✗ ${url}: ${result?.error || 'Unknown error'}`);
        }
      }

      // Write summary
      if (fileList.length > 0) {
        const summaryPath = path.join(mdDir, '00-SUMMARY.txt');
        const summary = `# URL-to-Markdown Conversion Summary
Generated: ${new Date().toISOString()}
Total URLs: ${validUrls.length}
Successful: ${successCount}
Failed: ${failureCount}

Files:
${fileList.map((f, i) => `${i + 1}. ${path.basename(f)}`).join('\n')}
`;
        fs.writeFileSync(summaryPath, summary, 'utf-8');
      }
    }

    // Report results
    console.log(`✅ Conversion complete!\n`);
    console.log(`📊 Results:`);
    console.log(`  ✓ Successful: ${successCount}`);
    console.log(`  ✗ Failed: ${failureCount}`);
    console.log(`\n📁 Output: ${mdDir}`);
    console.log(`📄 Files: ${fileList.length}`);

    if (fileList.length > 0 && fileList.length <= 5) {
      console.log('\n📋 Created files:');
      fileList.forEach((f, i) => {
        console.log(`  ${i + 1}. ${path.basename(f)}`);
      });
    }

    console.log('');
    process.exit(successCount > 0 ? 0 : 1);
  } catch (error) {
    console.error(`\n❌ Conversion failed: ${error.message}\n`);
    process.exit(1);
  }
}

if (require.main === module) {
  main();
}

module.exports = { parseArgs, generateOutputPath, loadUrls };
