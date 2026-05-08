import { Navigate, Route, Routes, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from './lib/api'
import type { Workspace } from './types'
import { LoginPage } from './features/auth/LoginPage'
import { WorkspacesPage } from './features/workspace/index'
import { WorkspaceLayout } from './features/workspace/route'
import { BoardPage } from './features/board/index'
import { TaskDetailPage } from './features/task-detail/index'
import { SettingsPage } from './features/settings/index'
import { NotificationsPage } from './features/notifications/index'
import { MembersPage } from './features/members/MembersPage'
import { AuditPage } from './features/audit/AuditPage'
import { TimeReportPage } from './features/time-tracking/TimeReportPage'
import { AutomationsPage } from './features/automations/AutomationsPage'
import { DocsPage } from './features/docs/DocsPage'
import { ChatPage } from './features/chat/ChatPage'
import { GoalsPage } from './features/goals/GoalsPage'
import { SprintsPage } from './features/sprints/SprintsPage'
import { DashboardsPage } from './features/dashboards/DashboardsPage'
import { WhiteboardsPage } from './features/whiteboards/WhiteboardsPage'
import { FormsPage } from './features/forms/FormsPage'
import { PublicFormPage } from './features/forms/PublicFormPage'
import { TemplatesPage } from './features/templates/TemplatesPage'
import { CredentialsPage } from './features/credentials/CredentialsPage'
import { AccomplishmentsPage } from './features/accomplishments/AccomplishmentsPage'
import { useAuthStore } from './store/auth'

function RequireAuth({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

function RequireWorkspaceMember({ children }: { children: React.ReactNode }) {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const { data, isLoading, isError } = useQuery({
    queryKey: ['workspaces'],
    queryFn: () => api.get('workspaces').json<Workspace[] | null>(),
  })

  if (isLoading) {
    return <div className="min-h-screen bg-canvas" aria-hidden="true" />
  }
  if (isError) return <Navigate to="/workspaces" replace />

  const workspaces = data ?? []
  const isMember = workspaces.some((w) => w.id === workspaceId)
  if (!isMember) return <Navigate to="/workspaces" replace />
  return <>{children}</>
}

function WorkspaceHome() {
  return (
    <div className="flex flex-col items-center justify-center h-full text-center p-12 min-h-[400px]">
      <div className="w-14 h-14 rounded-2xl bg-brand-50 border border-brand-100 flex items-center justify-center mx-auto mb-5">
        <svg className="w-7 h-7 text-brand-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
        </svg>
      </div>
      <h2 className="text-base font-semibold text-ink-1 mb-1.5">Pick a list to get started</h2>
      <p className="text-sm text-ink-4 max-w-xs leading-relaxed">
        Select a list from the sidebar, or create a new Space and List first.
      </p>
    </div>
  )
}

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/forms/:formId" element={<PublicFormPage />} />
      <Route path="/" element={<Navigate to="/workspaces" replace />} />

      <Route
        path="/workspaces"
        element={
          <RequireAuth>
            <WorkspacesPage />
          </RequireAuth>
        }
      />

      <Route
        path="/workspaces/:workspaceId"
        element={
          <RequireAuth>
            <RequireWorkspaceMember>
              <WorkspaceLayout />
            </RequireWorkspaceMember>
          </RequireAuth>
        }
      >
        <Route index element={<WorkspaceHome />} />
        <Route path="lists/:listId" element={<BoardPage />} />
        <Route path="tasks/:taskId" element={<TaskDetailPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="notifications" element={<NotificationsPage />} />
        <Route path="members" element={<MembersPage />} />
        <Route path="audit" element={<AuditPage />} />
        <Route path="time-report" element={<TimeReportPage />} />
        <Route path="automations" element={<AutomationsPage />} />
        <Route path="docs" element={<DocsPage />} />
        <Route path="chat" element={<ChatPage />} />
        <Route path="goals" element={<GoalsPage />} />
        <Route path="sprints" element={<SprintsPage />} />
        <Route path="dashboards" element={<DashboardsPage />} />
        <Route path="whiteboards" element={<WhiteboardsPage />} />
        <Route path="forms" element={<FormsPage />} />
        <Route path="templates" element={<TemplatesPage />} />
        <Route path="credentials" element={<CredentialsPage />} />
        <Route path="accomplishments" element={<AccomplishmentsPage />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
