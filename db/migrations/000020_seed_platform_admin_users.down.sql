-- Migration 000020 Down: Remove Seed Platform Admin Users

DELETE FROM users WHERE email IN ('admin@sovera.id', 'ops@sovera.id', 'support@sovera.id');
DELETE FROM organizations WHERE id = '00000000-0000-0000-0000-000000000000';
