import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import type { View, ViewConfig, ViewKind } from '../../../types'

export function useViewsForList(listId: string | undefined) {
  return useQuery({
    queryKey: ['views', 'list', listId],
    queryFn: () => api.get(`lists/${listId}/views`).json<View[]>(),
    enabled: !!listId,
  })
}

export function useCreateView(listId: string | undefined) {
  return useMutation({
    mutationFn: (body: { name: string; kind: ViewKind; config?: ViewConfig }) =>
      api.post(`lists/${listId}/views`, { json: body }).json<View>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['views', 'list', listId] }),
  })
}

export function useUpdateView(listId: string | undefined) {
  return useMutation({
    mutationFn: ({ id, ...patch }: { id: string; name?: string; kind?: ViewKind; config?: ViewConfig }) =>
      api.patch(`views/${id}`, { json: patch }).json<View>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['views', 'list', listId] }),
  })
}

export function useDeleteView(listId: string | undefined) {
  return useMutation({
    mutationFn: (id: string) => api.delete(`views/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['views', 'list', listId] }),
  })
}
