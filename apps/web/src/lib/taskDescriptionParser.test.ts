import { describe, it, expect } from 'vitest'
import { parseTaskDescription } from './taskDescriptionParser'

describe('taskDescriptionParser', () => {
  it('handles empty or null descriptions', () => {
    expect(parseTaskDescription(null).hasStructuredBlocks).toBe(false)
    expect(parseTaskDescription('').hasStructuredBlocks).toBe(false)
  })

  it('handles unstructured text as general notes', () => {
    const raw = 'Just a simple unstructured note.'
    const parsed = parseTaskDescription(raw)
    expect(parsed.hasStructuredBlocks).toBe(false)
    expect(parsed.generalNotes).toBe(raw)
  })

  it('parses structured blocks from the screenshot accurately', () => {
    const raw = `[CONTEXT]
Cổng hội tụ chất lượng cuối cùng sau khi hoàn tất toàn bộ các lát cắt MODE-01..04.

[INPUT]
- Thư mục test/: mobile/handmold/test/
- Thư mục android: mobile/handmold/android/

[OUTPUT]
1. 413+ Flutter unit & widget tests pass 100%.
2. fvm flutter analyze đạt zero issues.
3. Android Gradle test đạt 313 tasks pass 100%.

[CONDITION GUARDS]
- fvm flutter test && fvm flutter analyze && cd android && ./gradlew test EXIT 0.
`
    const parsed = parseTaskDescription(raw)
    expect(parsed.hasStructuredBlocks).toBe(true)
    expect(parsed.context).toContain('Cổng hội tụ chất lượng cuối cùng')
    expect(parsed.inputs).toHaveLength(2)
    expect(parsed.inputs?.[0]).toBe('Thư mục test/: mobile/handmold/test/')
    expect(parsed.outputs).toHaveLength(3)
    expect(parsed.outputs?.[0]).toBe('413+ Flutter unit & widget tests pass 100%.')
    expect(parsed.conditionGuards).toHaveLength(1)
    expect(parsed.conditionGuards?.[0]).toContain('fvm flutter test')
  })
})
