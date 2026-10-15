# Tariffs (paid access for rubles) — design

Status: **draft for review, nothing implemented.** Decisions below were made in chat
on 2026-10-14; the open items at the end block the build.

## Goal

Let a user pay real money for the paid levels (phases 4 and up) without grinding
the in-app currency. The currency path stays exactly as in
`2026-09-28-currency-design.md`: it is the free road, money is the shortcut.

This reverses one line of the currency spec ("no pay-with-money unlock path; money
buys currency"). Reason: Grisha wants tariffs that name a course, not a currency
amount, and a bundle of currency would open everything at once (1000 > the 605 that
levels 4 and 5 cost together), which makes "one level" meaningless.

## The two tariffs

| code | name | price | what the user gets |
|---|---|---|---|
| `level` | «Уровень» | **299 ₽**, first purchase **249 ₽** (299 shown struck through) | the next level the user has not opened, **forever** |
| `year` | «Год» | **1000 ₽** (2000 ₽ shown struck through) | every level for 365 days, including levels released during that time |

Decisions behind the table:

- **No auto-renew.** Both are one-off payments. No saved card, no scheduler, no
  cancel screen. A reminder before the year ends goes through the bot outbox.
- **«Месяц» is gone.** A level has ~17 lessons; locking it after 30 days would cut
  learners off mid-level. «Уровень» is a permanent unlock.
- **No currency in the tariffs.** A bonus of 1000 зёрнышек would open levels 4 and 5
  on its own and leave nothing for «Уровень» to sell. Tariffs and the currency
  ladder do not touch each other.
- **The user does not choose the level.** Levels are not independent: level 5 leans on
  the cases taught in level 4 (hint 76.2.8 even points at lesson 47). The app does
  not enforce the order today (each level is its own `phase_unlock` product), so
  the tariff does it. A «choose another» button is a later feature, worth adding
  when there are ~10 paid levels.

### "Next unopened level"

The first phase in `course.yaml` order that (a) has an active `phase_unlock` product
with price > 0, (b) is not owned by the user (`user_entitlements`), and (c) is for
sale (at least one lesson is not `planned`, the same rule as `forSale()` in
`CourseView.vue`). If there is none, `level` is unavailable and the card says
«Новые уровни скоро». With levels 4 and 5 that means a user can buy `level` twice.

### First-purchase price

249 ₽ applies when the user has no succeeded `level` payment. Everyone else pays
299 ₽. `year` has one price. All three numbers live in the `tariffs` table and are
editable from the admin, like every other number in the economy.

### What `year` does and does not do

- While `now < expires_at`, `phaseAccess` treats every phase as unlocked.
- Buying `year` while it is active extends from the current `expires_at`, not from now.
- When it ends, the levels it opened lock again. Levels bought with `level` or with
  currency stay open, because they are real `phase_unlock` entitlements. (Open item 3
  asks whether to be kinder than this.)

## Data model (migration `024_tariffs.sql`)

```sql
CREATE TABLE tariffs (
	code              TEXT PRIMARY KEY,          -- level | year
	title             TEXT NOT NULL,
	price_kopecks     BIGINT NOT NULL,
	first_price_kopecks BIGINT NOT NULL DEFAULT 0, -- 0 = no first-purchase price
	old_price_kopecks BIGINT NOT NULL DEFAULT 0, -- struck-through price, 0 = none
	period_days       INTEGER NOT NULL DEFAULT 0, -- 0 = permanent unlock
	active            INTEGER NOT NULL DEFAULT 1,
	updated_at        TEXT NOT NULL
);

CREATE TABLE payments (
	id                  {{.AutoID}},
	user_id             TEXT NOT NULL,
	tariff_code         TEXT NOT NULL,
	granted_phase       TEXT NOT NULL DEFAULT '',  -- level: the phase picked at order time
	amount_kopecks      BIGINT NOT NULL,
	provider            TEXT NOT NULL,             -- yookassa
	provider_payment_id TEXT NOT NULL,
	status              TEXT NOT NULL,             -- pending | succeeded | canceled
	receipt_contact     TEXT NOT NULL DEFAULT '',  -- email or phone the receipt went to
	created_at          TEXT NOT NULL,
	paid_at             TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX payments_provider_id ON payments (provider, provider_payment_id);

ALTER TABLE user_entitlements ADD COLUMN expires_at TEXT NOT NULL DEFAULT ''; -- '' = never
```

The year is a row `kind='all_access', ref='year'` with `expires_at` set. The level is
the existing `kind='phase_unlock', ref=<N>, qty=1`, so `phaseAccess` needs only one
new branch (all-access and not expired → nothing is locked).

The level is **fixed at order time** (`granted_phase`), not recomputed at the webhook.
Otherwise two quick orders both resolve to the same "next" level and the user pays twice
for one phase.

## The learner's flow (frontend built 2026-10-14, payments not connected)

Where the paths into a payment are, and what each one does:

| Where | What the learner sees | What the button does |
|---|---|---|
| Not enough зёрнышки for a level (modal on the course screen) | the shortfall, then a primary button next to «Как заработать» | the next level: «Открыть сразу за 249 ₽» → `/tariffs`. A level that is not the next one: «Все уровни — 1 000 ₽ в год», with the reason (levels open in order) |
| Quests screen | a banner under the balance: «Не хочется копить? Открой «Уровень 4» сразу», 249 ₽, link to the year | → `/tariffs`. Hidden when no level is left or a year is running |
| `/tariffs` | two plans; with a running year, only «Мой тариф» | pay (not wired yet: a toast says so) |
| «Мой тариф» (a running year) | until when, days left, «Автопродления нет», the reminder switch, «Продлить на год» | extend from the current end date |

**Unsubscribing.** Both plans are one-off payments, so there is no subscription to
cancel and the screen says so in as many words. What a person can switch off is the
bot's messages: study reminders, tariff reminders (the year ending) and news, each
on its own, from the profile, from the «Мой тариф» card, from a button under every
reminder, and with `/stop` / `/settings` in the bot. Sign-in and payment messages are
not switchable. Details and the rest of what the bot could write:
`docs/TELEGRAM_MESSAGES.md`. If auto-renew is ever added (it needs saved cards and a
scheduler, see "Decisions"), this is where «Отключить автопродление» goes, in the same
card.

**Not a top-up.** The banner and the modal open levels; they do not sell зёрнышки, as
decided above. A «купить N зёрнышек» pack would be a different product (`topup` in the
ledger) and is not built.

## Payment flow (ЮKassa, web only)

1. `POST /api/me/orders {tariff}` → validates the tariff and (for `level`) the next
   level, creates a `pending` payment, asks ЮKassa to create the payment with a redirect
   confirmation and a receipt, returns `confirmation_url`. The receipt needs an email or
   a phone: ask for it at checkout when the account has neither (Telegram-only accounts
   do not). The contact goes in the request body, never in a URL.
2. The user pays on ЮKassa and returns to `/profile`.
3. `POST /api/yookassa/webhook` (event `payment.succeeded`). ЮKassa notifications are not
   signed with a shared secret like Tribute's, so the handler **ignores the body's claim**
   and re-fetches the payment from the ЮKassa API by id, then trusts only that. Add the
   route to the CSRF origin exemption like `/api/tribute/webhook`.
4. In one transaction: mark the payment `succeeded` (the unique index makes a retried
   webhook a no-op), then grant the entitlement. Queue a bot message through the outbox
   («Уровень N открыт»).
5. Non-200 on a store error so ЮKassa retries; 200 for events we ignore.

Receipt: use «Чеки от ЮKassa»; one item named after the tariff, no VAT (`vat_code` 1)
if the entrepreneur is on УСН. Confirm the tax code with the accountant before launch.

## Inside Telegram

Telegram's rules (core.telegram.org/bots/payments-stars) require Stars for digital goods
sold **inside Telegram apps**. The app runs as the bot's Mini App, so the buy button
must not take rubles there. Proposed default for v1 (awaiting Grisha's yes): the Mini App
shows the tariffs and a «Оплатить на сайте» button that opens the **external browser**
(`Telegram.WebApp.openLink`) on a one-time pay link, so the payment itself happens on
the website, not inside Telegram. The link is a short-lived single-use token bound to the
user and valid only for creating that order; without it the user would have to sign in
again in the browser and conversion would drop. After paying, the Mini App re-reads
access when it regains focus.

Not verified: the Mini App terms (telegram.org/tos/mini-apps) have no clause on linking
out, and the Stars page speaks only of sales "inside Telegram apps", so this is neither
allowed nor forbidden in writing. Section 6 of those terms lets Telegram restrict a
service provider, so it is a risk, not a certainty. Stars as a fully compliant
in-Telegram path is a later iteration (withdrawal goes through Fragment/TON after a
21-day hold).

## Numbers

Net of ЮKassa (cards 2.8% + 1% receipt) and 6% УСН, assumed: ≈ 902 per 1000 ₽.

| sale | gross | net |
|---|---|---|
| `level`, first | 249 | ≈ 225 |
| `level` | 299 | ≈ 270 |
| `year` | 1000 | ≈ 902 |

For 100 000 ₽ net a month (ignoring hosting and other costs): ≈ 370 `level` sales a
month, or ≈ 1 330 `year` sales a year. Until levels 6+ exist, one user brings at most
249 + 299 = 548 ₽ through `level` (two paid levels), while `year` brings 1000 ₽ for the
same content. That is deliberate: the year is priced on the promise of new levels, so
it needs levels to keep shipping (`ROADMAP_LEVELS_6_20.md` is a hypothesis, not a plan).

## Open items

1. **ЮKassa shop.** Grisha confirmed on 2026-10-14 that he is an ИП, so the 54-ФЗ flow
   («Чеки от ЮKassa») applies. Still needed: an approved shop (`YOOKASSA_SHOP_ID`,
   `YOOKASSA_SECRET_KEY` as Dokploy env vars, never committed). ЮKassa reviews the site
   when connecting a shop; expect to need an offer, the entrepreneur's details,
   contacts, priced descriptions, refund terms and a personal-data policy on ucimo.ru
   (verify against ЮKassa's own checklist). Development can start against a ЮKassa test
   shop. Also check that the prod host can reach `api.yookassa.ru` outbound; Telegram's
   API is blocked from that host, so do not assume.
2. **Telegram rule.** Is a read-only card with a "pay on the website" note acceptable,
   or should the Mini App hide tariffs entirely? Share of Mini App vs browser users
   (identity provider in prod, read-only) would size the loss.
3. **After the year ends.** Default above: levels it opened lock again. Kinder option:
   keep the levels the user actually started. It costs renewals; decide before launch.
4. **Struck-through prices.** 299 is a real price after the first purchase. 2000 for
   the year has never been charged. As far as I know a "was" price nobody ever paid can
   be treated as misleading under Russian advertising rules (not legal advice). Safer
   copy: «Ранняя цена 1000 ₽, потом 2000 ₽», then really raise it.
5. **Copy.** The landing and blog say "без подписки" next to the three free levels.
   That stays true (no subscription, one-off payments), but re-read the FAQ before
   launch.
6. **Refunds** are out of scope: done by hand in the ЮKassa cabinet, then the admin
   removes the entitlement.
