import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { queryClient } from '../lib/queryClient'
import { useAuthStore } from '../store/auth'
import type { Member, User } from '../types'

interface AuthResult {
  token: string
  user: User
}

export function useLogin() {
  const { setAuth } = useAuthStore()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: (body: { email: string; password: string }) =>
      api.post('auth/login', { json: body }).json<AuthResult>(),
    onSuccess: (data) => {
      setAuth(data.token, data.user)
      navigate('/workspaces')
    },
  })
}

// useInviteMember adds an EXISTING user to a workspace. User creation
// happens on the User Management page (/workspaces/{id}/users) — when the
// email here doesn't map to a user, the server returns 404 and the
// MembersPage shows a deep-link to User Management.
export function useInviteMember(workspaceId: string | undefined) {
  return useMutation({
    mutationFn: (body: { email: string; role: string }) =>
      api
        .post(`workspaces/${workspaceId}/members`, { json: body })
        .json<Member>(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspace-members', workspaceId] })
    },
  })
}

export function useLogout() {
  const { clear } = useAuthStore()
  const navigate = useNavigate()
  return () => {
    clear()
    navigate('/login')
  }
}
