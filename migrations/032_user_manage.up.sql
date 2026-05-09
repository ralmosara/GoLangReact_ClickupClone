-- 032_user_manage.up.sql
-- Adds the `user.manage` permission to the catalog and grants it to the
-- two built-in roles whose users should see the new User Management page:
-- owner and admin.
--
-- Members and guests stay out: workspace admins manage workspace
-- membership; only owners/admins create new accounts.

INSERT INTO permissions (key, category, description) VALUES
    ('user.manage', 'user', 'Create, edit, reset password for, and delete user accounts')
ON CONFLICT (key) DO UPDATE
    SET description = EXCLUDED.description,
        category    = EXCLUDED.category;

INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, 'user.manage'
  FROM roles r
 WHERE r.workspace_id IS NULL AND r.name IN ('owner', 'admin')
ON CONFLICT DO NOTHING;
