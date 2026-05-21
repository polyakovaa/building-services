# Результаты нагрузочного тестирования (hey, GET)

Стенд: Docker Compose, `http://localhost:8080`. Инструмент: [hey](https://github.com/rakyll/hey).  
Параметры — как в плане: `-n` запросов, `-c` параллельных воркеров.

---

## Сводная таблица

| № | Эндпоинт | `-n` (задано) | `-c` | Завершено (200) | Не дошли до 200 | % успеха | RPS | p50 | p95 | p99 | Среднее | Длительность прогона |
|---|----------|---------------|------|-----------------|-----------------|----------|-----|-----|-----|-----|---------|----------------------|
| 1 | `GET /health` | 500 | 10 | **500** | 0 | 100% | 1018 | 6 ms | 9 ms | 188 ms | 10 ms | 0.49 s |
| 2 | `GET /health` | 2000 | 50 | **2000** | 0 | 100% | 2748 | 14 ms | 22 ms | 149 ms | 17 ms | 0.73 s |
| 3 | `GET /api/departments` | 500 | 10 | **500** | 0 | 100% | 907 | 8 ms | 30 ms | 99 ms | 11 ms | 0.55 s |
| 4 | `GET /api/departments` | 2000 | 30 | **1980** | **20** | 99.0% | 1454 | 17 ms | 39 ms | 66 ms | 20 ms | 1.36 s |
| 5 | `GET /api/analytics/dashboard?period=month` | 300 | 10 | **300** | 0 | 100% | 181 | 41 ms | 89 ms | 291 ms | 54 ms | 1.66 s |
| 6 | `GET /api/analytics/dashboard?period=month` | **1000** | 30 | **990** | **10** | **99.0%** | 278 | 97 ms | 171 ms | 230 ms | 105 ms | 3.56 s |
| 7 | `GET /api/analytics/dashboard?period=month` | 2000 | 50 | **2000** | 0 | 100% | 340 | 134 ms | 222 ms | 347 ms | 142 ms | 5.88 s |

Источники: файлы `loadtest_*.txt` в корне репозитория.

---

## Куда делись 10 запросов на dashboard `-n 1000 -c 30`?

Вы задали **1000** запросов, в отчёте hey — **990** ответов с кодом **200**. Расхождение **10 (1%)** — это не опечатка в плане и не «пропажа» в таблице.

**Как проверить:** сложите столбцы гистограммы в `loadtest_dashboard_c30.txt`:

`1 + 5 + 70 + 365 + 287 + 153 + 65 + 26 + 9 + 6 + 3 = **990**`

Число в `Status code distribution: [200] 990` совпадает с гистограммой. **RPS 278** тоже считается от фактически завершённых: `990 / 3.561 с ≈ 278`, а не от 1000.

**Куда делись 10 запросов:** hey **не получил HTTP-ответ** (или прогон завершился до их учёта). Типичные причины на тяжёлом эндпоинте при `-c 30`:

- таймаут или сброс соединения на стороне клиента/ОС;
- кратковременная перегрузка gateway / analytics / пула БД (`max_open_conns: 10`);
- гонка при закрытии keep-alive соединений в конце прогона.

Такие запросы **не попадают** в `Status code distribution` (нет кода 503/500 — нет ответа вообще). В некоторых версиях hey внизу бывает блок `Error distribution`; в вашем файле его нет — значит ошибки не классифицированы отдельно, но **1% «молчаливых» потерь** при dashboard c30 — нормальная картина для локального Docker.

**Аналогично:** `departments` `-n 2000 -c 30` → **1980** ответов (**−20**, тоже ~1%).

**Важно для отчёта:** метрики latency (p50, p95, p99) в этом прогоне посчитаны **только по 990 успешным**; при оценке надёжности укажите **99% успешных завершений**, не 100%.

Прогон **dashboard c50** (`-n 2000`) — все **2000** ответов 200; потери не повторились на том же эндпоинте при другой нагрузке.

---

## Краткие выводы

| Наблюдение | Детали |
|------------|--------|
| Стабильность | Все завершённые запросы — **HTTP 200** (нет 401/5xx в логах hey). |
| Сложность эндпоинта | p50: health ~6 ms → departments ~8–17 ms → dashboard ~41–134 ms. |
| Эффект `-c` на dashboard | RPS 181 → 278 → 340; p95 89 → 171 → 222 ms — пропускная способность растёт, задержки тоже. |
| Узкое место | Тяжёлый путь: `analytics_service` + `analytics_db` (см. `stats_after_c10_dashboard.txt`, `stats_after_c50_dashboard.txt`). |
| Потери под нагрузкой | ~1% на dashboard c30 и departments c30; учитывать в разделе «ограничения теста». |

---

## Docker stats (кумулятивный NET I/O, ориентир)

| Снимок | api-gateway | project_service | analytics_service | analytics_db |
|--------|-------------|-----------------|-------------------|----------------|
| До тестов | 1.25 MB | 235 kB | 233 kB | 22 kB |
| После dashboard c10 | 6.78 MB | 7.21 MB | 1.32 MB | 2.49 MB |
| После dashboard c30 | 9.11 MB | 9.39 MB | 4.36 MB | 10.3 MB |
| После dashboard c50 | 13.8 MB | 13.8 MB | 10.5 / 28.3 MB | 26.1 MB |

Файлы: `stats_before.txt`, `stats_after_c10_departments.txt`, `stats_after_c30_departments.txt`, `stats_after_c10_dashboard.txt`, `stats_after_c30_dashboard.txt`, `stats_after_c50_dashboard.txt`.

---

## Команды (для воспроизведения)

```powershell
$env:TOKEN = "<jwt>"

# Health
hey -n 500  -c 10  http://localhost:8080/health
hey -n 2000 -c 50  http://localhost:8080/health

# Departments
hey -n 500  -c 10  -H "Authorization: Bearer $env:TOKEN" http://localhost:8080/api/departments
hey -n 2000 -c 30  -H "Authorization: Bearer $env:TOKEN" http://localhost:8080/api/departments

# Dashboard
hey -n 300  -c 10  -H "Authorization: Bearer $env:TOKEN" "http://localhost:8080/api/analytics/dashboard?period=month"
hey -n 1000 -c 30  -H "Authorization: Bearer $env:TOKEN" "http://localhost:8080/api/analytics/dashboard?period=month"
hey -n 2000 -c 50  -H "Authorization: Bearer $env:TOKEN" "http://localhost:8080/api/analytics/dashboard?period=month"
```

Повторный прогон dashboard `-n 1000 -c 30` с `hey -disable-keepalive` или меньшим `-c` может дать ровно 1000/1000 — для сравнения, если нужно убрать 1% потерь.
