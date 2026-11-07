-- ZenForge v0.3.2 migration: Add intra-task micro-phases support
ALTER TABLE tasks ADD COLUMN phases TEXT NOT NULL DEFAULT '[]';
ALTER TABLE tasks ADD COLUMN current_phase TEXT NOT NULL DEFAULT '';
