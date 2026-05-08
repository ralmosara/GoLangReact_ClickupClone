import ky from 'ky'
import { useAuthStore } from '../store/auth'

export const api = ky.extend({
  prefixUrl: '/api/v1',
  timeout: 15_000,
  hooks: {
    beforeRequest: [
      (req) => {
        const token = useAuthStore.getState().token
        if (token) req.headers.set('Authorization', `Bearer ${token}`)
      },
    ],
    afterResponse: [
      async (_req, _opts, res) => {
        if (res.status === 401) {
          useAuthStore.getState().clear()
          if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
            window.location.href = '/login'
          }
        }
      },
    ],
  },
})
