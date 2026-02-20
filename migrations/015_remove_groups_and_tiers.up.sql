-- Remove tier/group system tables. Org-scoped links remain in the links table.
DROP TABLE IF EXISTS group_links;
DROP TABLE IF EXISTS user_group_memberships;
DROP TABLE IF EXISTS groups;
