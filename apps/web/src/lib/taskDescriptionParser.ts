export interface ParsedBlock {
  key: string
  label: string
  content: string
  items?: string[]
}

export interface ParsedTaskDescription {
  hasStructuredBlocks: boolean
  context?: string
  inputs?: string[]
  outputs?: string[]
  conditionGuards?: string[]
  constraints?: string[]
  customBlocks: ParsedBlock[]
  generalNotes?: string
  rawDescription: string
}

function extractItems(content: string): string[] {
  return content
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
    .map((line) => {
      // Remove leading markdown bullets: "- ", "* ", "• ", or numbers "1. ", "2) "
      return line.replace(/^([-*•]|\d+[.)])\s+/, '').trim()
    })
    .filter((line) => line.length > 0)
}

/**
 * Parses raw task description looking for bracketed section headers like
 * [CONTEXT], [INPUT], [OUTPUT], [CONDITION GUARDS], [CONSTRAINTS], etc.
 */
export function parseTaskDescription(raw: string | undefined | null): ParsedTaskDescription {
  if (!raw || !raw.trim()) {
    return {
      hasStructuredBlocks: false,
      customBlocks: [],
      rawDescription: '',
    }
  }

  const trimmed = raw.trim()
  const headerRegex = /^\s*\[([A-Z0-9_\s/-]+)\]\s*$/gm
  const matches = Array.from(trimmed.matchAll(headerRegex))

  if (matches.length === 0) {
    return {
      hasStructuredBlocks: false,
      generalNotes: trimmed,
      customBlocks: [],
      rawDescription: trimmed,
    }
  }

  let context: string | undefined
  let inputs: string[] | undefined
  let outputs: string[] | undefined
  let conditionGuards: string[] | undefined
  let constraints: string[] | undefined
  const customBlocks: ParsedBlock[] = []
  let preamble: string | undefined

  const firstMatchIndex = matches[0].index ?? 0
  if (firstMatchIndex > 0) {
    const textBefore = trimmed.slice(0, firstMatchIndex).trim()
    if (textBefore) {
      preamble = textBefore
    }
  }

  for (let i = 0; i < matches.length; i++) {
    const match = matches[i]
    const rawTag = match[1].trim()
    const tagUpper = rawTag.toUpperCase()

    const contentStart = (match.index ?? 0) + match[0].length
    const nextMatch = matches[i + 1]
    const contentEnd = nextMatch ? (nextMatch.index ?? trimmed.length) : trimmed.length
    const content = trimmed.slice(contentStart, contentEnd).trim()

    switch (tagUpper) {
      case 'CONTEXT':
      case 'BACKGROUND':
      case 'OVERVIEW':
        context = content
        break
      case 'INPUT':
      case 'INPUTS':
        inputs = extractItems(content)
        break
      case 'OUTPUT':
      case 'OUTPUTS':
      case 'DELIVERABLES':
        outputs = extractItems(content)
        break
      case 'CONDITION GUARDS':
      case 'GUARDS':
      case 'CONDITION_GUARDS':
      case 'VERIFICATION':
        conditionGuards = extractItems(content)
        break
      case 'CONSTRAINTS':
      case 'INVARIANTS':
        constraints = extractItems(content)
        break
      default:
        customBlocks.push({
          key: tagUpper.toLowerCase().replace(/\s+/g, '_'),
          label: rawTag,
          content,
          items: content.includes('\n-') || content.includes('\n1.') ? extractItems(content) : undefined,
        })
        break
    }
  }

  return {
    hasStructuredBlocks: true,
    context,
    inputs,
    outputs,
    conditionGuards,
    constraints,
    customBlocks,
    generalNotes: preamble,
    rawDescription: trimmed,
  }
}
