#!/usr/bin/env node

/**
 * Environment & Dependency Checker
 * Checks Node.js version and required packages
 * Reports what needs to be installed, doesn't auto-install
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const REQUIRED_VERSION = '18.0.0';
const REQUIRED_PACKAGES = [
  '@mozilla/readability',
  'turndown',
  'turndown-plugin-gfm',
  'jsdom',
  'defuddle'
];

function parseVersion(versionString) {
  const match = versionString.match(/v?(\d+)\.(\d+)\.(\d+)/);
  if (!match) return null;
  return {
    major: parseInt(match[1]),
    minor: parseInt(match[2]),
    patch: parseInt(match[3]),
    toString: () => `${match[1]}.${match[2]}.${match[3]}`
  };
}

function compareVersions(v1, v2) {
  if (v1.major !== v2.major) return v1.major - v2.major;
  if (v1.minor !== v2.minor) return v1.minor - v2.minor;
  return v1.patch - v2.patch;
}

function checkNodeVersion() {
  const nodeVersion = parseVersion(process.version);
  const requiredVersion = parseVersion(REQUIRED_VERSION);

  if (!nodeVersion) {
    return { ok: false, message: `Could not parse Node version: ${process.version}` };
  }

  if (compareVersions(nodeVersion, requiredVersion) < 0) {
    return {
      ok: false,
      message: `Node.js ${nodeVersion.toString()} detected. Required: v${REQUIRED_VERSION}+`
    };
  }

  return {
    ok: true,
    message: `✓ Node.js ${nodeVersion.toString()} (required: v${REQUIRED_VERSION}+)`
  };
}

function checkPackages() {
  const results = {
    ok: true,
    installed: [],
    missing: []
  };

  const nodeModulesPath = path.join(process.cwd(), 'node_modules');

  // Check if node_modules exists
  if (!fs.existsSync(nodeModulesPath)) {
    results.ok = false;
    results.missing = REQUIRED_PACKAGES;
    return results;
  }

  // Check each package
  for (const pkg of REQUIRED_PACKAGES) {
    const pkgPath = path.join(nodeModulesPath, pkg);
    if (fs.existsSync(pkgPath)) {
      try {
        const pkgJson = JSON.parse(fs.readFileSync(path.join(pkgPath, 'package.json'), 'utf-8'));
        results.installed.push({
          name: pkg,
          version: pkgJson.version
        });
      } catch {
        results.missing.push(pkg);
        results.ok = false;
      }
    } else {
      results.missing.push(pkg);
      results.ok = false;
    }
  }

  return results;
}

function checkOutputDir() {
  const mdDir = path.join(process.cwd(), '.md');
  if (!fs.existsSync(mdDir)) {
    try {
      fs.mkdirSync(mdDir, { recursive: true });
      return { ok: true, message: `✓ Output directory: ${mdDir}` };
    } catch (e) {
      return { ok: false, message: `Cannot create .md directory: ${e.message}` };
    }
  }
  return { ok: true, message: `✓ Output directory: ${mdDir}` };
}

function main() {
  console.log('\n📋 URL-to-Markdown Environment Check\n');

  // Check Node.js version
  const nodeCheck = checkNodeVersion();
  console.log(`Node.js: ${nodeCheck.message}`);
  if (!nodeCheck.ok) {
    console.error('\n❌ Node.js version requirement not met\n');
    process.exit(1);
  }

  // Check packages
  console.log('\nDependencies:');
  const pkgCheck = checkPackages();

  if (pkgCheck.installed.length > 0) {
    for (const pkg of pkgCheck.installed) {
      console.log(`  ✓ ${pkg.name} (v${pkg.version})`);
    }
  }

  if (pkgCheck.missing.length > 0) {
    console.log('\n⚠️  Missing packages:');
    for (const pkg of pkgCheck.missing) {
      console.log(`  • ${pkg}`);
    }

    console.log('\n📦 To install, run one of:\n');
    console.log('  npm install \\');
    console.log(pkgCheck.missing.map(p => `    ${p}`).join(' \\\n'));
    console.log('\n  OR\n');
    console.log('  pnpm add \\');
    console.log(pkgCheck.missing.map(p => `    ${p}`).join(' \\\n'));
    console.log('\n');

    if (pkgCheck.ok === false) {
      console.error('❌ Cannot proceed without dependencies\n');
      process.exit(1);
    }
  }

  // Check output directory
  const outCheck = checkOutputDir();
  console.log(`\nOutput: ${outCheck.message}`);
  if (!outCheck.ok) {
    console.error(`\n❌ ${outCheck.message}\n`);
    process.exit(1);
  }

  console.log('\n✅ Environment OK - Ready to convert URLs\n');
  process.exit(0);
}

if (require.main === module) {
  main();
}

module.exports = { checkNodeVersion, checkPackages, checkOutputDir };
