import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import type { TimeEntry } from '../../../types'

export function useTimeEntries(taskId: string | undefined) {
  return useQuery({
    queryKey: ['time-entries', taskId],
    queryFn: () => api.get(`tasks/${taskId}/time-entries`).json<TimeEntry[]>(),
    enabled: !!taskId,
  })
}

export function useActiveTimers() {
  return useQuery({
    queryKey: ['time-active'],
    queryFn: () => api.get('me/timer').json<TimeEntry[]>(),
    refetchOnWindowFocus: true,
  })
}

export function useStartTimer() {
  return useMutation({
    mutationFn: ({ taskId, note }: { taskId: string; note?: string }) =>
      api.post(`tasks/${taskId}/timer/start`, { json: note ? { note } : {} }).json<TimeEntry>(),
    onSuccess: (_d, { taskId }) => {
      queryClient.invalidateQueries({ queryKey: ['time-entries', taskId] })
      queryClient.invalidateQueries({ queryKey: ['time-active'] })
    },
  })
}

export function useStopTimer() {
  return useMutation({
    mutationFn: (id: string) => api.post(`time-entries/${id}/stop`).json<TimeEntry>(),
    onSuccess: (entry) => {
      queryClient.invalidateQueries({ queryKey: ['time-entries', entry.task_id] })
      queryClient.invalidateQueries({ queryKey: ['time-active'] })
    },
  })
}

export function useLogManual(taskId: string | undefined) {
  return useMutation({
    mutationFn: (body: { started_at: string; stopped_at: string; note?: string; billable?: boolean }) =>
      api.post(`tasks/${taskId}/time-entries`, { json: body }).json<TimeEntry>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['time-entries', taskId] }),
  })
}

export function useDeleteTimeEntry(taskId: string | undefined) {
  return useMutation({
    mutationFn: (id: string) => api.delete(`time-entries/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['time-entries', taskId] }),
  })
}
