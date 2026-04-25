CREATE TABLE forms (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id       UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    fields        JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_public     BOOLEAN NOT NULL DEFAULT TRUE,
    submit_count  INT NOT NULL DEFAULT 0,
    creator_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_forms_list ON forms(list_id);

-- Submissions are retained alongside the created task so we can re-materialise
-- raw inputs even after field config changes.
CREATE TABLE form_submissions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    form_id      UUID NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    task_id      UUID REFERENCES tasks(id) ON DELETE SET NULL,
    payload      JSONB NOT NULL DEFAULT '{}'::jsonb,
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitter_ip TEXT
);
CREATE INDEX idx_form_submissions_form ON form_submissions(form_id);
