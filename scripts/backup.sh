#!/bin/sh
set -eu

POSTGRES_HOST=${POSTGRES_HOST:-postgres}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
BACKUP_DIR="/backups"
BACKUP_RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-7}

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/${POSTGRES_DB}_${TIMESTAMP}.sql.gz"
SQL_FILE="${BACKUP_DIR}/${POSTGRES_DB}_${TIMESTAMP}.sql"

mkdir -p ${BACKUP_DIR}

echo "Starting PostgreSQL backup at ${TIMESTAMP}"

if pg_dump -h ${POSTGRES_HOST} -p ${POSTGRES_PORT} -U ${POSTGRES_USER} -d ${POSTGRES_DB} > "${SQL_FILE}"; then
    gzip "${SQL_FILE}"
    echo "Backup completed successfully: ${BACKUP_FILE}"
    chmod 644 ${BACKUP_FILE}
    echo "Removing backups older than ${BACKUP_RETENTION_DAYS} days"
    find ${BACKUP_DIR} -name "${POSTGRES_DB}_*.sql.gz" -type f -mtime +${BACKUP_RETENTION_DAYS} -delete
    echo "Current backups:"
    ls -la ${BACKUP_DIR}
else
    rm -f "${SQL_FILE}"
    echo "Backup failed!"
    exit 1
fi