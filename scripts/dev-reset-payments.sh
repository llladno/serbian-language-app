#!/bin/sh
# Undo what one local account bought at the stand-in cash desk (PAYMENTS_DEV=1), so the flow can be tried again:
# deletes its provider='dev' payments, the зёрнышки those paid out, and the levels / year they opened.
# Payments made any other way (and zёрнышки earned or spent in the app) are not touched.
#
#   scripts/dev-reset-payments.sh [email]      # default admin@ucimo.local
#   DB=path/to/app.db scripts/dev-reset-payments.sh
#
# Local development only: it edits the SQLite file directly. It is safe to run while `make dev` is up.
set -eu

EMAIL="${1:-admin@ucimo.local}"
DB="${DB:-data/app.db}"

case "$EMAIL" in *"'"* | *'"'* | *';'*) echo "bad email: $EMAIL" >&2; exit 1 ;; esac
[ -f "$DB" ] || { echo "no database at $DB (run from the serbian-app folder, or set DB=)" >&2; exit 1; }

UID_=$(sqlite3 "$DB" "SELECT user_id FROM identities WHERE lower(email) = lower('$EMAIL') LIMIT 1;")
[ -n "$UID_" ] || { echo "no account with the email $EMAIL in $DB" >&2; exit 1; }

sqlite3 "$DB" <<SQL
.bail on
.timeout 5000
BEGIN IMMEDIATE;
CREATE TEMP TABLE dev_payments AS
  SELECT id, tariff_code, status, granted_phase FROM payments WHERE user_id = '$UID_' AND provider = 'dev';
SELECT 'stand-in payments found: ' || count(*) FROM dev_payments;
DELETE FROM user_entitlements WHERE user_id = '$UID_' AND kind = 'phase_unlock'
  AND ref IN (SELECT granted_phase FROM dev_payments WHERE status = 'succeeded' AND granted_phase <> '');
DELETE FROM user_entitlements WHERE user_id = '$UID_' AND kind = 'all_access'
  AND EXISTS (SELECT 1 FROM dev_payments WHERE status = 'succeeded' AND tariff_code = 'year');
DELETE FROM currency_ledger WHERE user_id = '$UID_'
  AND idempotency_key IN (SELECT 'payment:' || id FROM dev_payments);
DELETE FROM payments WHERE id IN (SELECT id FROM dev_payments);
COMMIT;
SQL

# The bonus that was paid out may already have been taken away by hand (an adjustment that cleared the balance),
# and removing it again would leave the balance below zero. Bring it back to zero with an adjustment instead.
BAL=$(sqlite3 "$DB" "SELECT COALESCE(SUM(amount), 0) FROM currency_ledger WHERE user_id = '$UID_';")
if [ "$BAL" -lt 0 ]; then
  TS=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  sqlite3 "$DB" ".timeout 5000" "INSERT INTO currency_ledger (user_id, amount, kind, ref, idempotency_key, comment, created_by, created_at) VALUES ('$UID_', $((-BAL)), 'admin_adjustment', '', 'dev:reset-payments-fix:$TS', 'dev: reset-payments kept the balance from going below zero', 'dev', '$TS');"
  BAL=0
fi
echo "balance now: $BAL"
echo "done for $EMAIL"
