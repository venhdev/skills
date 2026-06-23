/**
 * Page-Structure Map Generator
 *
 * Parses ATX-style headings from a Markdown string and renders an ASCII tree.
 * Copied verbatim from MD-This-Page/tabs/markdown.tsx:178-246 (defuddle skill's
 * source repo). Pure JS — no DOM, no React.
 *
 * @param {string} markdown   The Markdown source to scan for headings.
 * @param {string} title      Root label for the tree. Defaults to "Document Structure".
 * @returns {string}          "# Page Structure Map\n```text\n...\n```\n"  (or "" if no headings)
 */
function generatePageMap(markdown, title = 'Document Structure') {
  const lines = markdown.split('\n');
  const headings = [];

  // Extract headings
  lines.forEach((line) => {
    const match = line.match(/^(#{1,6})\s+(.+)$/);
    if (match) {
      headings.push({ level: match[1].length, text: match[2].trim() });
    }
  });

  if (headings.length === 0) return '';

  // Build tree
  const root = { text: title, level: 0, children: [] };
  const stack = [root];

  headings.forEach((h) => {
    const node = { text: h.text, level: h.level, children: [] };
    while (stack.length > 1 && stack[stack.length - 1].level >= h.level) {
      stack.pop();
    }
    stack[stack.length - 1].children.push(node);
    stack.push(node);
  });

  // Render tree
  let mapStr = `${title}\n`;

  function renderNode(node, prefix, isLast, isRoot) {
    if (!isRoot) {
      const connector = isLast ? '└── ' : '├── ';
      mapStr += `${prefix}${connector}${node.text}\n`;

      if (node.children.length > 0) {
        const childPrefix = prefix + (isLast ? '    ' : '│   ');
        node.children.forEach((child, index) => {
          renderNode(
            child,
            childPrefix,
            index === node.children.length - 1,
            false
          );
        });
      }
    } else {
      node.children.forEach((child, index) => {
        renderNode(child, '', index === node.children.length - 1, false);
      });
    }
  }

  renderNode(root, '', true, true);

  // Clean up empty vertical lines at the end if any
  mapStr = mapStr.replace(/│\n$/g, '').trimEnd();

  return '# Page Structure Map\n```text\n' + mapStr + '\n```\n';
}

module.exports = { generatePageMap };