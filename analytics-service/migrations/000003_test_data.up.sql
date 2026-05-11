INSERT INTO events_raw (id, event_type, project_id, task_id, user_id, department_id, actor_user_id, occurred_at, payload, created_at) VALUES
(gen_random_uuid(), 'task.created', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '550e8400-e29b-41d4-a716-446655440001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', CURRENT_TIMESTAMP, '{"title":"Создание задачи"}', CURRENT_TIMESTAMP),
(gen_random_uuid(), 'task.status_changed', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '550e8400-e29b-41d4-a716-446655440001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', CURRENT_TIMESTAMP, '{"from_status":"todo","to_status":"in_progress"}', CURRENT_TIMESTAMP),
(gen_random_uuid(), 'task.completed', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '550e8400-e29b-41d4-a716-446655440001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', CURRENT_TIMESTAMP, '{"completed_at":"2026-05-10"}', CURRENT_TIMESTAMP),
(gen_random_uuid(), 'project.member_added', '11111111-1111-1111-1111-111111111111', NULL, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '550e8400-e29b-41d4-a716-446655440001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', CURRENT_TIMESTAMP, '{"role":"manager"}', CURRENT_TIMESTAMP);


INSERT INTO department_metrics (department_id, department_name, date, wip_count, completed_tasks, overdue_tasks, total_tasks, avg_cycle_time, productivity_rate, on_time_rate, throughput) VALUES

('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', CURRENT_DATE - INTERVAL '7 days', 15, 35, 5, 50, 4.2, 70.0, 85.0, 10),
('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', CURRENT_DATE - INTERVAL '6 days', 13, 35, 5, 48, 4.1, 72.9, 86.0, 9),
('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', CURRENT_DATE - INTERVAL '5 days', 17, 35, 6, 52, 4.3, 67.3, 84.0, 11),
('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', CURRENT_DATE - INTERVAL '4 days', 14, 35, 5, 49, 4.0, 71.4, 85.5, 10),
('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', CURRENT_DATE - INTERVAL '3 days', 16, 35, 5, 51, 4.2, 68.6, 85.2, 10),
('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', CURRENT_DATE - INTERVAL '2 days', 18, 35, 6, 53, 4.4, 66.0, 83.5, 12),
('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', CURRENT_DATE - INTERVAL '1 day', 15, 35, 5, 50, 4.2, 70.0, 85.0, 10),
('550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', CURRENT_DATE, 15, 35, 5, 50, 4.2, 70.0, 85.0, 10),


('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', CURRENT_DATE - INTERVAL '7 days', 8, 22, 2, 30, 3.5, 73.3, 91.7, 6),
('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', CURRENT_DATE - INTERVAL '6 days', 6, 22, 2, 28, 3.4, 78.6, 92.3, 5),
('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', CURRENT_DATE - INTERVAL '5 days', 10, 22, 3, 32, 3.6, 68.8, 91.0, 7),
('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', CURRENT_DATE - INTERVAL '4 days', 7, 22, 2, 29, 3.5, 75.9, 91.4, 6),
('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', CURRENT_DATE - INTERVAL '3 days', 9, 22, 2, 31, 3.5, 71.0, 91.7, 6),
('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', CURRENT_DATE - INTERVAL '2 days', 11, 22, 3, 33, 3.7, 66.7, 90.9, 7),
('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', CURRENT_DATE - INTERVAL '1 day', 8, 22, 2, 30, 3.5, 73.3, 91.7, 6),
('550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', CURRENT_DATE, 8, 22, 2, 30, 3.5, 73.3, 91.7, 6),


('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', CURRENT_DATE - INTERVAL '7 days', 10, 28, 3, 38, 3.8, 73.7, 86.8, 7),
('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', CURRENT_DATE - INTERVAL '6 days', 9, 28, 3, 37, 3.7, 75.7, 87.1, 7),
('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', CURRENT_DATE - INTERVAL '5 days', 12, 28, 4, 40, 3.9, 70.0, 85.0, 8),
('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', CURRENT_DATE - INTERVAL '4 days', 8, 28, 2, 36, 3.6, 77.8, 87.9, 6),
('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', CURRENT_DATE - INTERVAL '3 days', 10, 28, 3, 38, 3.8, 73.7, 86.8, 7),
('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', CURRENT_DATE - INTERVAL '2 days', 13, 28, 5, 41, 4.0, 68.3, 84.1, 8),
('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', CURRENT_DATE - INTERVAL '1 day', 10, 28, 3, 38, 3.8, 73.7, 86.8, 7),
('550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', CURRENT_DATE, 10, 28, 3, 38, 3.8, 73.7, 86.8, 7);

INSERT INTO task_analytics (task_id, project_id, department_id, assigned_user_id, created_at, assigned_at, completed_at, due_date, status, is_overdue, cycle_time_days, delayed_days, created_by) VALUES
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', CURRENT_DATE - INTERVAL '10 days', CURRENT_DATE - INTERVAL '9 days', CURRENT_DATE - INTERVAL '2 days', CURRENT_DATE - INTERVAL '1 day', 3, FALSE, 8, 1, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440001', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', CURRENT_DATE - INTERVAL '8 days', CURRENT_DATE - INTERVAL '7 days', NULL, CURRENT_DATE + INTERVAL '5 days', 1, FALSE, 0, 0, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa'),
('cccccccc-cccc-cccc-cccc-cccccccccccc', '22222222-2222-2222-2222-222222222222', '550e8400-e29b-41d4-a716-446655440002', 'cccccccc-cccc-cccc-cccc-cccccccccccc', CURRENT_DATE - INTERVAL '15 days', CURRENT_DATE - INTERVAL '14 days', CURRENT_DATE - INTERVAL '5 days', CURRENT_DATE - INTERVAL '7 days', 3, TRUE, 10, 2, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
('dddddddd-dddd-dddd-dddd-dddddddddddd', '22222222-2222-2222-2222-222222222222', '550e8400-e29b-41d4-a716-446655440002', 'cccccccc-cccc-cccc-cccc-cccccccccccc', CURRENT_DATE - INTERVAL '5 days', CURRENT_DATE - INTERVAL '4 days', NULL, CURRENT_DATE + INTERVAL '10 days', 2, FALSE, 0, 0, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'),
('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '33333333-3333-3333-3333-333333333333', '550e8400-e29b-41d4-a716-446655440003', 'dddddddd-dddd-dddd-dddd-dddddddddddd', CURRENT_DATE - INTERVAL '12 days', CURRENT_DATE - INTERVAL '11 days', CURRENT_DATE - INTERVAL '3 days', CURRENT_DATE - INTERVAL '4 days', 3, FALSE, 9, 1, 'cccccccc-cccc-cccc-cccc-cccccccccccc'),
('ffffffff-ffff-ffff-ffff-ffffffffffff', '33333333-3333-3333-3333-333333333333', '550e8400-e29b-41d4-a716-446655440003', 'dddddddd-dddd-dddd-dddd-dddddddddddd', CURRENT_DATE - INTERVAL '3 days', CURRENT_DATE - INTERVAL '2 days', NULL, CURRENT_DATE + INTERVAL '7 days', 1, FALSE, 0, 0, 'cccccccc-cccc-cccc-cccc-cccccccccccc');


INSERT INTO project_timeline_control (project_id, project_name, department_id, total_tasks, completed_on_time, overdue_tasks, on_time_rate, avg_delay_days, start_date, end_date) VALUES
('11111111-1111-1111-1111-111111111111', 'Реконструкция ТЦ Галерея', '550e8400-e29b-41d4-a716-446655440001', 25, 18, 5, 72.0, 3.5, CURRENT_DATE - INTERVAL '30 days', CURRENT_DATE + INTERVAL '30 days'),
('22222222-2222-2222-2222-222222222222', 'Обследование Жилого Комплекса Солнечный', '550e8400-e29b-41d4-a716-446655440002', 30, 22, 6, 73.3, 4.2, CURRENT_DATE - INTERVAL '20 days', CURRENT_DATE + INTERVAL '40 days'),
('33333333-3333-3333-3333-333333333333', 'Инструментальное Обследование Завода Металлист', '550e8400-e29b-41d4-a716-446655440003', 20, 15, 3, 75.0, 2.5, CURRENT_DATE - INTERVAL '15 days', CURRENT_DATE + INTERVAL '15 days'),
('44444444-4444-4444-4444-444444444444', 'Диагностика Фундамента БЦ Вертикаль', '550e8400-e29b-41d4-a716-446655440001', 15, 10, 4, 66.7, 5.0, CURRENT_DATE - INTERVAL '10 days', CURRENT_DATE + INTERVAL '20 days');


INSERT INTO department_delays (project_id, department_id, department_name, overdue_tasks, avg_delay_days, date) VALUES
('11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', 3, 4.2, CURRENT_DATE - INTERVAL '7 days'),
('11111111-1111-1111-1111-111111111111', '550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', 2, 3.5, CURRENT_DATE - INTERVAL '7 days'),
('22222222-2222-2222-2222-222222222222', '550e8400-e29b-41d4-a716-446655440002', 'Камеральная группа', 4, 5.0, CURRENT_DATE - INTERVAL '5 days'),
('33333333-3333-3333-3333-333333333333', '550e8400-e29b-41d4-a716-446655440003', 'Графический отдел', 2, 2.8, CURRENT_DATE - INTERVAL '3 days'),
('44444444-4444-4444-4444-444444444444', '550e8400-e29b-41d4-a716-446655440001', 'Полевая группа', 5, 6.0, CURRENT_DATE - INTERVAL '2 days');

INSERT INTO weekly_trends (week, department_id, tasks_created, tasks_completed, tasks_overdue, completion_rate, on_time_rate) VALUES

(CURRENT_DATE - INTERVAL '8 weeks', '550e8400-e29b-41d4-a716-446655440001', 25, 20, 3, 80.0, 85.0),
(CURRENT_DATE - INTERVAL '7 weeks', '550e8400-e29b-41d4-a716-446655440001', 28, 22, 4, 78.6, 82.1),
(CURRENT_DATE - INTERVAL '6 weeks', '550e8400-e29b-41d4-a716-446655440001', 30, 24, 4, 80.0, 83.3),
(CURRENT_DATE - INTERVAL '5 weeks', '550e8400-e29b-41d4-a716-446655440001', 27, 21, 3, 77.8, 85.7),
(CURRENT_DATE - INTERVAL '4 weeks', '550e8400-e29b-41d4-a716-446655440001', 32, 25, 5, 78.1, 80.0),
(CURRENT_DATE - INTERVAL '3 weeks', '550e8400-e29b-41d4-a716-446655440001', 29, 23, 3, 79.3, 87.0),
(CURRENT_DATE - INTERVAL '2 weeks', '550e8400-e29b-41d4-a716-446655440001', 31, 24, 4, 77.4, 83.3),
(CURRENT_DATE - INTERVAL '1 week', '550e8400-e29b-41d4-a716-446655440001', 26, 20, 3, 76.9, 85.0),


(CURRENT_DATE - INTERVAL '8 weeks', '550e8400-e29b-41d4-a716-446655440002', 15, 12, 1, 80.0, 93.3),
(CURRENT_DATE - INTERVAL '7 weeks', '550e8400-e29b-41d4-a716-446655440002', 18, 14, 2, 77.8, 85.7),
(CURRENT_DATE - INTERVAL '6 weeks', '550e8400-e29b-41d4-a716-446655440002', 16, 13, 1, 81.3, 92.3),
(CURRENT_DATE - INTERVAL '5 weeks', '550e8400-e29b-41d4-a716-446655440002', 17, 13, 2, 76.5, 84.6),
(CURRENT_DATE - INTERVAL '4 weeks', '550e8400-e29b-41d4-a716-446655440002', 19, 15, 2, 78.9, 86.7),
(CURRENT_DATE - INTERVAL '3 weeks', '550e8400-e29b-41d4-a716-446655440002', 14, 11, 1, 78.6, 92.9),
(CURRENT_DATE - INTERVAL '2 weeks', '550e8400-e29b-41d4-a716-446655440002', 16, 12, 2, 75.0, 83.3),
(CURRENT_DATE - INTERVAL '1 week', '550e8400-e29b-41d4-a716-446655440002', 15, 12, 1, 80.0, 91.7),

(CURRENT_DATE - INTERVAL '8 weeks', '550e8400-e29b-41d4-a716-446655440003', 20, 16, 2, 80.0, 88.9),
(CURRENT_DATE - INTERVAL '7 weeks', '550e8400-e29b-41d4-a716-446655440003', 22, 17, 3, 77.3, 85.0),
(CURRENT_DATE - INTERVAL '6 weeks', '550e8400-e29b-41d4-a716-446655440003', 24, 19, 3, 79.2, 86.4),
(CURRENT_DATE - INTERVAL '5 weeks', '550e8400-e29b-41d4-a716-446655440003', 21, 16, 2, 76.2, 88.9),
(CURRENT_DATE - INTERVAL '4 weeks', '550e8400-e29b-41d4-a716-446655440003', 23, 18, 3, 78.3, 85.7),
(CURRENT_DATE - INTERVAL '3 weeks', '550e8400-e29b-41d4-a716-446655440003', 20, 16, 2, 80.0, 88.9),
(CURRENT_DATE - INTERVAL '2 weeks', '550e8400-e29b-41d4-a716-446655440003', 25, 19, 4, 76.0, 82.6),
(CURRENT_DATE - INTERVAL '1 week', '550e8400-e29b-41d4-a716-446655440003', 22, 17, 3, 77.3, 85.0);



INSERT INTO employee_productivity (user_id, full_name, email, department_id, date, tasks_completed, tasks_overdue, avg_cycle_time, completion_rate, on_time_rate) VALUES

('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '550e8400-e29b-41d4-a716-446655440001', CURRENT_DATE - INTERVAL '7 days', 8, 2, 5.5, 80.0, 75.0),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '550e8400-e29b-41d4-a716-446655440001', CURRENT_DATE - INTERVAL '6 days', 7, 1, 5.2, 87.5, 85.7),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '550e8400-e29b-41d4-a716-446655440001', CURRENT_DATE - INTERVAL '5 days', 9, 2, 5.8, 81.8, 77.8),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '550e8400-e29b-41d4-a716-446655440001', CURRENT_DATE - INTERVAL '4 days', 6, 1, 5.0, 85.7, 83.3),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '550e8400-e29b-41d4-a716-446655440001', CURRENT_DATE - INTERVAL '3 days', 8, 1, 5.3, 88.9, 87.5),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '550e8400-e29b-41d4-a716-446655440001', CURRENT_DATE - INTERVAL '2 days', 7, 2, 5.6, 77.8, 71.4),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '550e8400-e29b-41d4-a716-446655440001', CURRENT_DATE - INTERVAL '1 day', 8, 1, 5.4, 88.9, 87.5),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '550e8400-e29b-41d4-a716-446655440001', CURRENT_DATE, 8, 1, 5.5, 88.9, 87.5),

('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '550e8400-e29b-41d4-a716-446655440002', CURRENT_DATE - INTERVAL '7 days', 5, 0, 3.2, 100.0, 100.0),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '550e8400-e29b-41d4-a716-446655440002', CURRENT_DATE - INTERVAL '6 days', 4, 0, 3.0, 100.0, 100.0),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '550e8400-e29b-41d4-a716-446655440002', CURRENT_DATE - INTERVAL '5 days', 6, 1, 3.4, 85.7, 83.3),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '550e8400-e29b-41d4-a716-446655440002', CURRENT_DATE - INTERVAL '4 days', 5, 0, 3.1, 100.0, 100.0),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '550e8400-e29b-41d4-a716-446655440002', CURRENT_DATE - INTERVAL '3 days', 4, 0, 2.9, 100.0, 100.0),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '550e8400-e29b-41d4-a716-446655440002', CURRENT_DATE - INTERVAL '2 days', 5, 1, 3.3, 83.3, 80.0),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '550e8400-e29b-41d4-a716-446655440002', CURRENT_DATE - INTERVAL '1 day', 5, 0, 3.2, 100.0, 100.0),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '550e8400-e29b-41d4-a716-446655440002', CURRENT_DATE, 5, 0, 3.1, 100.0, 100.0),

('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '550e8400-e29b-41d4-a716-446655440003', CURRENT_DATE - INTERVAL '7 days', 7, 1, 4.1, 87.5, 85.7),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '550e8400-e29b-41d4-a716-446655440003', CURRENT_DATE - INTERVAL '6 days', 6, 0, 3.9, 100.0, 100.0),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '550e8400-e29b-41d4-a716-446655440003', CURRENT_DATE - INTERVAL '5 days', 8, 1, 4.3, 88.9, 87.5),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '550e8400-e29b-41d4-a716-446655440003', CURRENT_DATE - INTERVAL '4 days', 7, 1, 4.2, 87.5, 85.7),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '550e8400-e29b-41d4-a716-446655440003', CURRENT_DATE - INTERVAL '3 days', 6, 0, 4.0, 100.0, 100.0),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '550e8400-e29b-41d4-a716-446655440003', CURRENT_DATE - INTERVAL '2 days', 7, 1, 4.2, 87.5, 85.7),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '550e8400-e29b-41d4-a716-446655440003', CURRENT_DATE - INTERVAL '1 day', 8, 1, 4.4, 88.9, 87.5),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '550e8400-e29b-41d4-a716-446655440003', CURRENT_DATE, 7, 1, 4.2, 87.5, 85.7);