import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TaskSpecPane } from './TaskSpecPane'
import type { Task } from '../../types/contracts'

function createMockTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 'STK-04',
    title: 'Compress Footer to 1-Row & End-to-End Regression Gate',
    status: 'RUNNING',
    description: `[CONTEXT]
Refactor mobile footer action bar.

[INPUT]
mobile/handmold/lib/features/movement/presentation/stockout_v2_screen.dart

[OUTPUT]
1-row ergonomic bottom action bar

[ACCEPTANCE]
100% tests pass`,
    acceptance_criteria: [
      'Khu vực đáy chỉ chiếm đúng 1 row duy nhất với chiều cao 56dp',
      'Đã xóa bỏ hoàn toàn 2 nút phụ SỨC CHỨA và XÓA PHIÊN cũ',
    ],
    verification_commands: ['cd mobile/handmold && fvm flutter test'],
    target_files: ['mobile/handmold/lib/features/movement/presentation/stockout_v2_screen.dart'],
    dependencies: ['STK-03'],
    priority: 'P1',
    complexity: 'LOW',
    risk_level: 'LOW',
    harness: 'orchestrator',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
    ...overrides,
  }
}

describe('TaskSpecPane', () => {
  it('renders empty fallback when task is null', () => {
    render(<TaskSpecPane task={null} allTasks={[]} />)
    expect(screen.getByText('No task selected')).toBeDefined()
  })

  it('renders task specification with structured blocks and triage badges', () => {
    const task = createMockTask()
    const depTask: Task = {
      id: 'STK-03',
      title: 'Setup Form',
      status: 'COMPLETED',
      dependencies: [],
      harness: 'orchestrator',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }

    render(<TaskSpecPane task={task} allTasks={[task, depTask]} />)

    expect(screen.getByText('STK-04')).toBeDefined()
    expect(screen.getByText('Compress Footer to 1-Row & End-to-End Regression Gate')).toBeDefined()
    expect(screen.getByText('Priority: P1')).toBeDefined()
    expect(screen.getByText('Complexity: LOW')).toBeDefined()
    expect(screen.getByText('Risk: LOW')).toBeDefined()
    expect(screen.getByText('Refactor mobile footer action bar.')).toBeDefined()
    expect(screen.getByText('STK-03')).toBeDefined()
  })

  it('triggers onSelectTaskId when clicking a dependency badge', () => {
    const task = createMockTask()
    const depTask: Task = {
      id: 'STK-03',
      title: 'Prerequisite',
      status: 'COMPLETED',
      dependencies: [],
      harness: 'orchestrator',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    }
    const onSelectTaskId = vi.fn()

    render(
      <TaskSpecPane
        task={task}
        allTasks={[task, depTask]}
        onSelectTaskId={onSelectTaskId}
      />
    )

    const depButton = screen.getByTitle('Jump and focus on STK-03')
    fireEvent.click(depButton)

    expect(onSelectTaskId).toHaveBeenCalledTimes(1)
    expect(onSelectTaskId).toHaveBeenCalledWith('STK-03')
  })

  it('scroll container has flex containment and isolated scroll boundaries', () => {
    const task = createMockTask()
    const { container } = render(<TaskSpecPane task={task} allTasks={[task]} />)

    const scrollContainer = container.querySelector('.overflow-y-auto')
    expect(scrollContainer).toBeDefined()
    expect(scrollContainer?.className).toContain('min-h-0')
    expect(scrollContainer?.className).toContain('overscroll-contain')
    expect(scrollContainer?.className).toContain('[contain:content]')
  })
})
