# mmdc CLI — Quick Reference

Source: official README + `mmdc -h` against mmdc 11.15.0.

## Basic

```bash
mmdc -i input.mmd -o output.svg
```

## Common flags

| Flag | Purpose | Default |
|---|---|---|
| `-V, --version` | Output version | — |
| `-i, --input <input>` | Input. `.md` extracts all ```mermaid blocks; `-` reads stdin | required |
| `-o, --output [file]` | Output. `.svg`/`.png`/`.pdf`/`.md`, or `-` for stdout | input + ".svg" |
| `-e, --outputFormat <fmt>` | Output format (svg/png/pdf) | from `-o` extension |
| `-t, --theme [theme]` | default / forest / dark / neutral | default |
| `-b, --backgroundColor [color]` | Background | white |
| `-w, --width` / `-H, --height` | Page size (px) | 800 / 600 |
| `-q, --quiet` | Suppress logs | off |
| `-a, --artefacts [path]` | Output dir for multi-block | `-o` directory |
| `-j, --jobs <n>` | Parallel jobs | half CPUs (4) |
| `-c, --configFile` / `-C, --cssFile` | Mermaid JSON config / CSS for SVG | — |
| `-I, --svgId [id]` | SVG id attribute | — |
| `-s, --scale [n]` | Puppeteer scale | 1 |
| `-f, --pdfFit` | Scale PDF to fit | off |
| `-p, --puppeteerConfigFile [json]` | Puppeteer config (schema: see Puppeteer docs) | — |
| `--iconPacks <icons...>` | Icon packs (e.g. `@iconify-json/logos`) | [] |
| `--iconPacksNamesAndUrls <prefix#url...>` | Icon packs from custom JSON URLs (`<prefix>#<url>`) | [] |

## Render from .md

When input ends in `.md`, mmdc extracts every ```mermaid block and:

1. Generates `<o-stem>-1.svg`, `<o-stem>-2.svg`, ... where `<o-stem>` is the basename of `-o` (not `-i`).
2. If `-o` ends in `.md`/`.markdown`, writes transformed Markdown replacing each block with `![diagram](./<o-stem>-N.svg)`.
3. Non-mermaid Markdown passes through.

```bash
mmdc -i diagrams.md -o ./out.md
# → ./out.md + diagrams-1.svg, diagrams-2.svg, ...
```

By default SVGs land beside the `-o` file (same directory). Use `-a <dir>` to redirect to a different directory — see SKILL.md §7a for the gotcha.

## Render from stdin

```bash
echo 'graph TD; A-->B' | mmdc -i - -o - --quiet
```

`-i -` reads raw mermaid source (no `.md` extraction path for stdin).

## Common errors

- `Parse error on line N` — invalid Mermaid
- `Output file must end with ".md"/".markdown", ".svg", ".png" or ".pdf"` — wrong `-o` extension
- `Chromium launch failed` — Chromium not installed. See SKILL.md §6 STOP gate before installing.
- `timeout` — large diagram; pass `--puppeteerConfigFile <json>`

## Version check

```bash
mmdc --version   # e.g. 11.15.0
```

If `mmdc` not on PATH, see SKILL.md §6 STOP gate for the fallback policy.

**Verification scope:** flag table verified against `mmdc -h`. Multi-block behavior verified empirically against mmdc 11.15.0.
