-- Migration 000024 Down: Remove expanded 78 prospective organizations

DELETE FROM organizations WHERE id::text LIKE 'b1000000-0000-4000-a000-0000000000%';
