#!/bin/sh
# Make the swallow's tour (onboarding) start again for one local account: forgets its answer (put off / seen to the end),
# and takes back the 5 зёрнышек the tour paid, so the reward can be earned again. The tour still starts by itself only
# after a finished lesson (see "Знакомство с ласточкой" in CLAUDE.md); until then the quest on «Задания» runs it by hand.
#
#   scripts/dev-reset-onboarding.sh [email]      # default admin@ucimo.local
#   DB=path/to/app.db scripts/dev-reset-onboarding.sh
#
# Local development only: it edits the SQLite file directly. It is safe to run while `make dev` is up.
set -eu

EMAIL="${1:-admin@ucimo.local}"
DB="${DB:-data/app.db}"

case "$EMAIL" in *"'"* | *'"'* | *';'*) echo "bad email: $EMAIL" >&2; exit 1 ;; esac
[ -f "$DB" ] || { echo "no database at $DB (run from the serbian-app folder, or set DB=)" >&2; exit 1; }
sqlite3 "$DB" "SELECT 1 FROM onboarding LIMIT 1;" >/dev/null 2>&1 || { echo "no onboarding table: restart make dev once so migration 039 runs" >&2; exit 1; }

UID_=$(sqlite3 "$DB" "SELECT user_id FROM identities WHERE lower(email) = lower('$EMAIL') LIMIT 1;")
[ -n "$UID_" ] || { echo "no account with the email $EMAIL in $DB" >&2; exit 1; }

sqlite3 "$DB" <<SQL
.bail on
.timeout 5000
BEGIN IMMEDIATE;
DELETE FROM currency_ledger WHERE user_id = '$UID_'
  AND idempotency_key IN (SELECT 'quest:$UID_:' || id FROM quests WHERE kind = 'onboarding_done');
DELETE FROM quest_claims WHERE user_id = '$UID_'
  AND quest_id IN (SELECT id FROM quests WHERE kind = 'onboarding_done');
DELETE FROM onboarding WHERE user_id = '$UID_';
COMMIT;
SQL

DONE=$(sqlite3 "$DB" "SELECT count(*) FROM lesson_progress WHERE user_id = '$UID_' AND status = 'done';")
echo "tour reset for $EMAIL (finished lessons: $DONE)"
[ "$DONE" -gt 0 ] || echo "no finished lesson yet: the tour starts by itself after the first one"
