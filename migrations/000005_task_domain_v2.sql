-- 000005_task_domain_v2.sql
-- Memperluas schema menuju domain Task Manager v1 (lihat API_CONTRACT.md).
-- Idempoten: aman dijalankan ulang. Tidak ada DROP/TRUNCATE.

-- ---------------------------------------------------------------------------
-- projects: owner eksplisit + timestamp update
-- ---------------------------------------------------------------------------
ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- Backfill owner dari project_members yang sudah ada.
UPDATE projects p
SET owner_id = pm.user_id
FROM project_members pm
WHERE pm.project_id = p.id
  AND pm.role = 'owner'
  AND p.owner_id IS NULL;

-- Set NOT NULL hanya bila semua baris sudah punya owner.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM projects WHERE owner_id IS NULL) THEN
        ALTER TABLE projects ALTER COLUMN owner_id SET NOT NULL;
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- tasks: field domain v1
-- ---------------------------------------------------------------------------
ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS project_id BIGINT REFERENCES projects(id) ON DELETE CASCADE;

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS assignee_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'medium';

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- Normalisasi nilai lama agar lolos CHECK constraint di bawah.
UPDATE tasks SET status = 'todo' WHERE status NOT IN ('todo', 'in_progress', 'done');
UPDATE tasks SET priority = 'medium' WHERE priority NOT IN ('low', 'medium', 'high', 'urgent');

-- ---------------------------------------------------------------------------
-- Constraints (dibuat hanya bila belum ada)
-- ---------------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tasks_status_check') THEN
        ALTER TABLE tasks ADD CONSTRAINT tasks_status_check
            CHECK (status IN ('todo', 'in_progress', 'done'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tasks_priority_check') THEN
        ALTER TABLE tasks ADD CONSTRAINT tasks_priority_check
            CHECK (priority IN ('low', 'medium', 'high', 'urgent'));
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'project_members_role_check') THEN
        ALTER TABLE project_members ADD CONSTRAINT project_members_role_check
            CHECK (role IN ('owner', 'admin', 'member'));
    END IF;
END $$;

-- ---------------------------------------------------------------------------
-- Indexes untuk query list/filter
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_tasks_user_id ON tasks (user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks (project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_assignee_id ON tasks (assignee_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);
CREATE INDEX IF NOT EXISTS idx_tasks_deadline ON tasks (deadline);
CREATE INDEX IF NOT EXISTS idx_project_members_user_id ON project_members (user_id);
