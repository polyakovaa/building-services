#!/bin/bash
# Пример ежедневного бэкапа Postgres (запуск на хосте VPS, не в контейнере).
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/building-services}"
mkdir -p "$BACKUP_DIR"
STAMP=$(date +%Y%m%d_%H%M%S)

docker exec auth_db pg_dump -U "${AUTH_DB_USER:-auth_user}" "${AUTH_DB_NAME:-auth_db}" \
  | gzip > "$BACKUP_DIR/auth_${STAMP}.sql.gz"

docker exec project_db pg_dump -U "${PROJECT_DB_USER:-project_user}" "${PROJECT_DB_NAME:-project_db}" \
  | gzip > "$BACKUP_DIR/project_${STAMP}.sql.gz"

docker exec analytics_db pg_dump -U "${ANALYTICS_DB_USER:-analytics_user}" "${ANALYTICS_DB_NAME:-analytics_db}" \
  | gzip > "$BACKUP_DIR/analytics_${STAMP}.sql.gz"

docker exec notification_db pg_dump -U "${NOTIFICATION_DB_USER:-notification_user}" "${NOTIFICATION_DB_NAME:-notification_db}" \
  | gzip > "$BACKUP_DIR/notification_${STAMP}.sql.gz"

find "$BACKUP_DIR" -name '*.sql.gz' -mtime +14 -delete
echo "Backup done: $BACKUP_DIR/*_${STAMP}.sql.gz"
