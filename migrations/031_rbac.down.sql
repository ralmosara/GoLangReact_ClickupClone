-- Reverses 031_rbac.up.sql.
ALTER TABLE workspace_members DROP COLUMN IF EXISTS role_id;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS permissions;
