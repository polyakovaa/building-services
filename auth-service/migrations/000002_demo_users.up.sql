DELETE FROM users WHERE email IN (
    'john5@example.com',
    'antonina@field.ru',
    'ivan.ivanov@field.ru',
    'petr.petrov@field.ru',
    'maria.sidorova@cam.ru',
    'ekaterina.kuznetsova@graph.ru'
);

INSERT INTO users (id, full_name, email, password_hash, role) VALUES
('ffffffff-ffff-ffff-ffff-ffffffffffff', 'Василий Васин', 'john5@example.com', '$2a$10$EqPcxYWHeAB/RB47J3r8/OYlzFRmr95S.VHUc4Iopyu8W8JT7LOca', 'ROLE_GIP'),
('99999999-9999-9999-9999-999999999999', 'Антонина Ивановна', 'antonina@field.ru', '$2a$10$EqPcxYWHeAB/RB47J3r8/OYlzFRmr95S.VHUc4Iopyu8W8JT7LOca', 'ROLE_DEPARTMENT_MANAGER'),
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Иванов Иван Иванович', 'ivan.ivanov@field.ru', '$2a$10$EqPcxYWHeAB/RB47J3r8/OYlzFRmr95S.VHUc4Iopyu8W8JT7LOca', 'ROLE_WORKER'),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Петров Пётр Петрович', 'petr.petrov@field.ru', '$2a$10$EqPcxYWHeAB/RB47J3r8/OYlzFRmr95S.VHUc4Iopyu8W8JT7LOca', 'ROLE_WORKER'),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Сидорова Мария Ивановна', 'maria.sidorova@cam.ru', '$2a$10$EqPcxYWHeAB/RB47J3r8/OYlzFRmr95S.VHUc4Iopyu8W8JT7LOca', 'ROLE_WORKER'),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Кузнецова Екатерина Дмитриевна', 'ekaterina.kuznetsova@graph.ru', '$2a$10$EqPcxYWHeAB/RB47J3r8/OYlzFRmr95S.VHUc4Iopyu8W8JT7LOca', 'ROLE_WORKER');
