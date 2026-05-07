import { Component, type ReactNode, useMemo, useState, useEffect } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, Outlet, useMatch, useNavigate, useParams } from 'react-router-dom'
import * as Popover from '@radix-ui/react-popover'
import {
  Activity as ActivityIcon,
  BarChart3,
  Bell,
  Bot,
  Check,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ChevronsLeft,
  ClipboardList,
  Clock3,
  Copy as CopyIcon,
  FileText,
  Flag,
  Inbox,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Menu,
  MessageSquare,
  MoreHorizontal,
  Palette,
  Pencil,
  Plus,
  Presentation,
  Search,
  Settings,
  Target,
  Trash2,
  Users,
  X,
} from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { useLogout } from '../../hooks/useAuth'
import { Modal, Button, Input } from '../../components/ui'
import { useAuthStore } from '../../store/auth'
import { cn } from '../../lib/utils'
import type { List, Space, Workspace } from '../../types'
import { NotificationToast } from '../notifications/components/NotificationToast'
import { useUnreadCount } from '../notifications/hooks/useUnreadCount'
import { SearchPalette } from '../search/SearchPalette'

/* -------------------------------------------------------------------------- */
/* Modals                                                                     */
/* -------------------------------------------------------------------------- */

function CreateSpaceModal({ workspaceId, onClose }: { workspaceId: string; onClose: () => void }) {
  const [name, setName] = useState('')
  const [color, setColor] = useState('#6366f1')

  const mut = useMutation({
    mutationFn: () =>
      api.post('spaces', { json: { workspace_id: workspaceId, name, color } }).json<Space>(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['spaces', workspaceId] })
      onClose()
    },
  })

  return (
    <Modal open title="New space" description="Spaces group related lists." onClose={onClose}>
      <div className="space-y-4">
        <Input
          label="Space name"
          placeholder="e.g. Engineering"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && mut.mutate()}
          autoFocus
        />
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-2">Accent</label>
          <div className="flex items-center gap-3">
            <input
              type="color"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              className="h-9 w-14 rounded-lg border border-ink-5/40 cursor-pointer bg-transparent"
            />
            <span className="text-xs text-ink-4 font-mono">{color}</span>
          </div>
        </div>
        {mut.error && <p className="text-red-500 text-xs">{String(mut.error)}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => mut.mutate()} disabled={!name.trim() || mut.isPending} loading={mut.isPending}>
            Create space
          </Button>
        </div>
      </div>
    </Modal>
  )
}

function EditSpaceModal({
  space,
  focus = 'name',
  onClose,
}: {
  space: Space
  focus?: 'name' | 'color'
  onClose: () => void
}) {
  const [name, setName] = useState(space.name)
  const [color, setColor] = useState(space.color || '#6366f1')

  const mut = useMutation({
    mutationFn: () =>
      api.patch(`spaces/${space.id}`, { json: { name, color } }).json<Space>(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['spaces', space.workspace_id] })
      onClose()
    },
  })

  const dirty = name.trim() !== space.name || color !== (space.color || '#6366f1')
  const canSave = !!name.trim() && dirty && !mut.isPending

  return (
    <Modal open title="Edit space" description="Rename or change the accent color." onClose={onClose}>
      <div className="space-y-4">
        <Input
          label="Space name"
          placeholder="e.g. Engineering"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && canSave && mut.mutate()}
          autoFocus={focus === 'name'}
        />
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-2">Accent</label>
          <div className="flex items-center gap-3">
            <input
              type="color"
              value={color}
              onChange={(e) => setColor(e.target.value)}
              autoFocus={focus === 'color'}
              className="h-9 w-14 rounded-lg border border-ink-5/40 cursor-pointer bg-transparent"
            />
            <span className="text-xs text-ink-4 font-mono">{color}</span>
          </div>
        </div>
        {mut.error && <p className="text-red-500 text-xs">{String(mut.error)}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => mut.mutate()} disabled={!canSave} loading={mut.isPending}>
            Save changes
          </Button>
        </div>
      </div>
    </Modal>
  )
}

function CreateListModal({
  space, workspaceId, onClose,
}: {
  space: Space
  workspaceId: string
  onClose: () => void
}) {
  const navigate = useNavigate()
  const [name, setName] = useState('')

  const mut = useMutation({
    mutationFn: () =>
      api.post('lists', { json: { space_id: space.id, name } }).json<List>(),
    onSuccess: (list) => {
      queryClient.invalidateQueries({ queryKey: ['lists', space.id] })
      onClose()
      navigate(`/workspaces/${workspaceId}/lists/${list.id}`)
    },
  })

  return (
    <Modal open title="New list" description={`Adding to "${space.name}"`} onClose={onClose}>
      <div className="space-y-4">
        <Input
          label="List name"
          placeholder="e.g. Sprint Backlog"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && mut.mutate()}
          autoFocus
        />
        {mut.error && <p className="text-red-500 text-xs">{String(mut.error)}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => mut.mutate()} disabled={!name.trim() || mut.isPending} loading={mut.isPending}>
            Create list
          </Button>
        </div>
      </div>
    </Modal>
  )
}

/* -------------------------------------------------------------------------- */
/* Layout                                                                     */
/* -------------------------------------------------------------------------- */

export function WorkspaceLayout() {
  const { workspaceId, listId, taskId } = useParams<{ workspaceId: string; listId?: string; taskId?: string }>()
  const [createSpace, setCreateSpace] = useState(false)
  const [sidebarOpen, setSidebarOpen] = useState(false)

  // Auto-close sidebar on mobile when navigating
  useEffect(() => {
    setSidebarOpen(false)
  }, [listId, taskId])

  return (
    <div className="flex h-screen overflow-hidden bg-canvas relative">
      {/* Mobile Sidebar Overlay */}
      {sidebarOpen && (
        <div
            className="fixed inset-0 bg-black/50 z-40 lg:hidden backdrop-blur-sm transition-opacity"
            onClick={() => setSidebarOpen(false)}
        />
      )}

      <div className={cn(
        "fixed inset-y-0 left-0 z-50 transform lg:relative lg:translate-x-0 transition-transform duration-300 ease-in-out",
        sidebarOpen ? "translate-x-0" : "-translate-x-full"
      )}>
        <LayoutErrorBoundary>
            <Sidebar workspaceId={workspaceId ?? ''} onCreateSpace={() => setCreateSpace(true)} onClose={() => setSidebarOpen(false)} />
        </LayoutErrorBoundary>
      </div>

      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {/* Mobile Header */}
        <header className="lg:hidden flex items-center justify-between h-14 px-4 bg-surface border-b border-ink-5/20 shrink-0">
            <button
                onClick={() => setSidebarOpen(true)}
                className="p-2 -ml-2 text-ink-3 hover:text-ink-1"
            >
                <Menu className="w-6 h-6" />
            </button>
            <div className="flex-1 px-4 text-center">
                 <span className="text-sm font-bold text-ink-1 truncate block">Clikr</span>
            </div>
            <div className="w-10" /> {/* spacer */}
        </header>

        <main className="flex-1 overflow-y-auto bg-canvas">
            <LayoutErrorBoundary>
            <Outlet />
            </LayoutErrorBoundary>
        </main>
      </div>

      {createSpace && workspaceId && (
        <CreateSpaceModal workspaceId={workspaceId} onClose={() => setCreateSpace(false)} />
      )}
      {workspaceId && <NotificationToast workspaceId={workspaceId} />}
      <SearchPalette />
    </div>
  )
}

// Error boundary keeps one broken pane from white-screening the whole page.
class LayoutErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null }
  static getDerivedStateFromError(error: Error) { return { error } }
  componentDidCatch(error: Error, info: unknown) {
    // eslint-disable-next-line no-console
    console.error('[layout]', error, info)
  }
  render() {
    if (this.state.error) {
      return (
        <div className="p-6 max-w-lg m-auto my-8 bg-surface border border-red-200 rounded-xl shadow-card">
          <p className="text-sm font-semibold text-red-700 mb-1">Something broke rendering this area.</p>
          <p className="text-xs text-ink-3 mb-3">The rest of the app is still usable. Reload to retry.</p>
          <pre className="text-[10px] bg-canvas/60 border border-ink-5/20 rounded p-2 overflow-auto max-h-48 whitespace-pre-wrap">
            {String(this.state.error?.stack ?? this.state.error?.message ?? this.state.error)}
          </pre>
          <button
            onClick={() => { this.setState({ error: null }); window.location.reload() }}
            className="mt-3 text-xs font-medium text-brand-600 hover:text-brand-700"
          >
            Reload
          </button>
        </div>
      )
    }
    return this.props.children
  }
}

/* -------------------------------------------------------------------------- */
/* Sidebar                                                                    */
/* -------------------------------------------------------------------------- */

function Sidebar({ workspaceId, onCreateSpace, onClose }: { workspaceId: string; onCreateSpace: () => void; onClose: () => void }) {
  // Go nil slices serialize as JSON null, and destructuring default only fires
  // for `undefined` — so we null-coalesce explicitly to keep .map() safe.
  const spacesQ = useQuery({
    queryKey: ['spaces', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/spaces`).json<Space[] | null>(),
    enabled: !!workspaceId,
  })
  const spaces = spacesQ.data ?? []

  return (
    <aside className="relative w-[280px] lg:w-[252px] h-full shrink-0 flex flex-col select-none bg-gradient-to-b from-[#131320] to-[#0B0B13] text-sidebar-text border-r border-white/[0.06]">
      <button
        onClick={onClose}
        className="absolute top-4 right-4 p-2 text-white/40 hover:text-white lg:hidden z-50"
      >
        <X className="w-5 h-5" />
      </button>

      {/* Ambient accents — purely decorative, soft brand tint top-left */}
      <div className="pointer-events-none absolute -top-24 -left-16 w-64 h-64 bg-brand-500/10 rounded-full blur-3xl" />
      <div className="pointer-events-none absolute top-1/3 -right-20 w-60 h-60 bg-brand-700/8 rounded-full blur-3xl" />

      <WorkspaceSwitcher workspaceId={workspaceId} />

      <div className="px-3 pt-3 space-y-1.5 relative">
        <SearchLauncher />
        <PrimaryLink
          to={`/workspaces/${workspaceId}/notifications`}
          icon={<Inbox className="w-4 h-4" />}
          label="Inbox"
          badge={<InboxBadge />}
        />
        <PrimaryLink
          to={`/workspaces/${workspaceId}/credentials`}
          icon={<KeyRound className="w-4 h-4" />}
          label="Credentials"
        />
      </div>

      <nav className="flex-1 overflow-y-auto px-3 pt-4 pb-3 relative">
        <Section label="Work">
          <NavItem to={`/workspaces/${workspaceId}/docs`}        icon={<FileText className="w-4 h-4" />}       label="Docs" />
          <NavItem to={`/workspaces/${workspaceId}/chat`}        icon={<MessageSquare className="w-4 h-4" />}  label="Chat" />
          <NavItem to={`/workspaces/${workspaceId}/whiteboards`} icon={<Presentation className="w-4 h-4" />}   label="Whiteboards" />
        </Section>

        <Section label="Plan">
          <NavItem to={`/workspaces/${workspaceId}/dashboards`}  icon={<LayoutDashboard className="w-4 h-4" />} label="Dashboards" />
          <NavItem to={`/workspaces/${workspaceId}/goals`}       icon={<Target className="w-4 h-4" />}          label="Goals" />
          <NavItem to={`/workspaces/${workspaceId}/sprints`}     icon={<Flag className="w-4 h-4" />}            label="Sprints" />
          <NavItem to={`/workspaces/${workspaceId}/time-report`} icon={<Clock3 className="w-4 h-4" />}          label="Time report" />
          <NavItem to={`/workspaces/${workspaceId}/accomplishments`} icon={<CheckCircle2 className="w-4 h-4" />} label="Accomplishments" />
        </Section>

        <Section label="Automate">
          <NavItem to={`/workspaces/${workspaceId}/automations`} icon={<Bot className="w-4 h-4" />}           label="Automations" />
          <NavItem to={`/workspaces/${workspaceId}/forms`}       icon={<ClipboardList className="w-4 h-4" />} label="Forms" />
          <NavItem to={`/workspaces/${workspaceId}/templates`}   icon={<CopyIcon className="w-4 h-4" />}      label="Templates" />
        </Section>

        <Section
          label="Spaces"
          action={
            <SidebarIconButton title="New space" onClick={onCreateSpace}>
              <Plus className="w-3.5 h-3.5" />
            </SidebarIconButton>
          }
        >
          {spaces.map((sp) => (
            <SpaceSection key={sp.id} space={sp} workspaceId={workspaceId} />
          ))}
          {spaces.length === 0 && (
            <button
              onClick={onCreateSpace}
              className="w-full flex items-center gap-2 h-7 px-2 text-[12px] text-white/45 hover:text-white/80 rounded-md hover:bg-white/[0.04] transition-colors"
            >
              <Plus className="w-3.5 h-3.5" />
              New space
            </button>
          )}
        </Section>
      </nav>

      <UserRow workspaceId={workspaceId} />
    </aside>
  )
}

/* -------------------------------------------------------------------------- */
/* Workspace switcher                                                         */
/* -------------------------------------------------------------------------- */

function WorkspaceSwitcher({ workspaceId }: { workspaceId: string }) {
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()

  const workspacesQ = useQuery({
    queryKey: ['workspaces'],
    queryFn: () => api.get('workspaces').json<Workspace[] | null>(),
  })
  const workspaces = workspacesQ.data ?? []
  const current = workspaces.find((w) => w.id === workspaceId)

  const initials = useMemo(() => {
    const n = current?.name ?? 'W'
    return n.trim().split(/\s+/).slice(0, 2).map((p) => p[0]).join('').toUpperCase() || 'W'
  }, [current])

  return (
    <div className="relative px-3 pt-3">
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Trigger asChild>
          <button
            className={cn(
              'w-full flex items-center gap-2.5 px-2 h-10 rounded-lg',
              'bg-white/[0.04] hover:bg-white/[0.08] border border-white/[0.06]',
              'transition-colors group',
            )}
            aria-label="Switch workspace"
          >
            <div className="w-7 h-7 rounded-md bg-brand-gradient shadow-button flex items-center justify-center shrink-0">
              <span className="text-[10.5px] font-bold text-white tracking-tight">{initials}</span>
            </div>
            <div className="flex-1 min-w-0 text-left">
              <p className="text-[13px] font-semibold text-white truncate leading-tight">
                {current?.name ?? 'Workspace'}
              </p>
              <p className="text-[10.5px] text-white/45 truncate leading-tight mt-0.5">
                {current?.slug ? `${current.slug}.clikr` : 'click to switch'}
              </p>
            </div>
            <ChevronDown className="w-3.5 h-3.5 text-white/40 group-hover:text-white/70 shrink-0 transition-colors" />
          </button>
        </Popover.Trigger>

        <Popover.Portal>
          <Popover.Content
            align="start"
            sideOffset={6}
            className="w-[250px] bg-surface border border-ink-5/30 rounded-xl shadow-modal p-1.5 z-50 animate-fade-in"
          >
            <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-2.5 pt-1.5 pb-1">Workspaces</p>
            <div className="max-h-64 overflow-y-auto">
              {workspaces.map((w) => (
                <button
                  key={w.id}
                  onClick={() => { setOpen(false); navigate(`/workspaces/${w.id}`) }}
                  className={cn(
                    'flex items-center gap-2.5 w-full px-2 py-1.5 rounded-lg text-left transition-colors',
                    w.id === workspaceId ? 'bg-brand-50' : 'hover:bg-ink-1/[0.04]',
                  )}
                >
                  <div className="w-6 h-6 rounded-md bg-brand-gradient text-[10px] font-bold text-white flex items-center justify-center shrink-0">
                    {(w.name || '?').slice(0, 1).toUpperCase()}
                  </div>
                  <span className="flex-1 text-[13px] text-ink-1 truncate font-medium">{w.name}</span>
                  {w.id === workspaceId && <Check className="w-3.5 h-3.5 text-brand-500 shrink-0" />}
                </button>
              ))}
            </div>
            <div className="h-px bg-ink-5/20 my-1" />
            <Link
              to="/workspaces"
              onClick={() => setOpen(false)}
              className="flex items-center gap-2 px-2 py-1.5 rounded-lg text-[12.5px] text-ink-2 hover:bg-ink-1/[0.04] transition-colors"
            >
              <ChevronsLeft className="w-3.5 h-3.5 text-ink-4" />
              All workspaces
            </Link>
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
    </div>
  )
}

/* -------------------------------------------------------------------------- */
/* Search launcher + Primary link                                             */
/* -------------------------------------------------------------------------- */

function SearchLauncher() {
  const fireCmdK = () => {
    const isMac = typeof navigator !== 'undefined' && /Mac/.test(navigator.platform)
    window.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'k',
      code: 'KeyK',
      metaKey: isMac,
      ctrlKey: !isMac,
      bubbles: true,
    }))
  }
  return (
    <button
      onClick={fireCmdK}
      className={cn(
        'w-full flex items-center gap-2 h-8 px-2.5 rounded-lg',
        'bg-white/[0.04] hover:bg-white/[0.08] border border-white/[0.06]',
        'text-white/55 text-[12.5px] transition-colors',
      )}
    >
      <Search className="w-3.5 h-3.5" />
      <span className="flex-1 text-left">Search…</span>
      <kbd className="font-mono text-[10px] text-white/50 bg-white/[0.06] border border-white/[0.06] rounded px-1.5 py-0.5">⌘K</kbd>
    </button>
  )
}

function PrimaryLink({
  to, icon, label, badge,
}: {
  to: string
  icon: React.ReactNode
  label: string
  badge?: React.ReactNode
}) {
  // useMatch gives us isActive without the NavLink function-children pattern,
  // which was the white-screen culprit.
  const isActive = !!useMatch(to)
  return (
    <Link
      to={to}
      className={cn(
        'relative flex items-center gap-2.5 h-8 pl-2.5 pr-2 rounded-lg text-[12.5px] transition-colors',
        isActive
          ? 'bg-white/[0.08] text-white font-semibold'
          : 'text-white/75 hover:bg-white/[0.04] hover:text-white',
      )}
    >
      {isActive && <span className="absolute left-0 top-1.5 bottom-1.5 w-[3px] rounded-r bg-brand-400" />}
      <span className={cn('shrink-0', isActive ? 'text-brand-300' : 'text-white/55')}>{icon}</span>
      <span className="flex-1 truncate">{label}</span>
      {badge}
    </Link>
  )
}

function InboxBadge() {
  const unread = useUnreadCount()
  if (unread === 0) return null
  return (
    <span className="inline-flex items-center justify-center min-w-[18px] h-[18px] px-1 rounded-full bg-brand-500 text-white text-[10px] font-bold leading-none">
      {unread > 99 ? '99+' : unread}
    </span>
  )
}

/* -------------------------------------------------------------------------- */
/* Sections + nav items                                                       */
/* -------------------------------------------------------------------------- */

function Section({
  label, action, children,
}: {
  label: string
  action?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <div className="mb-4">
      <div className="flex items-center justify-between px-2 mb-1 h-6">
        <p className="text-[10px] font-bold uppercase tracking-wider text-white/40">{label}</p>
        {action}
      </div>
      <div className="space-y-[1px]">{children}</div>
    </div>
  )
}

function NavItem({
  to, icon, label, badge,
}: {
  to: string
  icon: React.ReactNode
  label: string
  badge?: React.ReactNode
}) {
  const isActive = !!useMatch(to)
  return (
    <Link
      to={to}
      className={cn(
        'relative flex items-center gap-2.5 h-8 pl-2.5 pr-2 rounded-lg text-[12.5px] transition-colors',
        isActive
          ? 'bg-white/[0.08] text-white font-semibold'
          : 'text-white/70 hover:bg-white/[0.04] hover:text-white',
      )}
    >
      {isActive && <span className="absolute left-0 top-1.5 bottom-1.5 w-[3px] rounded-r bg-brand-400" />}
      <span className={cn('shrink-0', isActive ? 'text-brand-300' : 'text-white/50')}>{icon}</span>
      <span className="flex-1 truncate">{label}</span>
      {badge}
    </Link>
  )
}

function SidebarIconButton({
  children, onClick, title,
}: {
  children: React.ReactNode
  onClick?: () => void
  title: string
}) {
  return (
    <button
      onClick={onClick}
      title={title}
      aria-label={title}
      className="h-6 w-6 flex items-center justify-center rounded text-white/45 hover:text-white hover:bg-white/[0.08] transition-colors"
    >
      {children}
    </button>
  )
}

/* -------------------------------------------------------------------------- */
/* Space tree                                                                 */
/* -------------------------------------------------------------------------- */

function SpaceSection({ space, workspaceId }: { space: Space; workspaceId: string }) {
  const [open, setOpen] = useState(true)
  const [createList, setCreateList] = useState(false)
  const [editing, setEditing] = useState<null | 'name' | 'color'>(null)

  const listsQ = useQuery({
    queryKey: ['lists', space.id],
    queryFn: () => api.get(`spaces/${space.id}/lists`).json<List[] | null>(),
  })
  const lists = listsQ.data ?? []

  return (
    <div className="mb-1">
      <div className="group/space flex items-center gap-1 px-1 h-7 rounded-md hover:bg-white/[0.04] transition-colors">
        <button
          onClick={() => setOpen((v) => !v)}
          className="h-5 w-5 flex items-center justify-center text-white/40 hover:text-white/80 rounded transition-colors"
          aria-label={open ? 'Collapse' : 'Expand'}
        >
          {open ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
        </button>
        <span
          className="w-2 h-2 rounded-full shrink-0 ring-[2px] ring-white/[0.04]"
          style={{ backgroundColor: space.color || '#6366f1' }}
        />
        <span className="flex-1 text-[12px] font-semibold text-white/85 truncate uppercase tracking-wide">
          {space.name}
        </span>
        <SpaceMenu
          space={space}
          onRename={() => setEditing('name')}
          onChangeColor={() => setEditing('color')}
        />
        <button
          onClick={() => setCreateList(true)}
          className="opacity-0 group-hover/space:opacity-100 h-5 w-5 flex items-center justify-center rounded text-white/50 hover:text-white hover:bg-white/[0.08] transition-all"
          title="Add list"
          aria-label="Add list"
        >
          <Plus className="w-3 h-3" />
        </button>
      </div>

      {open && (
        <div className="mt-0.5 ml-3 pl-2 border-l border-white/[0.06] space-y-[1px]">
          {lists.map((l) => (
            <ListRow
              key={l.id}
              to={`/workspaces/${workspaceId}/lists/${l.id}`}
              name={l.name}
            />
          ))}
          {lists.length === 0 && (
            <button
              onClick={() => setCreateList(true)}
              className="flex items-center gap-2 w-full h-7 px-2 text-[12px] text-white/40 hover:text-white/80 hover:bg-white/[0.04] rounded-md transition-colors"
            >
              <Plus className="w-3 h-3" />
              Add a list
            </button>
          )}
        </div>
      )}

      {createList && (
        <CreateListModal space={space} workspaceId={workspaceId} onClose={() => setCreateList(false)} />
      )}
      {editing && (
        <EditSpaceModal space={space} focus={editing} onClose={() => setEditing(null)} />
      )}
    </div>
  )
}

function SpaceMenu({
  space,
  onRename,
  onChangeColor,
}: {
  space: Space
  onRename: () => void
  onChangeColor: () => void
}) {
  const [open, setOpen] = useState(false)

  const del = useMutation({
    mutationFn: () => api.delete(`spaces/${space.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['spaces', space.workspace_id] }),
  })

  const handleDelete = () => {
    setOpen(false)
    if (!confirm(`Delete space "${space.name}"? This cannot be undone.`)) return
    del.mutate()
  }

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button
          className={cn(
            'h-5 w-5 flex items-center justify-center rounded text-white/50 hover:text-white hover:bg-white/[0.08] transition-all',
            open ? 'opacity-100 bg-white/[0.08] text-white' : 'opacity-0 group-hover/space:opacity-100',
          )}
          title="Space options"
          aria-label="Space options"
          onClick={(e) => e.stopPropagation()}
        >
          <MoreHorizontal className="w-3 h-3" />
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="end"
          sideOffset={6}
          className="w-[180px] bg-surface border border-ink-5/30 rounded-xl shadow-modal p-1.5 z-50 animate-fade-in"
        >
          <MenuItem
            icon={<Pencil className="w-4 h-4" />}
            onClick={() => { setOpen(false); onRename() }}
          >
            Rename
          </MenuItem>
          <MenuItem
            icon={<Palette className="w-4 h-4" />}
            onClick={() => { setOpen(false); onChangeColor() }}
          >
            Change color
          </MenuItem>
          <div className="h-px bg-ink-5/20 my-1" />
          <MenuItem
            icon={<Trash2 className="w-4 h-4" />}
            onClick={handleDelete}
            danger
          >
            Delete
          </MenuItem>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

function ListRow({ to, name }: { to: string; name: string }) {
  const isActive = !!useMatch(to)
  return (
    <Link
      to={to}
      className={cn(
        'relative flex items-center gap-2 h-7 px-2 rounded-md text-[12.5px] transition-colors',
        isActive
          ? 'bg-white/[0.08] text-white font-medium'
          : 'text-white/60 hover:bg-white/[0.04] hover:text-white/95',
      )}
    >
      {isActive && <span className="absolute -left-[9px] top-1.5 bottom-1.5 w-[2px] rounded bg-brand-400" />}
      <span className={cn('w-1 h-1 rounded-full shrink-0', isActive ? 'bg-brand-300' : 'bg-white/35')} />
      <span className="truncate">{name}</span>
    </Link>
  )
}

/* -------------------------------------------------------------------------- */
/* User row                                                                   */
/* -------------------------------------------------------------------------- */

function UserRow({ workspaceId }: { workspaceId: string }) {
  const user = useAuthStore((s) => s.user)
  const logout = useLogout()
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()

  const initial = user?.name?.charAt(0).toUpperCase() ?? user?.email?.charAt(0).toUpperCase() ?? '?'

  return (
    <div className="relative border-t border-white/[0.06] p-2">
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Trigger asChild>
          <button
            className={cn(
              'w-full flex items-center gap-2.5 h-11 px-2 rounded-lg',
              'hover:bg-white/[0.05] transition-colors group',
              open && 'bg-white/[0.05]',
            )}
          >
            <div className="relative shrink-0">
              <div className="w-7 h-7 rounded-full bg-brand-gradient flex items-center justify-center text-[11px] font-bold text-white">
                {initial}
              </div>
              <span className="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-green-500 ring-2 ring-[#0B0B13]" />
            </div>
            <div className="flex-1 min-w-0 text-left">
              <p className="text-[12.5px] font-semibold text-white truncate leading-tight">
                {user?.name || 'Account'}
              </p>
              <p className="text-[10.5px] text-white/45 truncate leading-tight mt-0.5">{user?.email}</p>
            </div>
            <ChevronDown className={cn(
              'w-3.5 h-3.5 text-white/40 shrink-0 transition-transform',
              open && 'rotate-180',
            )} />
          </button>
        </Popover.Trigger>

        <Popover.Portal>
          <Popover.Content
            align="start"
            side="top"
            sideOffset={6}
            className="w-[236px] bg-surface border border-ink-5/30 rounded-xl shadow-modal p-1.5 z-50 animate-fade-in"
          >
            <div className="px-2 py-2 border-b border-ink-5/20 mb-1">
              <p className="text-[13px] font-semibold text-ink-1 truncate">{user?.name || 'Account'}</p>
              <p className="text-[11px] text-ink-4 truncate">{user?.email}</p>
            </div>
            <MenuItem
              icon={<Users className="w-4 h-4" />}
              onClick={() => { setOpen(false); navigate(`/workspaces/${workspaceId}/members`) }}
            >
              Members
            </MenuItem>
            <MenuItem
              icon={<ActivityIcon className="w-4 h-4" />}
              onClick={() => { setOpen(false); navigate(`/workspaces/${workspaceId}/audit`) }}
            >
              Activity log
            </MenuItem>
            <MenuItem
              icon={<Bell className="w-4 h-4" />}
              onClick={() => { setOpen(false); navigate(`/workspaces/${workspaceId}/notifications`) }}
            >
              Notifications
            </MenuItem>
            <MenuItem
              icon={<BarChart3 className="w-4 h-4" />}
              onClick={() => { setOpen(false); navigate(`/workspaces/${workspaceId}/dashboards`) }}
            >
              Dashboards
            </MenuItem>
            <div className="h-px bg-ink-5/20 my-1" />
            <MenuItem
              icon={<Settings className="w-4 h-4" />}
              onClick={() => { setOpen(false); navigate(`/workspaces/${workspaceId}/settings`) }}
            >
              Workspace settings
            </MenuItem>
            <MenuItem
              icon={<LogOut className="w-4 h-4" />}
              onClick={logout}
              danger
            >
              Sign out
            </MenuItem>
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
    </div>
  )
}

function MenuItem({
  icon, children, onClick, danger,
}: {
  icon: React.ReactNode
  children: React.ReactNode
  onClick: () => void
  danger?: boolean
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'flex items-center gap-2.5 w-full px-2 py-1.5 text-[12.5px] text-left rounded-lg transition-colors',
        danger ? 'text-red-600 hover:bg-red-50' : 'text-ink-2 hover:bg-ink-1/[0.04]',
      )}
    >
      <span className={cn('shrink-0', danger ? 'text-red-500' : 'text-ink-3')}>{icon}</span>
      {children}
    </button>
  )
}
