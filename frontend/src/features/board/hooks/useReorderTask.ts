import { useMutation } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import type { Task } from '../../../types'

/**
 * Gap-insertion math: given the sorted positions of the items that will flank
 * the dropped card in the target column, return a float that preserves ordering
 * without rewriting siblings.
 */
export function computePosition(prev: number | undefined, next: number | undefined): number {
  if (prev == null && next == null) return 1000
  if (prev == null && next != null) return next - 1000
  if (prev != null && next == null) return prev + 1000
  return ((prev as number) + (next as number)) / 2
}

export interface ReorderArgs {
  taskId: string
  listId: string
  /** `null` to clear to a sentinel "no status"; `undefined` to leave unchanged. */
  statusId: string | null | undefined
  prev: number | undefined
  next: number | undefined
}

export function useReorderTask() {
  return useMutation({
    mutationFn: async ({ taskId, statusId, prev, next }: ReorderArgs) => {
      const body: Record<string, unknown> = {}
      if (statusId !== undefined) body.status_id = statusId
      if (prev !== undefined) body.prev = prev
      if (next !== undefined) body.next = next
      return api.patch(`tasks/${taskId}/position`, { json: body }).json<Task>()
    },
    onMutate: async ({ taskId, listId, statusId, prev, next }) => {
      await queryClient.cancelQueries({ queryKey: ['tasks', listId] })
      const previous = queryClient.getQueryData<Task[]>(['tasks', listId])
      if (previous) {
        const newPos = computePosition(prev, next)
        const updated = previous.map((t) =>
          t.id === taskId
            ? { ...t, position: newPos, status_id: statusId === undefined ? t.status_id : (statusId ?? undefined) }
            : t,
        )
        // Re-sort so the optimistic view matches the server's ORDER BY position.
        updated.sort((a, b) => a.position - b.position)
        queryClient.setQueryData(['tasks', listId], updated)
      }
      return { previous }
    },
    onError: (_err, { listId }, ctx) => {
      if (ctx?.previous) queryClient.setQueryData(['tasks', listId], ctx.previous)
    },
    onSettled: (_data, _err, { listId }) => {
      queryClient.invalidateQueries({ queryKey: ['tasks', listId] })
    },
  })
}
