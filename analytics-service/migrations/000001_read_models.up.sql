CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS events_raw (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    project_id UUID,
    task_id UUID,
    user_id UUID,
    department_id UUID,
    actor_user_id UUID,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255),
    full_name VARCHAR(255),
    role VARCHAR(100),
    department_id UUID,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS projects (
    project_id UUID PRIMARY KEY,
    project_name VARCHAR(255),
    start_date TIMESTAMP WITH TIME ZONE,
    end_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS task_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL,
    project_id UUID NOT NULL,
    department_id UUID,
    assigned_user_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    due_date TIMESTAMP WITH TIME ZONE,
    status INTEGER,
    is_overdue BOOLEAN DEFAULT FALSE,
    cycle_time_days INTEGER,
    delayed_days INTEGER,
    created_by UUID,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(task_id)
);

CREATE INDEX IF NOT EXISTS idx_task_analytics_project ON task_analytics(project_id);
CREATE INDEX IF NOT EXISTS idx_task_analytics_assignee ON task_analytics(assigned_user_id);
CREATE INDEX IF NOT EXISTS idx_task_analytics_department ON task_analytics(department_id);
CREATE INDEX IF NOT EXISTS idx_users_department ON users(department_id);

INSERT INTO departments (id, name) VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа'),
    ('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа'),
    ('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO users (id, email, full_name, role, department_id, updated_at) VALUES
    ('ffffffff-ffff-ffff-ffff-ffffffffffff', 'john5@example.com', 'Василий Васин', 'ROLE_GIP', '550e8400-e29b-41d4-a716-446655440001', CURRENT_TIMESTAMP),
    ('99999999-9999-9999-9999-999999999999', 'antonina@field.ru', 'Антонина Ивановна', 'ROLE_DEPARTMENT_MANAGER', '550e8400-e29b-41d4-a716-446655440001', CURRENT_TIMESTAMP),
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'ivan.ivanov@field.ru', 'Иванов Иван Иванович', 'ROLE_WORKER', '550e8400-e29b-41d4-a716-446655440001', CURRENT_TIMESTAMP),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'petr.petrov@field.ru', 'Петров Пётр Петрович', 'ROLE_WORKER', '550e8400-e29b-41d4-a716-446655440001', CURRENT_TIMESTAMP),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'maria.sidorova@cam.ru', 'Сидорова Мария Ивановна', 'ROLE_WORKER', '550e8400-e29b-41d4-a716-446655440002', CURRENT_TIMESTAMP),
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', 'ekaterina.kuznetsova@graph.ru', 'Кузнецова Екатерина Дмитриевна', 'ROLE_WORKER', '550e8400-e29b-41d4-a716-446655440003', CURRENT_TIMESTAMP)
ON CONFLICT (id) DO UPDATE SET
    email = EXCLUDED.email,
    full_name = EXCLUDED.full_name,
    role = EXCLUDED.role,
    department_id = EXCLUDED.department_id,
    updated_at = CURRENT_TIMESTAMP;

INSERT INTO projects (project_id, project_name, start_date, end_date) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Реконструкция ТЦ Галерея', CURRENT_DATE - 30, CURRENT_DATE + 30),
    ('22222222-2222-2222-2222-222222222222', 'Обследование Жилого Комплекса Солнечный', CURRENT_DATE - 20, CURRENT_DATE + 40),
    ('33333333-3333-3333-3333-333333333333', 'Инструментальное Обследование Завода Металлист', CURRENT_DATE - 15, CURRENT_DATE + 15),
    ('44444444-4444-4444-4444-444444444444', 'Диагностика Фундамента БЦ Вертикаль', CURRENT_DATE - 10, CURRENT_DATE + 20)
ON CONFLICT (project_id) DO UPDATE SET
    project_name = EXCLUDED.project_name,
    start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date;

INSERT INTO task_analytics (task_id, project_id, department_id, assigned_user_id, created_at, assigned_at, completed_at, due_date, status, is_overdue, cycle_time_days, delayed_days, created_by) VALUES
    ('10000001-0001-4000-8000-000000000001', '11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440001', 'ffffffff-ffff-ffff-ffff-ffffffffffff', '2026-05-01', '2026-05-01', NULL, '2026-05-08', 1, FALSE, 0, 0, 'ffffffff-ffff-ffff-ffff-ffffffffffff'),
    ('10000001-0001-4000-8000-000000000002', '11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2026-05-02', '2026-05-02', NULL, '2026-05-17', 2, FALSE, 0, 0, 'ffffffff-ffff-ffff-ffff-ffffffffffff'),
    ('10000001-0001-4000-8000-000000000003', '11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '2026-05-03', '2026-05-03', NULL, '2026-05-22', 1, FALSE, 0, 0, 'ffffffff-ffff-ffff-ffff-ffffffffffff'),
    ('10000001-0001-4000-8000-000000000004', '11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440001', 'ffffffff-ffff-ffff-ffff-ffffffffffff', '2026-05-04', '2026-05-04', NULL, '2026-05-20', 1, FALSE, 0, 0, 'ffffffff-ffff-ffff-ffff-ffffffffffff'),
    ('10000001-0001-4000-8000-000000000005', '11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440001', '99999999-9999-9999-9999-999999999999', '2026-05-05', '2026-05-05', '2026-05-13', '2026-05-12', 3, TRUE, 8, 1, 'ffffffff-ffff-ffff-ffff-ffffffffffff'),
    ('10000001-0001-4000-8000-000000000006', '22222222-2222-2222-2222-222222222222', '550e8400-e29b-41d4-a716-446655440002', 'cccccccc-cccc-cccc-cccc-cccccccccccc', '2026-05-06', '2026-05-06', NULL, '2026-05-27', 1, FALSE, 0, 0, 'ffffffff-ffff-ffff-ffff-ffffffffffff'),
    ('10000001-0001-4000-8000-000000000007', '22222222-2222-2222-2222-222222222222', '550e8400-e29b-41d4-a716-446655440001', '99999999-9999-9999-9999-999999999999', '2026-05-07', '2026-05-07', NULL, '2026-05-20', 1, FALSE, 0, 0, 'ffffffff-ffff-ffff-ffff-ffffffffffff'),
    ('10000001-0001-4000-8000-000000000008', '33333333-3333-3333-3333-333333333333', '550e8400-e29b-41d4-a716-446655440003', 'dddddddd-dddd-dddd-dddd-dddddddddddd', '2026-05-08', '2026-05-08', NULL, '2026-05-19', 2, FALSE, 0, 0, 'ffffffff-ffff-ffff-ffff-ffffffffffff')
ON CONFLICT (task_id) DO NOTHING;
