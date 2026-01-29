-- Note: This rollback will lose group_links data that doesn't correspond to existing org links
-- The original org-scoped links in the links table remain untouched

DROP TABLE IF EXISTS group_links;
DROP TABLE IF EXISTS user_group_memberships;
DROP TABLE IF EXISTS groups;
