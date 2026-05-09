// useIsAdmin — single source of truth for "is the current user an admin?"
//
// "Admin" here means: the user is assigned the literal built-in `admin`
// role in at least one workspace. Owners and members are NOT admins for
// this purpose, even though owners have a superset of admin's
// permissions. That's a deliberate restriction so /admin/users is
// gated on role identity, not permission set — matches the user-facing
// label "admin" rather than "anyone with user.manage".
//
// We re-use the same TanStack query keys WorkspacesPage already
// populates (`['workspaces']` + `['workspace-effective-permissions',
// id]`), so navigating from /workspaces → /admin/users hits the cache
// and renders without a network roundtrip.
//
// Note: the *backend* /admin/users endpoint still gates on the
// user.manage permission (so owners can also call it from curl) — this
// hook only narrows the *UI* surface. If you want to also lock the
// backend to literal admin, add a strict role check on the
// AdminGlobalRoutes in handler/user.
import { useQueries, useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import type { Workspace } from '../types'

interface EffectivePermissions {
  workspace_id: string
  user_id: string
  permissions: string[]
  // Newly surfaced — the role name(s) the user holds in this workspace.
  // Older backend deploys may omit this; the hook tolerates absent.
  roles?: string[]
}

// The literal built-in role name that grants /admin/users access.
const ADMIN_ROLE = 'admin'

export type AdminCheck = {
  /** true while the underlying queries are still loading. */
  loading: boolean
  /** true once at least one workspace returned the `admin` role. */
  isAdmin: boolean
  /** Workspaces where the caller is `admin` — useful for deep-links. */
  adminWorkspaceIds: string[]
}

export function useIsAdmin(): AdminCheck {
  const wsQ = useQuery({
    queryKey: ['workspaces'],
    queryFn: () => api.get('workspaces').json<Workspace[] | null>(),
    staleTime: 30_000,
  })

  const workspaces = wsQ.data ?? []

  const permsQueries = useQueries({
    queries: workspaces.map((w) => ({
      queryKey: ['workspace-effective-permissions', w.id],
      queryFn: () =>
        api
          .get(`workspaces/${w.id}/roles/effective`)
          .json<EffectivePermissions>(),
      staleTime: 30_000,
    })),
  })

  const stillLoading =
    wsQ.isLoading || permsQueries.some((q) => q.isLoading)

  const adminWorkspaceIds: string[] = []
  workspaces.forEach((w, i) => {
    const roles = permsQueries[i]?.data?.roles ?? []
    if (roles.includes(ADMIN_ROLE)) adminWorkspaceIds.push(w.id)
  })

  return {
    loading: stillLoading,
    isAdmin: adminWorkspaceIds.length > 0,
    adminWorkspaceIds,
  }
}
