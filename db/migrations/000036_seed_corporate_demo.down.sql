-- Rollback Migration 000036
DELETE FROM users WHERE email IN ('admin@corporate.com', 'csr@corporate.com', 'reviewer@corporate.com');
DELETE FROM organizations WHERE id = '99999999-9999-4000-a000-000000000001';
