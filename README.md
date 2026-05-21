Микросервисная система управления проектами компании по обследованию зданий: задачи, участники, таймлайн, уведомления и аналитика

```
Пользовательский интерфейс и REST API - api-gateway;
бизнес-логика - в микросервисах по gRPC;
асинхронное взаимодесйтвие - RabbitMQ
```

## Архитектура

```
Браузер - api-gateway (HTTP :8080) - gRPC - auth-service, project-service, analytics-service, notification-service
```

| Сервис | Порт gRPC | БД (Postgres) | Назначение |
|--------|-----------|---------------|------------|
| **api-gateway** | (HTTP 8080) |  | REST и статика фронта, JWT, маршрутизация |
| **auth-service** | 50051 | auth_db (:5432) | Регистрация, логин, refresh, роли |
| **project-service** | 50052 | project_db (:5433) | Проекты, задачи, отделы, виды работ, вложения |
| **analytics-service** | 50053 | analytics_db (:5434) | Аналитика по проектам|
| **notification-service** | 50054 | notification_db (:5435) | Уведомления, дедлайны |

Инфраструктура в `docker-compose.yml`
Контракты API: `proto/` сгенерированный код в `gen/.`

---

## Сборка и запуск локально

Требования: Docker, Docker Compose.

```bash
docker compose up --build
```

### Сборка одного сервиса

```bash
cd api-gateway
go build -o api-gateway ./cmd
```

Аналогично: auth-service/cmd/auth, project-service/cmd, analytics-service/cmd, notification-service/cmd/notification

После старта:

| Что | URL |
|-----|-----|
| Приложение | http://localhost:8080 |
| RabbitMQ Management | http://localhost:15672 (guest/guest) |
| Health Check | http://localhost:8080/health |

Миграции БД выполняются контейнерами `*-migrate` при первом поднятии стека.

> **Замечание:** в docker-compose.yml у gateway в depends_on указаны auth, project и notification. Для полной аналитики должен быть запущен и analytics-service (он в compose есть отдельно; при docker compose up поднимается весь файл).

---

### Конфигурация

У каждого сервиса свой config/config.yaml (хосты БД, порты, RabbitMQ). В Docker имена хостов - имена контейнеров (auth_db, project_service, rabbitmq и т.д.).

### Protobuf (после изменения `proto/`)

Из корня репозитория (нужны protoc, protoc-gen-go, protoc-gen-go-grpc):

```bash
protoc --proto_path=proto \
  --go_out=gen --go_opt=paths=source_relative \
  --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
  proto/analytics/v1/analytics.proto
```

Для остальных .proto — та же команда с нужным файлом или все файлы в proto/

### Миграции вручную

Пример для analytics (из хоста, если БД на :5434):

```bash
migrate -path analytics-service/migrations \
  -database "postgres://analytics_user:password@localhost:5434/analytics_db?sslmode=disable" up
```

---

## API Gateway (HTTP)

Базовый URL: http://localhost:8080 
Защищённые маршруты: заголовок Authorization: Bearer <access_token> (после POST /login)

### Публичные

| Метод | Путь | Описание |
|-------|------|----------|
| POST | /login | Вход |
| POST | /register | Регистрация |
| POST | /refresh | Обновление токена |
| GET | /health | Состояние зависимых gRPC-сервисов |

### Проекты и задачи (/api/)

| Метод | Путь |
|-------|------|
| GET/POST | /api/projects, /api/projects/:id |
| PUT/DELETE/PATCH | /api/projects/:id, /api/projects/:id/status|
| GET/POST/DELETE | /api/projects/:id/members, .../members/:userId |
| GET/POST | /api/projects/:id/tasks, /api/tasks/:id |
| PUT/DELETE/PATCH |/api/tasks/:id, .../status, .../assign, .../labor |
| GET | /api/tasks/my |
| GET/PATCH | /api/projects/:id/timeline |
| GET/POST/DELETE | /api/tasks/:id/attachments, /api/attachments/:id |
| GET/POST/PUT/DELETE | /api/departments, /api/departments/:id, .../users |
| GET/POST | /api/activity-types |
| GET/PUT | /api/users/me, /api/users/:id, /api/users/find |

### Аналитика (/api/analytics/)

Параметры фильтра (общие): from_date, to_date (YYYY-MM-DD), department_id, project_id

| Метод | Путь | Назначение |
|-------|------|------------|
| GET | /api/analytics/dashboard | KPI, нагрузка отделов и тренды (один тяжёлый запрос) |
| GET | /api/analytics/workload | Нагрузка по отделам (days или период дат) |
| GET | /api/analytics/trends | Тренды задач (weeks, group_by=day\|week) |
| GET | /api/analytics/productivity | Продуктивность сотрудников |
| GET | /api/analytics/projects/timeline | Таймлайн проектов |
| GET | /api/analytics/labor | План/факт трудоёмкости (group_by=total\|department\|activity\|project) |
| GET | /api/analytics/freshness | Актуальность данных (last_event_at) |

Доступ к аналитике: все роли кроме ROLE_WORKER. Справочники отделов/видов работ — страница Инструменты (/tools, директор и ГИП).

### Уведомления

| Метод | Путь |
|-------|------|
| GET | /api/notifications, /api/notifications/unread-count |
| PATCH | /api/notifications/:id/read, /api/notifications/read-all |

### Админ (роль ROLE_ADMIN)

| Метод | Путь |
|-------|------|
| GET | /admin/users |
| PUT | /admin/users/:id/role |

---

## Страницы UI (gateway)

Статика: api-gateway/frontend/, скрипты: frontend/static/.

| URL | Страница |
|-----|----------|
| / | Вход |
| /dashboard | Главная |
| /projects, /project/:id | Проекты |
| /tasks, /task/:id` | Задачи |
| /calendar | Календарь |
| /analytics| Аналитика |
| /tools | Справочники (отделы, виды работ) |
| /notifications | Уведомления |
| /profile | Профиль |

---

## Сервисы подробнее

### auth-service

- JWT, Redis, публикация событий пользователей.
- gRPC: proto/auth/v1/auth.proto

### project-service

- Источник по проектам/задачам; публикует события в RabbitMQ (создание/обновление задач, проектов, отделов, видов работ, трудоёмкость)
- gRPC: proto/project/v1/*.proto

### analytics-service

- Не ходит в project-service за отчётами: строит по событиям из очереди (events_raw - task_analytics, projects, departments).

### notification-service

- Consumer событий и планировщик дедлайнов
- gRPC: proto/notification/v1/notification.proto

---

## Роли (кратко)

| Роль | Возможности |
|------|-------------|
| ROLE_DIRECTOR | Вся аналитика, инструменты, все проекты |
| ROLE_GIP | Аналитика, виды работ, свои проекты? некоторые доп. права |
| ROLE_DEPARTMENT_MANAGER | Аналитика в рамках отдела |
|ROLE_PROJECT_MANAGER | Аналитика, виды работ, свои проекты |
| ROLE_WORKER | Задачи; аналитика недоступна worker |

Точные проверки - в project-service/internal/authz и api-gateway (scope проектов для аналитики)
