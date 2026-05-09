package authz

// Permission keys.
//
// These constants MUST stay in sync with the seed in
// migrations/031_rbac.up.sql. The DB is the source of truth for runtime
// checks (the `permissions` table is queried during EffectivePermissions),
// but the constants below are what the rest of Go code references so a
// renamed key shows up as a compile error rather than a runtime forbidden.
//
// New permission? Add the constant here AND a row in the migration's
// `INSERT INTO permissions` block AND grant it to the appropriate
// built-in roles.
const (
	// Workspace
	PermWorkspaceRead           = "workspace.read"
	PermWorkspaceUpdate         = "workspace.update"
	PermWorkspaceDelete         = "workspace.delete"
	PermWorkspaceManageMembers  = "workspace.manage_members"

	// Spaces / folders / lists
	PermSpaceCreate  = "space.create"
	PermSpaceRead    = "space.read"
	PermSpaceUpdate  = "space.update"
	PermSpaceDelete  = "space.delete"
	PermFolderCreate = "folder.create"
	PermFolderUpdate = "folder.update"
	PermFolderDelete = "folder.delete"
	PermListCreate   = "list.create"
	PermListRead     = "list.read"
	PermListUpdate   = "list.update"
	PermListDelete   = "list.delete"

	// Tasks
	PermTaskCreate       = "task.create"
	PermTaskRead         = "task.read"
	PermTaskUpdate       = "task.update"
	PermTaskDelete       = "task.delete"
	PermTaskAssign       = "task.assign"
	PermTaskChangeStatus = "task.change_status"

	// Comments / attachments
	PermCommentCreate     = "comment.create"
	PermCommentUpdateOwn  = "comment.update_own"
	PermCommentDeleteAny  = "comment.delete_any"
	PermAttachmentUpload  = "attachment.upload"
	PermAttachmentDelete  = "attachment.delete"

	// Categorisation / customisation
	PermTagManage         = "tag.manage"
	PermStatusManage      = "status.manage"
	PermViewManage        = "view.manage"
	PermCustomFieldManage = "custom_field.manage"

	// Planning
	PermDashboardCreate = "dashboard.create"
	PermDashboardUpdate = "dashboard.update"
	PermDashboardDelete = "dashboard.delete"
	PermAutomationManage = "automation.manage"
	PermGoalManage       = "goal.manage"
	PermSprintManage     = "sprint.manage"

	// Knowledge / collaboration
	PermDocCreate         = "doc.create"
	PermDocUpdate         = "doc.update"
	PermDocDelete         = "doc.delete"
	PermChatSend          = "chat.send"
	PermChatManageChannel = "chat.manage_channels"
	PermWhiteboardManage  = "whiteboard.manage"
	PermFormManage        = "form.manage"
	PermTemplateManage    = "template.manage"

	// Compliance / data
	PermAuditRead   = "audit.read"
	PermAuditExport = "audit.export"
	PermGDPRExport  = "gdpr.export"

	// Credentials vault
	PermCredentialRead   = "credential.read"
	PermCredentialManage = "credential.manage"

	// Members & roles
	PermMemberInvite      = "member.invite"
	PermMemberRemove      = "member.remove"
	PermMemberManageRoles = "member.manage_roles"
	PermRoleManage        = "role.manage"

	// User accounts (the global users table). PermUserManage gates the
	// User Management page — separate from PermMemberInvite which only
	// adds an existing account to a workspace. Granted to owner + admin
	// built-in roles via migration 032.
	PermUserManage = "user.manage"

	// Time tracking
	PermTimeEntryCreate    = "time_entry.create"
	PermTimeEntryUpdateOwn = "time_entry.update_own"
	PermTimeEntryUpdateAny = "time_entry.update_any"

	// Reports
	PermReportRead = "report.read"

	// Security policy
	PermMFAEnforceWorkspace = "mfa.enforce_workspace"
)

// AllPermissions is the canonical list. The migration seeds the same set;
// keeping this list also lets tests assert "every code-defined permission
// has a DB row" by comparing against the permissions table.
var AllPermissions = []string{
	PermWorkspaceRead, PermWorkspaceUpdate, PermWorkspaceDelete, PermWorkspaceManageMembers,
	PermSpaceCreate, PermSpaceRead, PermSpaceUpdate, PermSpaceDelete,
	PermFolderCreate, PermFolderUpdate, PermFolderDelete,
	PermListCreate, PermListRead, PermListUpdate, PermListDelete,
	PermTaskCreate, PermTaskRead, PermTaskUpdate, PermTaskDelete,
	PermTaskAssign, PermTaskChangeStatus,
	PermCommentCreate, PermCommentUpdateOwn, PermCommentDeleteAny,
	PermAttachmentUpload, PermAttachmentDelete,
	PermTagManage, PermStatusManage, PermViewManage, PermCustomFieldManage,
	PermDashboardCreate, PermDashboardUpdate, PermDashboardDelete,
	PermAutomationManage, PermGoalManage, PermSprintManage,
	PermDocCreate, PermDocUpdate, PermDocDelete,
	PermChatSend, PermChatManageChannel,
	PermWhiteboardManage, PermFormManage, PermTemplateManage,
	PermAuditRead, PermAuditExport, PermGDPRExport,
	PermCredentialRead, PermCredentialManage,
	PermMemberInvite, PermMemberRemove, PermMemberManageRoles, PermRoleManage,
	PermUserManage,
	PermTimeEntryCreate, PermTimeEntryUpdateOwn, PermTimeEntryUpdateAny,
	PermReportRead,
	PermMFAEnforceWorkspace,
}
