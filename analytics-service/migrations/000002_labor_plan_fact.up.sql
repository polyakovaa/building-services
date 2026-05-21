CREATE TABLE IF NOT EXISTS activity_types (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    sort_order INT NOT NULL DEFAULT 0
);

INSERT INTO activity_types (id, name, sort_order) VALUES
    ('a1000001-0001-4000-8000-000000000001', 'Дорога на объект', 1),
    ('a1000001-0001-4000-8000-000000000002', 'Общее планирование работ', 2),
    ('a1000001-0001-4000-8000-000000000003', 'Сбор и анализ исходной документации', 3),
    ('a1000001-0001-4000-8000-000000000004', 'Панорамная видеосъемка', 4),
    ('a1000001-0001-4000-8000-000000000005', 'Проверка облака точек', 5),
    ('a1000001-0001-4000-8000-000000000006', 'Составление полевой ведомости дефектов', 6),
    ('a1000001-0001-4000-8000-000000000007', 'Визуальное обследование конструкций', 7),
    ('a1000001-0001-4000-8000-000000000008', 'Камеральная обработка результатов', 8),
    ('a1000001-0001-4000-8000-000000000009', 'Подготовка отчета', 9),
    ('a1000001-0001-4000-8000-000000000010', 'Согласование с заказчиком', 10)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    sort_order = EXCLUDED.sort_order;

ALTER TABLE task_analytics
    ADD COLUMN IF NOT EXISTS activity_type_id UUID,
    ADD COLUMN IF NOT EXISTS planned_hours NUMERIC(10, 2),
    ADD COLUMN IF NOT EXISTS actual_hours NUMERIC(10, 2);

CREATE INDEX IF NOT EXISTS idx_task_analytics_activity ON task_analytics(activity_type_id);
