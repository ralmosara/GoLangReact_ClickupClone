import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import type { CustomField, CustomValue, FieldConfig, FieldKind } from '../../../types'

export function useFieldsForList(listId: string | undefined) {
  return useQuery({
    queryKey: ['custom-fields', 'list', listId],
    queryFn: () => api.get(`lists/${listId}/custom-fields`).json<CustomField[]>(),
    enabled: !!listId,
  })
}

export function useCreateField(listId: string | undefined) {
  return useMutation({
    mutationFn: (body: { name: string; kind: FieldKind; config?: FieldConfig; required?: boolean; order_index?: number }) =>
      api.post(`lists/${listId}/custom-fields`, { json: body }).json<CustomField>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['custom-fields', 'list', listId] }),
  })
}

export function useUpdateField(listId: string | undefined) {
  return useMutation({
    mutationFn: ({ id, ...patch }: { id: string } & Partial<Pick<CustomField, 'name' | 'kind' | 'config' | 'required' | 'order_index'>>) =>
      api.patch(`custom-fields/${id}`, { json: patch }).json<CustomField>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['custom-fields', 'list', listId] }),
  })
}

export function useDeleteField(listId: string | undefined) {
  return useMutation({
    mutationFn: (id: string) => api.delete(`custom-fields/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['custom-fields', 'list', listId] }),
  })
}

export function useValuesForTask(taskId: string | undefined) {
  return useQuery({
    queryKey: ['custom-values', taskId],
    queryFn: () => api.get(`tasks/${taskId}/custom-values`).json<CustomValue[]>(),
    enabled: !!taskId,
  })
}

export function useUpsertValue(taskId: string | undefined) {
  return useMutation({
    mutationFn: (body: { field_id: string; value: unknown }) =>
      api.put(`tasks/${taskId}/custom-values`, { json: body }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['custom-values', taskId] }),
  })
}

export function useClearValue(taskId: string | undefined) {
  return useMutation({
    mutationFn: (fieldID: string) => api.delete(`tasks/${taskId}/custom-values/${fieldID}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['custom-values', taskId] }),
  })
}
