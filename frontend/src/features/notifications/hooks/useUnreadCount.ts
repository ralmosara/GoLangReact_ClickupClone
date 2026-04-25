import { useQuery } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { rooms } from '../../../lib/ws'
import { useWsEvent, useWsRooms } from '../../../hooks/useWebSocket'
import { useAuthStore } from '../../../store/auth'

export function useUnreadCount() {
  const userId = useAuthStore((s) => s.user?.id)

  useWsRooms(userId ? [rooms.user(userId)] : [])
  useWsEvent('notification.created', () => {
    queryClient.invalidateQueries({ queryKey: ['notifications-unread'] })
    queryClient.invalidateQueries({ queryKey: ['notifications'] })
  })

  const { data } = useQuery({
    queryKey: ['notifications-unread'],
    queryFn: () => api.get('me/notifications/unread-count').json<{ count: number }>(),
    enabled: !!userId,
    refetchOnWindowFocus: true,
  })
  return data?.count ?? 0
}
