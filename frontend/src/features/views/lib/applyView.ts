import type { FilterClause, SortField, Task, ViewConfig } from '../../../types'

/**
 * Apply a view's config to a set of tasks on the client. Backend support for
 * SQL-side filtering lands in M6 with saved global views; for now everything is
 * in-memory.
 */
export function applyView(tasks: Task[], config: ViewConfig, taskTagIds?: Record<string, string[]>, taskAssignees?: Record<string, string[]>): Task[] {
  let out = tasks.slice()

  const filters = config.filters ?? []
  for (const f of filters) {
    out = out.filter((t) => matchFilter(t, f, { taskTagIds, taskAssignees }))
  }

  const sort = config.sort ?? []
  if (sort.length > 0) {
    out.sort((a, b) => {
      for (const s of sort) {
        const cmp = compareField(a, b, s)
        if (cmp !== 0) return s.desc ? -cmp : cmp
      }
      return 0
    })
  }

  return out
}

function matchFilter(
  t: Task,
  f: FilterClause,
  ctx: { taskTagIds?: Record<string, string[]>; taskAssignees?: Record<string, string[]> },
): boolean {
  switch (f.field) {
    case 'status_id':
      if (f.op === 'in' && Array.isArray(f.value)) return (f.value as string[]).includes(t.status_id ?? '')
      if (f.op === 'eq') return t.status_id === f.value
      return true
    case 'priority':
      if (f.op === 'eq') return t.priority === f.value
      if (f.op === 'gt') return t.priority > Number(f.value)
      if (f.op === 'lt') return t.priority < Number(f.value)
      return true
    case 'assignee_id':
      if (f.op === 'in' && Array.isArray(f.value)) {
        const assignees = ctx.taskAssignees?.[t.id] ?? []
        return (f.value as string[]).some((id) => assignees.includes(id))
      }
      return true
    case 'tag_id':
      if (f.op === 'in' && Array.isArray(f.value)) {
        const tags = ctx.taskTagIds?.[t.id] ?? []
        return (f.value as string[]).some((id) => tags.includes(id))
      }
      return true
    case 'due_at':
      if (f.op === 'empty') return !t.due_at
      if (f.op === 'notempty') return !!t.due_at
      if (f.op === 'before' && typeof f.value === 'string') return !!t.due_at && t.due_at < f.value
      if (f.op === 'after'  && typeof f.value === 'string') return !!t.due_at && t.due_at > f.value
      return true
    default:
      return true
  }
}

function compareField(a: Task, b: Task, s: SortField): number {
  const av = (a as unknown as Record<string, unknown>)[s.field]
  const bv = (b as unknown as Record<string, unknown>)[s.field]
  if (av == null && bv == null) return 0
  if (av == null) return 1
  if (bv == null) return -1
  if (typeof av === 'number' && typeof bv === 'number') return av - bv
  return String(av).localeCompare(String(bv))
}

export function groupTasks(
  tasks: Task[],
  groupBy: string | undefined,
  ctx: { taskAssignees?: Record<string, string[]> },
): Array<{ key: string; label: string; tasks: Task[] }> {
  if (!groupBy) return [{ key: '__all', label: '', tasks }]

  const buckets = new Map<string, { key: string; label: string; tasks: Task[] }>()
  const put = (key: string, label: string, t: Task) => {
    let b = buckets.get(key)
    if (!b) {
      b = { key, label, tasks: [] }
      buckets.set(key, b)
    }
    b.tasks.push(t)
  }

  for (const t of tasks) {
    switch (groupBy) {
      case 'status_id':
        put(t.status_id ?? '__none', t.status_id ? '' : 'No status', t)
        break
      case 'priority':
        put(String(t.priority), '', t)
        break
      case 'assignee': {
        const assignees = ctx.taskAssignees?.[t.id] ?? []
        if (assignees.length === 0) put('__none', 'Unassigned', t)
        else assignees.forEach((uid) => put(uid, '', t))
        break
      }
      default:
        put('__all', '', t)
    }
  }

  return Array.from(buckets.values())
}
