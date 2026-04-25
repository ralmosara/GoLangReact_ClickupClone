import { useEffect, useRef } from 'react'
import { wsClient, type WsEvent } from '../lib/ws'
import { useAuthStore } from '../store/auth'

/**
 * Subscribe to a single (room, type) pair. Legacy; prefer useWsRooms + useWsEvent.
 */
export function useWsRoom(room: string, type: string, handler: (e: WsEvent) => void) {
  const token = useAuthStore((s) => s.token)
  const handlerRef = useRef(handler)
  useEffect(() => { handlerRef.current = handler }, [handler])

  useEffect(() => {
    if (!token) return
    wsClient.connect(token)
    wsClient.join(room)
    const off = wsClient.on(type, (e) => handlerRef.current(e))
    return () => {
      off()
      wsClient.leave(room)
    }
  }, [room, type, token])
}

/**
 * Join multiple rooms for the lifetime of the component.
 */
export function useWsRooms(rooms: string[]) {
  const token = useAuthStore((s) => s.token)
  const key = rooms.join('|')
  useEffect(() => {
    if (!token) return
    wsClient.connect(token)
    rooms.forEach((r) => wsClient.join(r))
    return () => {
      rooms.forEach((r) => wsClient.leave(r))
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, token])
}

/**
 * Subscribe to a specific event type. The handler identity is stabilised via a
 * ref so inline arrow-function handlers don't re-subscribe on every render.
 */
export function useWsEvent<P = unknown>(type: string, handler: (e: WsEvent<P>) => void) {
  const token = useAuthStore((s) => s.token)
  const handlerRef = useRef(handler)
  useEffect(() => { handlerRef.current = handler }, [handler])

  useEffect(() => {
    if (!token) return
    wsClient.connect(token)
    const off = wsClient.on<P>(type, (e) => handlerRef.current(e))
    return off
  }, [type, token])
}
