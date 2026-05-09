-- 031_rbac.up.sql
-- Phase 2 — granular RBAC.
--
-- Adds three tables:
--   permissions      — flat catalog of well-known permission keys
--   roles            — built-in roles (workspace_id IS NULL) + per-workspace
--                      custom roles
--   role_permissions — many-to-many between roles and permissions
--
-- And one column:
--   workspace_members.role_id — optional FK to roles.id; when set it OVERRIDES
--                               the legacy `role` text column. When NULL, the
--                               legacy text-name-based lookup keeps working
--                               for backwards-compatible deployments.
--
-- The four built-in roles (owner / admin / member / guest) are seeded with
-- permission assignments at the bottom of this file. The permissions list
-- mirrors the Go constants in internal/authz/permissions.go — the two MUST
-- be kept in sync; CI can grep both files to detect drift.

-- ── catalog ──────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS permissions (
    key         TEXT PRIMARY KEY,
    category    TEXT NOT NULL,             -- 'task', 'workspace', 'audit', etc. (UI grouping)
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── roles ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS roles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE, -- NULL = built-in
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    is_builtin   BOOLEAN NOT NULL DEFAULT FALSE,
    rank         INT  NOT NULL DEFAULT 2,   -- compatibility with the legacy authz.RoleRank ladder
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- A built-in role is unique by name across the catalog.
CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_builtin_name
    ON roles(name) WHERE workspace_id IS NULL;
-- A custom role is unique by name within a workspace.
CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_workspace_name
    ON roles(workspace_id, name) WHERE workspace_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_roles_workspace ON roles(workspace_id);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id        UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_key TEXT NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_key)
);
CREATE INDEX IF NOT EXISTS idx_role_permissions_perm
    ON role_permissions(permission_key);

-- ── workspace_members.role_id ────────────────────────────────────────────

ALTER TABLE workspace_members
    ADD COLUMN IF NOT EXISTS role_id UUID REFERENCES roles(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_workspace_members_role_id
    ON workspace_members(role_id);

-- ── seed: permissions catalog ───────────────────────────────────────────
-- Categories are mirrored in internal/authz/permissions.go. Adding a new
-- key here AND in that file is the only safe way to introduce a permission.

INSERT INTO permissions (key, category, description) VALUES
    -- workspace
    ('workspace.read',            'workspace', 'View workspace settings & metadata'),
    ('workspace.update',          'workspace', 'Edit workspace name, slug, branding'),
    ('workspace.delete',          'workspace', 'Permanently delete the workspace'),
    ('workspace.manage_members',  'workspace', 'Invite, remove, and re-role members'),
    -- spaces / folders / lists
    ('space.create',              'space',     'Create new spaces'),
    ('space.read',                'space',     'View spaces and their contents'),
    ('space.update',              'space',     'Rename/recolor/archive spaces'),
    ('space.delete',              'space',     'Delete spaces'),
    ('folder.create',             'folder',    'Create folders within a space'),
    ('folder.update',             'folder',    'Edit folder name/position'),
    ('folder.delete',             'folder',    'Delete folders'),
    ('list.create',               'list',      'Create lists'),
    ('list.read',                 'list',      'View lists'),
    ('list.update',               'list',      'Rename/move lists'),
    ('list.delete',               'list',      'Delete lists'),
    -- tasks
    ('task.create',               'task',      'Create tasks'),
    ('task.read',                 'task',      'View tasks'),
    ('task.update',               'task',      'Edit task fields'),
    ('task.delete',               'task',      'Delete tasks'),
    ('task.assign',               'task',      'Add or remove assignees'),
    ('task.change_status',        'task',      'Move tasks across statuses'),
    -- comments / attachments
    ('comment.create',            'comment',   'Post comments'),
    ('comment.update_own',        'comment',   'Edit your own comments'),
    ('comment.delete_any',        'comment',   'Delete any comment (moderation)'),
    ('attachment.upload',         'attachment','Upload files to tasks'),
    ('attachment.delete',         'attachment','Delete attachments'),
    -- categorisation
    ('tag.manage',                'tag',       'Create / edit / delete tags'),
    ('status.manage',             'status',    'Create / edit / delete statuses'),
    ('view.manage',               'view',      'Create / edit / delete saved views'),
    ('custom_field.manage',       'task',      'Create / edit / delete custom fields'),
    -- planning
    ('dashboard.create',          'dashboard', 'Create dashboards'),
    ('dashboard.update',          'dashboard', 'Edit dashboards'),
    ('dashboard.delete',          'dashboard', 'Delete dashboards'),
    ('automation.manage',         'automation','Create / edit / disable automations'),
    ('goal.manage',               'goal',      'Manage workspace goals'),
    ('sprint.manage',             'sprint',    'Plan and close sprints'),
    -- knowledge / collaboration
    ('doc.create',                'doc',       'Create docs'),
    ('doc.update',                'doc',       'Edit docs'),
    ('doc.delete',                'doc',       'Delete docs'),
    ('chat.send',                 'chat',      'Post chat messages'),
    ('chat.manage_channels',      'chat',      'Create / archive channels'),
    ('whiteboard.manage',         'whiteboard','Manage whiteboards'),
    ('form.manage',               'form',      'Create / edit forms'),
    ('template.manage',           'template',  'Manage templates'),
    -- compliance / data
    ('audit.read',                'audit',     'View audit log entries'),
    ('audit.export',              'audit',     'Stream the audit log to CSV/NDJSON'),
    ('gdpr.export',               'gdpr',      'Export the workspace data dump'),
    -- credentials vault
    ('credential.read',           'credential','Read credentials in the vault'),
    ('credential.manage',         'credential','Create / update / delete credentials'),
    -- members & roles
    ('member.invite',             'member',    'Invite users to the workspace'),
    ('member.remove',             'member',    'Remove users from the workspace'),
    ('member.manage_roles',       'member',    'Re-role members'),
    ('role.manage',               'role',      'Create / edit / delete custom roles'),
    -- time tracking
    ('time_entry.create',         'time',      'Log your own time entries'),
    ('time_entry.update_own',     'time',      'Edit your own time entries'),
    ('time_entry.update_any',     'time',      'Edit or delete other users'' time entries'),
    -- reports
    ('report.read',               'report',    'Run accomplishment reports'),
    -- security policy
    ('mfa.enforce_workspace',     'security',  'Require MFA for everyone in the workspace')
ON CONFLICT (key) DO UPDATE
    SET description = EXCLUDED.description,
        category    = EXCLUDED.category;

-- ── seed: built-in roles ────────────────────────────────────────────────

INSERT INTO roles (workspace_id, name, description, is_builtin, rank) VALUES
    (NULL, 'owner',  'Workspace owner — every permission, including delete.', TRUE, 4),
    (NULL, 'admin',  'Workspace administrator — all permissions except delete-workspace.', TRUE, 3),
    (NULL, 'member', 'Regular member — create + edit own content, cannot manage org-wide settings.', TRUE, 2),
    (NULL, 'guest',  'External collaborator — read-only on shared content + can comment.', TRUE, 1)
ON CONFLICT (name) WHERE workspace_id IS NULL DO UPDATE
    SET description = EXCLUDED.description,
        rank        = EXCLUDED.rank,
        updated_at  = NOW();

-- ── seed: built-in role → permission grants ─────────────────────────────
-- Owner gets every permission.
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key
  FROM roles r
  JOIN permissions p ON TRUE
 WHERE r.workspace_id IS NULL AND r.name = 'owner'
ON CONFLICT DO NOTHING;

-- Admin gets every permission EXCEPT workspace.delete.
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key
  FROM roles r
  JOIN permissions p ON p.key NOT IN ('workspace.delete')
 WHERE r.workspace_id IS NULL AND r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Member gets a curated set: read everything, create/update most things,
-- but can't manage members/roles, can't export audit/gdpr, can't delete
-- the workspace, can't enforce MFA workspace-wide.
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key
  FROM roles r
  JOIN permissions p ON p.key IN (
      'workspace.read',
      'space.create','space.read','space.update',
      'folder.create','folder.update',
      'list.create','list.read','list.update',
      'task.create','task.read','task.update','task.delete',
        'task.assign','task.change_status',
      'comment.create','comment.update_own',
      'attachment.upload','attachment.delete',
      'tag.manage','status.manage','view.manage','custom_field.manage',
      'dashboard.create','dashboard.update',
      'automation.manage','goal.manage','sprint.manage',
      'doc.create','doc.update',
      'chat.send','chat.manage_channels',
      'whiteboard.manage','form.manage','template.manage',
      'credential.read','credential.manage',
      'time_entry.create','time_entry.update_own',
      'report.read'
  )
 WHERE r.workspace_id IS NULL AND r.name = 'member'
ON CONFLICT DO NOTHING;

-- Guest gets read + comment.
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key
  FROM roles r
  JOIN permissions p ON p.key IN (
      'workspace.read',
      'space.read', 'list.read',
      'task.read',
      'comment.create','comment.update_own',
      'time_entry.create','time_entry.update_own'
  )
 WHERE r.workspace_id IS NULL AND r.name = 'guest'
ON CONFLICT DO NOTHING;

-- ── back-fill existing workspace_members.role_id ────────────────────────
-- For every existing membership, point role_id at the matching built-in role
-- whose name equals the legacy `role` text column. Future code paths look at
-- role_id first; the text column stays as a denormalised cache for query
-- convenience but is no longer authoritative.
UPDATE workspace_members wm
   SET role_id = r.id
  FROM roles r
 WHERE r.workspace_id IS NULL
   AND r.name = wm.role
   AND wm.role_id IS NULL;
