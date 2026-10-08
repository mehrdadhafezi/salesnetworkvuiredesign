#!/usr/bin/env bash
# Run as root on the user's cPanel host BEFORE replacing the current plugin.
set -euo pipefail
umask 077
crm_site=/home/payasetareh/crm.payasetareh.ir
crm_account=payasetareh
crm_backup_root=/home/payasetareh/private-crm-backups
crm_backup="$crm_backup_root/$(date +%Y%m%d-%H%M%S)"
test -f "$crm_site/wp-config.php"
test -d "$crm_site/wp-content/plugins/sales-network-v4"
mkdir -p "$crm_backup"
chmod 700 "$crm_backup_root" "$crm_backup"
chown "$crm_account:$crm_account" "$crm_backup_root" "$crm_backup"
crm_wp=$(command -v wp)
runuser -u "$crm_account" -- "$crm_wp" --path="$crm_site" --skip-plugins --skip-themes db export "$crm_backup/database.sql" --single-transaction --quick
test -s "$crm_backup/database.sql"
gzip "$crm_backup/database.sql"
gzip -t "$crm_backup/database.sql.gz"
tar -czf "$crm_backup/site-files.tar.gz" -C "$crm_site" .
tar -tzf "$crm_backup/site-files.tar.gz" >/dev/null
sha256sum "$crm_backup/database.sql.gz" "$crm_backup/site-files.tar.gz" > "$crm_backup/SHA256SUMS"
chmod 600 "$crm_backup"/*
chown "$crm_account:$crm_account" "$crm_backup"/*
printf 'BACKUP_OK\n%s\n' "$crm_backup"
