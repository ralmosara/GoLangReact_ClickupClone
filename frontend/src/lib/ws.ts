/**
 * WebSocket client for the v1 envelope shape:
 *   { v, room, type, workspace_id?, actor_id?, entity_id?, ts, payload? }
 * The client keeps one long-lived socket and dispatches events to typed listeners.
 *
 * Auth: browsers can't attach custom headers to `new WebSocket()`, so we pass
 * the JWT as a `?token=<jwt>` query param. The backend `ParseToken` accepts
 * either that or `Authorization: Bearer`, so the same validation path applies.
 */

export interface WsEvent<P = unknown> {
  v?: number
  room: string
  type: string
  workspace_id?: string
  actor_id?: string
  entity_id?: string
  ts?: string
  payload?: P
}

type Listener = (e: WsEvent) => void

export class WsClient {
  private ws: WebSocket | null = null
  private listenersByType = new Map<string, Set<Listener>>()
  private rooms = new Set<string>()
  private connecting = false
  private currentToken: string | null = null

  /**
   * Open or reuse the socket. When the token differs from the previously used
   * one, the old socket is closed and re-opened so the new identity takes
   * effect (e.g. after login/logout).
   */
  connect(token: string | null | undefined) {
    const next = token ?? null

    // Token changed? Close current socket so it reconnects with the new one.
    if (this.ws && next !== this.currentToken) {
      try { this.ws.close() } catch { /* ignore */ }
      this.ws = null
    }
    this.currentToken = next

    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }
    if (this.connecting) return
    // Don't attempt to open without a token — the server will 401 the upgrade.
    if (!next) return

    this.connecting = true
    const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${scheme}://${location.host}/ws?token=${encodeURIComponent(next)}`
    this.ws = new WebSocket(url)
    this.ws.onopen = () => {
      this.connecting = false
      this.rooms.forEach((room) => this.send('join', room))
    }
    this.ws.onmessage = (m) => {
      try {
        const e: WsEvent = JSON.parse(m.data)
        this.listenersByType.get(e.type)?.forEach((fn) => fn(e))
      } catch {
        // ignore malformed
      }
    }
    this.ws.onclose = () => {
      this.connecting = false
      this.ws = null
      // Only reconnect if we still have a token — otherwise we'd loop 401s.
      if (this.currentToken) {
        setTimeout(() => this.connect(this.currentToken), 1500)
      }
    }
  }

  private send(action: 'join' | 'leave', room: string) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action, room }))
    }
  }

  join(room: string) {
    if (this.rooms.has(room)) return
    this.rooms.add(room)
    this.send('join', room)
  }

  leave(room: string) {
    if (!this.rooms.has(room)) return
    this.rooms.delete(room)
    this.send('leave', room)
  }

  on<P = unknown>(type: string, fn: (e: WsEvent<P>) => void) {
    if (!this.listenersByType.has(type)) this.listenersByType.set(type, new Set())
    this.listenersByType.get(type)!.add(fn as Listener)
    return () => {
      this.listenersByType.get(type)?.delete(fn as Listener)
    }
  }

  disconnect() {
    this.currentToken = null
    this.ws?.close()
    this.ws = null
  }
}

export const wsClient = new WsClient()

// Room naming — keep in sync with internal/ws/event.go helpers.
export const rooms = {
  workspace: (id: string) => `ws:${id}`,
  list: (id: string) => `list:${id}`,
  task: (id: string) => `task:${id}`,
  user: (id: string) => `user:${id}`,
  doc: (id: string) => `doc:${id}`,
  channel: (id: string) => `channel:${id}`,
}
