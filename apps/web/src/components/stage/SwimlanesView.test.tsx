import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { SwimlanesView } from './SwimlanesView'
import type { Task, TaskStatus } from '../../types/contracts'

const ALL_STATUSES: TaskStatus[] = [
  'PENDING',
  'READY',
  'RUNNING',
  'AWAITING_REVIEW',
  'AWAITING_MERGE',
  'CONFLICT',
  'COMPLETED',
  'FAILED',
  'BLOCKED',
  'CANCELLED',
]

function makeMockTask(id: string, status: TaskStatus): Task {
  return {
    id,
    title: `Task ${id}`,
    status,
    dependencies: [],
    harness: 'orchestrator',
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }
}

describe('SwimlanesView', () => {
  it('renders all 10 canonical status lanes with snap-start class', () => {
    const tasks = ALL_STATUSES.map((status, idx) => makeMockTask(`T-${idx}`, status))
    const onSelect = vi.fn()

    render(
      <SwimlanesView tasks={tasks} selectedTaskId={null} onSelectTask={onSelect} />
    )

    // All 10 lanes must be present in DOM
    for (const status of ALL_STATUSES) {
      const lane = screen.getByTestId(`lane-${status}`)
      expect(lane).toBeDefined()
      expect(lane.className).toContain('snap-start')
    }

    // All 10 tasks must be visible in their respective lane
    for (const task of tasks) {
      expect(screen.getByText(task.id)).toBeDefined()
      expect(screen.getByText(task.title)).toBeDefined()
    }
  })

  it('triggers onSelectTask when a task card is clicked', () => {
    const task = makeMockTask('T-SELECT', 'READY')
    const onSelect = vi.fn()

    render(<SwimlanesView tasks={[task]} selectedTaskId={null} onSelectTask={onSelect} />)

    const card = screen.getByText('T-SELECT')
    fireEvent.click(card)

    expect(onSelect).toHaveBeenCalledTimes(1)
    expect(onSelect).toHaveBeenCalledWith(task)
  })

  it('renders empty lane indicators when a lane has no tasks', () => {
    render(<SwimlanesView tasks={[]} selectedTaskId={null} onSelectTask={vi.fn()} />)

    const emptyNotices = screen.getAllByText('No tasks in this lane')
    expect(emptyNotices.length).toBe(10)
  })
})
