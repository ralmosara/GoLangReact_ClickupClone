import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuthStore } from '../store/auth'
import type { User } from '../types'

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

export function useRegister() {
  const { setAuth } = useAuthStore()
  const navigate = useNavigate()

  return useMutation({
    mutationFn: (body: { email: string; password: string; name: string }) =>
      api.post('auth/register', { json: body }).json<AuthResult>(),
    onSuccess: (data) => {
      setAuth(data.token, data.user)
      navigate('/workspaces')
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
