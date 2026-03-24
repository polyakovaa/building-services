CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS project_timeline (
    project_id UUID PRIMARY KEY REFERENCES projects(id) ON DELETE CASCADE,
    contract_date TIMESTAMP,
    work_start_date TIMESTAMP,
    work_end_date TIMESTAMP,
    handover_date TIMESTAMP,
    comments_date TIMESTAMP,
    comments_fixed_date TIMESTAMP,
    acceptance_date TIMESTAMP,
    final_payment_date TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID
);

CREATE INDEX idx_timeline_dates ON project_timeline(work_start_date, work_end_date);