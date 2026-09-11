-- Migration 000022 Down: Rollback seeded humanitarian organizations

DELETE FROM organizations WHERE id LIKE 'b1000000-0000-4000-a000-%';
