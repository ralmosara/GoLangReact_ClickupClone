-- Reverses 032_user_manage.up.sql. Removes the user.manage grant from
-- every role and drops the catalog row. Custom roles that picked up the
-- permission lose it too — the assumption is `down` is run for an
-- intentional rollback, not in steady state.
DELETE FROM role_permissions WHERE permission_key = 'user.manage';
DELETE FROM permissions      WHERE key            = 'user.manage';
