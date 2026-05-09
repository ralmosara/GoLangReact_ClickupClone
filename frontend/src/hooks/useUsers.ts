// useUsers.ts — TanStack Query hooks for the User Management page.
//
// All endpoints are workspace-scoped under /workspaces/{wsID}/users and
// require the user.manage permission on that workspace. Mutations
// invalidate both ['workspace-users', wsID] (the User Management table)
// and ['workspace-members', wsID] (the MembersPage list) so the two
// pages stay in lockstep.
import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import { queryClient } from '../lib/queryClient'
import type { User } from '../types'

// WorkspaceUser mirrors the backend's domain.WorkspaceUser shape — User
// fields plus the per-workspace role + joined_at + is_owner.
export interface WorkspaceUser extends User {
  role: string
  joined_at: string
  is_owner: boolean
}

const invalidateAll = (workspaceId: string) => {
  queryClient.invalidateQueries({ queryKey: ['workspace-users', workspaceId] })
  queryClient.invalidateQueries({ queryKey: ['workspace-members', workspaceId] })
}

export function useWorkspaceUsers(workspaceId: string | undefined) {
  return useQuery({
    queryKey: ['workspace-users', workspaceId],
    queryFn: () =>
      api.get(`workspaces/${workspaceId}/users`).json<WorkspaceUser[]>(),
    enabled: !!workspaceId,
  })
}

export function useCreateUser(workspaceId: string | undefined) {
  return useMutation({
    mutationFn: (body: { email: string; name: string; password: string; role: string }) =>
      api
        .post(`workspaces/${workspaceId}/users`, { json: body })
        .json<WorkspaceUser>(),
    onSuccess: () => {
      if (workspaceId) invalidateAll(workspaceId)
    },
  })
}

export function useUpdateUser(workspaceId: string | undefined) {
  return useMutation({
    mutationFn: ({ userId, name }: { userId: string; name: string }) =>
      api
        .patch(`workspaces/${workspaceId}/users/${userId}`, { json: { name } })
        .json<User>(),
    onSuccess: () => {
      if (workspaceId) invalidateAll(workspaceId)
    },
  })
}

export function useResetUserPassword(workspaceId: string | undefined) {
  return useMutation({
    mutationFn: ({ userId, password }: { userId: string; password: string }) =>
      api.post(`workspaces/${workspaceId}/users/${userId}/reset-password`, {
        json: { password },
      }),
    // No onSuccess invalidate — the user list shape doesn't change. The
    // hook's caller can show a toast on success themselves.
  })
}

export function useDeleteUser(workspaceId: string | undefined) {
  return useMutation({
    mutationFn: (userId: string) =>
      api.delete(`workspaces/${workspaceId}/users/${userId}`),
    onSuccess: () => {
      if (workspaceId) invalidateAll(workspaceId)
    },
  })
}

// ── Global admin (no workspace context) ──────────────────────────────────
//
// Backed by /api/v1/admin/users. Authorized as `user.manage on at least
// one workspace`, so workspace owners can create accounts that don't yet
// belong to any workspace. Users created here log in successfully but
// land on an empty /workspaces page until invited.

export interface AdminUserListResponse {
  users: User[]
  total: number
  limit: number
  offset: number
}

const invalidateGlobal = () => {
  queryClient.invalidateQueries({ queryKey: ['admin-users'] })
}

export function useAllUsers() {
  return useQuery({
    queryKey: ['admin-users'],
    queryFn: () => api.get('admin/users').json<AdminUserListResponse>(),
  })
}

export function useGlobalCreateUser() {
  return useMutation({
    mutationFn: (body: { email: string; name: string; password: string }) =>
      api.post('admin/users', { json: body }).json<User>(),
    onSuccess: () => invalidateGlobal(),
  })
}

export function useGlobalUpdateUser() {
  return useMutation({
    mutationFn: ({ userId, name }: { userId: string; name: string }) =>
      api.patch(`admin/users/${userId}`, { json: { name } }).json<User>(),
    onSuccess: () => invalidateGlobal(),
  })
}

export function useGlobalResetPassword() {
  return useMutation({
    mutationFn: ({ userId, password }: { userId: string; password: string }) =>
      api.post(`admin/users/${userId}/reset-password`, { json: { password } }),
  })
}

export function useGlobalDeleteUser() {
  return useMutation({
    mutationFn: (userId: string) => api.delete(`admin/users/${userId}`),
    onSuccess: () => invalidateGlobal(),
  })
}
