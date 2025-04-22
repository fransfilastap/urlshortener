#!/bin/bash
set -e

# Configuration
POSTGRES_HOST=${POSTGRES_HOST:-postgres}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
BACKUP_DIR="/backups"
BACKUP_RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-7}

# Create timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/${POSTGRES_DB}_${TIMESTAMP}.sql.gz"

# Ensure backup directory exists
mkdir -p ${BACKUP_DIR}

echo "Starting PostgreSQL backup at ${TIMESTAMP}"

# Perform backup
pg_dump -h ${POSTGRES_HOST} -p ${POSTGRES_PORT} -U ${POSTGRES_USER} -d ${POSTGRES_DB} | gzip > ${BACKUP_FILE}

# Check if backup was successful
if [ $? -eq 0 ]; then
    echo "Backup completed successfully: ${BACKUP_FILE}"
    
    # Set proper permissions
    chmod 644 ${BACKUP_FILE}
    
    # Remove backups older than retention period
    echo "Removing backups older than ${BACKUP_RETENTION_DAYS} days"
    find ${BACKUP_DIR} -name "${POSTGRES_DB}_*.sql.gz" -type f -mtime +${BACKUP_RETENTION_DAYS} -delete
    
    # List current backups
    echo "Current backups:"
    ls -la ${BACKUP_DIR}
else
    echo "Backup failed!"
    exit 1
fi