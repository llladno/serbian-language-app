# Broadcast/Notification History + Resend-to-Missing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let Grisha see which Telegram broadcasts and in-app notifications were sent in the past, and resend a given one to only the users who weren't targeted the first time (e.g. registered after it went out) — without adding any new send mechanism.

**Architecture:** `bot_outbox` and `notification_recipients` already record every send at the per-recipient level — this plan exposes that data instead of duplicating it. serbian-app gets one small schema addition (a `broadcasts` table that groups `bot_outbox` rows, since broadcasts currently have no stable id — notifications already have one via `notifications`/`notification_recipients`). ucimo-content-admin's two existing modules (`broadcasts`, `notifications`) each get two new read endpoints (`list`, `missing`) and their two existing Vue pages get a "История" section that computes the missing-recipient set on demand and pre-fills the existing composer — no new send code path.

**Tech Stack:** Go 1.25 for serbian-app (migration only, no behavior change); NestJS + `pg` (via the existing `ProdDbService`) + Vue 3 + Tailwind + Vitest/`@vue/test-utils` for ucimo-content-admin.

**Spec:** [docs/superpowers/specs/2026-09-28-broadcast-history-design.md](../specs/2026-09-28-broadcast-history-design.md)

## Global Constraints

- `broadcasts.id` and `bot_outbox.broadcast_id` are `BIGINT` on Postgres (via the shared `{{.AutoID}}` migration macro) — `node-pg` returns `bigint` columns as JS **strings**, so every id in this feature is typed `string` end to end in TypeScript (service, DTO param, frontend type), matching the existing `notifications.id` convention. Never introduce a `number` id for these.
- "Не получил" (missing) means: never targeted by this specific broadcast/notification at send time. It has nothing to do with Telegram delivery status (`bot_outbox.status`) or notification read-state (`notification_recipients.read_at`) — don't conflate them.
- No Go behavior changes anywhere in this plan — only a migration. `internal/outbox`'s worker and `ListNotificationsForUser` are untouched; they simply don't look at the new column/table.
- `ucimo_admin_ro`'s grants gain: `SELECT, INSERT` on `broadcasts`; `SELECT` on `bot_outbox`, `notifications`, `notification_recipients` (their existing `INSERT` grants are untouched). Nothing else about that role changes.
- Never write a test that performs a real (non-dry-run) `send()` or a direct `INSERT`/`UPDATE` against the tunneled prod DB (`PROD_DATABASE_URL` pointed at the real tunnel). A live `bot_outbox` row is a live Telegram send to a real chat_id; a live `notifications` row is a live in-app notification to real users. Every DB-touching test in this plan stays inside the established "pure function" / "unreachable DB → 503" patterns — see each task's testing note.
- No pagination on the history list, no delivery-status or read-state columns shown in it, no editing/deleting a past broadcast or notification — all explicitly out of scope per the spec.

---

## Part A — serbian-app: schema

### Task 1: Migration 012 — `broadcasts` table + `bot_outbox.broadcast_id`

**Files:**
- Create: `server/internal/store/migrations/012_broadcast_history.sql`
- Modify: `server/internal/store/migrate_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: Postgres/SQLite tables `broadcasts(id, text, created_at)` and `bot_outbox.broadcast_id` (nullable `BIGINT`). Task 2's `BroadcastsService` depends on both.

- [ ] **Step 1: Write the failing test**

Append to the end of `server/internal/store/migrate_test.go` (reuses the `queryOK` helper already defined earlier in this file):

```go
func TestMigration012Schema(t *testing.T) {
	s := newStore(t)
	if err := queryOK(s, `SELECT id, text, created_at FROM broadcasts LIMIT 1`); err != nil {
		t.Fatalf("broadcasts table missing: %v", err)
	}
	if err := queryOK(s, `SELECT broadcast_id FROM bot_outbox LIMIT 1`); err != nil {
		t.Fatalf("bot_outbox.broadcast_id missing: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd server && go test ./internal/store/... -run Migration012 -v`
Expected: FAIL — `broadcasts` table / `broadcast_id` column don't exist yet.

- [ ] **Step 3: Write the migration**

`server/internal/store/migrations/012_broadcast_history.sql`:
```sql
-- Migration 012: broadcasts — groups bot_outbox rows from a single admin
-- Telegram broadcast under a stable id, so ucimo-content-admin can list
-- past broadcasts and compute which current candidates weren't targeted
-- by a given one (e.g. registered after it went out). notifications/
-- notification_recipients need no schema change — they already have this
-- shape. See docs/superpowers/specs/2026-09-28-broadcast-history-design.md.
--
-- broadcast_id is nullable and unconstrained (no REFERENCES — same
-- convention as notification_recipients.notification_id in migration
-- 010): reminders and /start-flow replies keep it NULL, only rows
-- inserted by ucimo-content-admin's broadcast composer set it.

CREATE TABLE IF NOT EXISTS broadcasts (
	id         {{.AutoID}},
	text       TEXT NOT NULL,
	created_at TEXT NOT NULL
);

ALTER TABLE bot_outbox ADD COLUMN broadcast_id BIGINT;
CREATE INDEX IF NOT EXISTS bot_outbox_broadcast_idx ON bot_outbox (broadcast_id);
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd server && go test ./internal/store/... -run Migration012 -v`
Expected: PASS

- [ ] **Step 5: Run the full store suite (migration ordering can affect other tests) and commit**

Run: `cd server && go test ./internal/store/...`
Expected: PASS

```bash
git add server/internal/store/migrations/012_broadcast_history.sql server/internal/store/migrate_test.go
git commit -m "$(cat <<'EOF'
Add broadcasts table + bot_outbox.broadcast_id for history tracking

Groups the send-queue rows a single admin broadcast produces under a
stable id, so past broadcasts can be listed and diffed against the
current candidate set. No Go behavior changes — the outbox worker and
webhook/reminders code don't look at the new column.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part B — ucimo-content-admin: broadcasts (Telegram)

### Task 2: `broadcasts` backend — history list + missing-recipients

**Files:**
- Modify: `server/src/broadcasts/broadcasts.service.ts`
- Modify: `server/src/broadcasts/broadcasts.service.spec.ts`
- Modify: `server/src/broadcasts/broadcasts.controller.ts`
- Modify: `server/src/broadcasts/broadcasts.controller.spec.ts`

**Interfaces:**
- Consumes: `ProdDbService` (existing); `broadcasts` table + `bot_outbox.broadcast_id` (Task 1).
- Produces: `GET /api/broadcasts` → `BroadcastHistoryEntry[]`; `GET /api/broadcasts/:id/missing` → `BroadcastCandidate[]`; `send()` now also creates a `broadcasts` row and stamps `broadcast_id` on every `bot_outbox` row it inserts. `BroadcastsService.resolveMissing(all, targetedChatIds)` — pure, unit-tested. Task 3's frontend depends on these exact shapes.

**Testing note:** same danger as the existing `send()` — a real `INSERT` into `broadcasts`/`bot_outbox` against the tunneled prod DB is a live Telegram send. `resolveMissing` stays pure and is unit-tested directly (like `resolveTargets` already is); `list()`/`missing()`/the modified `send()` are only exercised through the controller's existing "unreachable DB → 503" pattern, never against a real tunnel.

- [ ] **Step 1: Write the failing tests**

Append to `server/src/broadcasts/broadcasts.service.spec.ts` (after the existing `resolveTargets` describe block):
```typescript
describe('BroadcastsService.resolveMissing (pure — no DB involved)', () => {
  const service = new BroadcastsService({} as any); // db is never touched by resolveMissing
  const all: BroadcastCandidate[] = [
    { id: 'u1', name: 'Аня', chatId: '111' },
    { id: 'u2', name: 'Боря', chatId: '222' },
  ];

  it('returns everyone when nobody was targeted', () => {
    expect(service.resolveMissing(all, [])).toEqual(all);
  });

  it('excludes candidates whose chatId was already targeted', () => {
    expect(service.resolveMissing(all, ['111'])).toEqual([all[1]]);
  });

  it('returns nothing when everyone was targeted', () => {
    expect(service.resolveMissing(all, ['111', '222'])).toEqual([]);
  });

  it('ignores targeted chatIds that match no current candidate', () => {
    expect(service.resolveMissing(all, ['999'])).toEqual(all);
  });
});
```

Append to `server/src/broadcasts/broadcasts.controller.spec.ts` (after the existing `describe('POST /broadcasts', ...)` block — reuses the same imports already at the top of the file):
```typescript
describe('GET /broadcasts', () => {
  let app: INestApplication;
  const originalUrl = process.env.PROD_DATABASE_URL;

  beforeAll(async () => {
    delete process.env.PROD_DATABASE_URL;
    const moduleRef: TestingModule = await Test.createTestingModule({
      imports: [BroadcastsModule],
    }).compile();
    app = moduleRef.createNestApplication();
    app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));
    await app.init();
  });

  afterAll(async () => {
    process.env.PROD_DATABASE_URL = originalUrl;
    await app.close();
  });

  it('returns 503 when the prod DB is unreachable', async () => {
    await request(app.getHttpServer()).get('/broadcasts').expect(503);
  });
});

describe('GET /broadcasts/:id/missing', () => {
  let app: INestApplication;
  const originalUrl = process.env.PROD_DATABASE_URL;

  beforeAll(async () => {
    delete process.env.PROD_DATABASE_URL;
    const moduleRef: TestingModule = await Test.createTestingModule({
      imports: [BroadcastsModule],
    }).compile();
    app = moduleRef.createNestApplication();
    app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));
    await app.init();
  });

  afterAll(async () => {
    process.env.PROD_DATABASE_URL = originalUrl;
    await app.close();
  });

  it('rejects a non-numeric id with 400 before ever touching the DB', async () => {
    await request(app.getHttpServer()).get('/broadcasts/not-a-number/missing').expect(400);
  });

  it('returns 503 for a numeric id when the prod DB is unreachable', async () => {
    await request(app.getHttpServer()).get('/broadcasts/7/missing').expect(503);
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd server && npx jest broadcasts`
Expected: FAIL — `resolveMissing` doesn't exist; `GET /broadcasts` and `GET /broadcasts/:id/missing` 404 (no such routes yet), not 503/400.

- [ ] **Step 3: Implement**

Replace `server/src/broadcasts/broadcasts.service.ts` in full:
```typescript
import { Injectable } from '@nestjs/common';
import { ProdDbService } from '../prod-db/prod-db.service';
import { SendBroadcastDto, BroadcastRecipientsDto } from './dto/send-broadcast.dto';

export interface BroadcastCandidate {
  id: string;
  name: string;
  chatId: string;
}

export interface BroadcastHistoryEntry {
  id: string;
  text: string;
  createdAt: string;
  recipientCount: number;
}

interface CandidateRow {
  id: string;
  name: string;
  chat_id: string;
}

interface HistoryRow {
  id: string;
  text: string;
  created_at: string;
  recipient_count: string; // COUNT(...) comes back from pg as a string
}

// Must match server/internal/store/bot_outbox.go's PriorityNormal.
const PRIORITY_NORMAL = 1;

function rfc3339(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

@Injectable()
export class BroadcastsService {
  constructor(private readonly db: ProdDbService) {}

  async candidates(): Promise<BroadcastCandidate[]> {
    const rows = await this.db.query<CandidateRow>(`
      SELECT u.id AS id, u.name AS name, i.provider_uid AS chat_id
      FROM users u
      JOIN identities i ON i.user_id = u.id
      WHERE i.provider = 'telegram' AND i.provider_uid NOT LIKE 'pending:%'
      ORDER BY u.name
    `);
    return rows.map((r) => ({ id: r.id, name: r.name, chatId: r.chat_id }));
  }

  // The pure part of send() — no DB write — so it's unit-testable without
  // touching Postgres (a real INSERT into bot_outbox is a live send, never
  // do that from a test).
  resolveTargets(all: BroadcastCandidate[], recipients: BroadcastRecipientsDto): BroadcastCandidate[] {
    if (recipients.mode === 'all') return all;
    const wanted = new Set(recipients.userIds ?? []);
    return all.filter((c) => wanted.has(c.id));
  }

  // The pure part of missing() — no DB call — so it's unit-testable without
  // touching Postgres.
  resolveMissing(all: BroadcastCandidate[], targetedChatIds: string[]): BroadcastCandidate[] {
    const targeted = new Set(targetedChatIds);
    return all.filter((c) => !targeted.has(c.chatId));
  }

  async send(dto: SendBroadcastDto): Promise<{ count: number }> {
    const all = await this.candidates();
    const targets = this.resolveTargets(all, dto.recipients);

    if (dto.dryRun || targets.length === 0) {
      return { count: targets.length };
    }

    const createdAt = rfc3339(new Date());
    const inserted = await this.db.query<{ id: string }>(
      `INSERT INTO broadcasts (text, created_at) VALUES ($1, $2) RETURNING id`,
      [dto.text, createdAt],
    );
    const broadcastId = inserted[0].id;

    const values: string[] = [];
    const params: unknown[] = [];
    targets.forEach((t, i) => {
      const base = i * 4;
      values.push(`($${base + 1}, $${base + 2}, ${PRIORITY_NORMAL}, 'pending', $${base + 3}, $${base + 4})`);
      params.push(t.chatId, dto.text, createdAt, broadcastId);
    });
    await this.db.query(
      `INSERT INTO bot_outbox (chat_id, text, priority, status, created_at, broadcast_id) VALUES ${values.join(', ')}`,
      params,
    );
    return { count: targets.length };
  }

  async list(): Promise<BroadcastHistoryEntry[]> {
    const rows = await this.db.query<HistoryRow>(`
      SELECT b.id AS id, b.text AS text, b.created_at AS created_at, COUNT(o.id) AS recipient_count
      FROM broadcasts b
      LEFT JOIN bot_outbox o ON o.broadcast_id = b.id
      GROUP BY b.id, b.text, b.created_at
      ORDER BY b.created_at DESC
    `);
    return rows.map((r) => ({
      id: r.id,
      text: r.text,
      createdAt: r.created_at,
      recipientCount: Number(r.recipient_count),
    }));
  }

  async missing(broadcastId: string): Promise<BroadcastCandidate[]> {
    const all = await this.candidates();
    const rows = await this.db.query<{ chat_id: string }>(
      `SELECT chat_id FROM bot_outbox WHERE broadcast_id = $1`,
      [broadcastId],
    );
    return this.resolveMissing(
      all,
      rows.map((r) => r.chat_id),
    );
  }
}
```

Replace `server/src/broadcasts/broadcasts.controller.ts` in full:
```typescript
import { BadRequestException, Body, Controller, Get, Param, Post } from '@nestjs/common';
import { BroadcastsService } from './broadcasts.service';
import { SendBroadcastDto } from './dto/send-broadcast.dto';

@Controller('broadcasts')
export class BroadcastsController {
  constructor(private readonly broadcasts: BroadcastsService) {}

  @Get('candidates')
  candidates() {
    return this.broadcasts.candidates();
  }

  @Get()
  list() {
    return this.broadcasts.list();
  }

  @Get(':id/missing')
  missing(@Param('id') id: string) {
    if (!/^\d+$/.test(id)) {
      throw new BadRequestException('id must be numeric');
    }
    return this.broadcasts.missing(id);
  }

  @Post()
  send(@Body() dto: SendBroadcastDto) {
    return this.broadcasts.send(dto);
  }
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd server && npx jest broadcasts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/src/broadcasts
git commit -m "$(cat <<'EOF'
Add broadcast history + resend-to-missing to the broadcasts backend

send() now groups its bot_outbox rows under a new broadcasts row.
list() surfaces past broadcasts with recipient counts; missing()
computes which current Telegram-linked candidates weren't targeted by
a given broadcast (e.g. registered after it went out). resolveMissing
is pure and unit-tested directly — no test performs a real send.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: `broadcasts` frontend — history section + resend-to-missing

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/api/broadcasts.ts`
- Modify: `web/src/views/BroadcastsView.vue`
- Create: `web/src/views/BroadcastsView.spec.ts`

**Interfaces:**
- Consumes: `GET /api/broadcasts` → `BroadcastHistoryEntry[]`, `GET /api/broadcasts/:id/missing` → `BroadcastCandidate[]` (Task 2).
- Produces: nothing further downstream — this is the leaf of the broadcasts feature.

- [ ] **Step 1: Write the failing test**

Create `web/src/views/BroadcastsView.spec.ts`:
```typescript
import { afterEach, describe, expect, it, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import BroadcastsView from './BroadcastsView.vue';
import * as broadcastsApi from '../api/broadcasts';

afterEach(() => vi.restoreAllMocks());

function stubApi() {
  vi.spyOn(broadcastsApi, 'listCandidates').mockResolvedValue([
    { id: 'u1', name: 'Аня', chatId: '111' },
    { id: 'u2', name: 'Боря', chatId: '222' },
  ]);
  vi.spyOn(broadcastsApi, 'listBroadcastHistory').mockResolvedValue([
    { id: 'b1', text: 'Привет всем!', createdAt: '2026-09-20T10:00:00Z', recipientCount: 2 },
  ]);
  const missingSpy = vi
    .spyOn(broadcastsApi, 'missingRecipients')
    .mockResolvedValue([{ id: 'u3', name: 'Вася', chatId: '333' }]);
  return { missingSpy };
}

describe('BroadcastsView history', () => {
  it('fetches missing recipients only once per row, on first expand', async () => {
    const { missingSpy } = stubApi();
    const wrapper = mount(BroadcastsView);
    await flushPromises();

    const row = wrapper.findAll('button').find((b) => b.text().includes('Привет всем!'));
    if (!row) throw new Error('history row not found');
    await row.trigger('click'); // expand — fetches
    await flushPromises();
    await row.trigger('click'); // collapse — no fetch
    await row.trigger('click'); // expand again — cached, no fetch
    await flushPromises();

    expect(missingSpy).toHaveBeenCalledTimes(1);
    expect(missingSpy).toHaveBeenCalledWith('b1');
  });

  it('prefills the composer with the broadcast text and the missing recipients', async () => {
    stubApi();
    const wrapper = mount(BroadcastsView);
    await flushPromises();

    const row = wrapper.findAll('button').find((b) => b.text().includes('Привет всем!'));
    await row!.trigger('click');
    await flushPromises();

    const resend = wrapper.findAll('button').find((b) => b.text().includes('Отправить недостающим'));
    await resend!.trigger('click');

    const textarea = wrapper.find('textarea');
    expect((textarea.element as HTMLTextAreaElement).value).toBe('Привет всем!');
    const allCheckbox = wrapper.find('input[type="checkbox"]');
    expect((allCheckbox.element as HTMLInputElement).checked).toBe(false);
  });
});
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd web && npx vitest run BroadcastsView`
Expected: FAIL — `listBroadcastHistory`/`missingRecipients` don't exist on the `broadcasts` API module; no history rendered.

- [ ] **Step 3: Implement**

Append to `web/src/types.ts` (after the existing `BroadcastCandidate` interface):
```typescript
export interface BroadcastHistoryEntry {
  id: string;
  text: string;
  createdAt: string;
  recipientCount: number;
}
```

Replace `web/src/api/broadcasts.ts` in full:
```typescript
import type { BroadcastCandidate, BroadcastHistoryEntry } from '../types';

const BASE = '/api/broadcasts';

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.message ?? `Request failed: ${res.status}`);
  }
  return res.json();
}

export function listCandidates(): Promise<BroadcastCandidate[]> {
  return fetch(`${BASE}/candidates`).then((r) => handle(r));
}

export function listBroadcastHistory(): Promise<BroadcastHistoryEntry[]> {
  return fetch(BASE).then((r) => handle(r));
}

export function missingRecipients(id: string): Promise<BroadcastCandidate[]> {
  return fetch(`${BASE}/${id}/missing`).then((r) => handle(r));
}

export type BroadcastRecipients = { mode: 'all' } | { mode: 'users'; userIds: string[] };

export function sendBroadcast(
  text: string,
  recipients: BroadcastRecipients,
  dryRun: boolean,
): Promise<{ count: number }> {
  return fetch(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, recipients, dryRun }),
  }).then((r) => handle(r));
}
```

Replace `web/src/views/BroadcastsView.vue` in full:
```vue
<template>
  <div>
    <h1 class="mb-6 text-2xl font-semibold text-primary-700">Рассылки</h1>
    <p v-if="loading" class="text-sm text-text-500">Загрузка…</p>
    <p v-else-if="error" class="text-sm text-error">{{ error }}</p>
    <div v-else class="flex max-w-xl flex-col gap-4">
      <label class="flex items-center gap-2 text-sm text-text-700">
        <input type="checkbox" v-model="sendToAll" />
        Всем пользователям с Telegram ({{ candidates.length }})
      </label>

      <div v-if="!sendToAll">
        <input
          v-model="search"
          type="text"
          placeholder="Поиск по имени"
          class="mb-2 w-full rounded border border-border px-2 py-1 text-sm"
        />
        <div class="max-h-48 overflow-y-auto rounded border border-border">
          <label
            v-for="c in filteredCandidates"
            :key="c.id"
            class="flex items-center gap-2 border-b border-border px-2 py-1 text-sm last:border-b-0"
          >
            <input type="checkbox" :value="c.id" v-model="selectedIds" />
            {{ c.name }}
          </label>
        </div>
      </div>

      <textarea
        v-model="text"
        rows="5"
        placeholder="Текст сообщения"
        class="w-full rounded border border-border p-2 text-sm"
      ></textarea>

      <button
        type="button"
        class="self-start rounded bg-primary-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        :disabled="!canSend"
        @click="openConfirm"
      >
        Отправить
      </button>
      <p v-if="sendError" class="text-sm text-error">{{ sendError }}</p>
      <p v-if="sentCount !== null" class="text-sm text-success">Поставлено в очередь: {{ sentCount }}</p>

      <div v-if="confirming" class="rounded border border-border bg-white p-4 shadow-sm">
        <p class="mb-3 text-sm text-text-800">
          Отправить {{ pendingCount }} получател{{ pendingCount === 1 ? 'ю' : 'ям' }}?
        </p>
        <div class="flex gap-3">
          <button
            type="button"
            class="rounded bg-primary-600 px-3 py-1.5 text-sm font-medium text-white"
            @click="confirmSend"
          >
            Да, отправить {{ sendToAll ? `всем ${pendingCount}` : pendingCount }}
          </button>
          <button type="button" class="rounded px-3 py-1.5 text-sm text-text-600" @click="confirming = false">
            Отмена
          </button>
        </div>
      </div>

      <div class="mt-4">
        <h2 class="mb-3 text-lg font-semibold text-primary-700">История</h2>
        <p v-if="historyError" class="text-sm text-error">{{ historyError }}</p>
        <p v-else-if="history.length === 0" class="text-sm text-text-500">Рассылок ещё не было.</p>
        <ul v-else class="flex flex-col gap-2">
          <li v-for="entry in history" :key="entry.id" class="rounded border border-border p-2 text-sm">
            <button
              type="button"
              class="flex w-full items-center justify-between gap-2 text-left"
              @click="toggleHistoryRow(entry)"
            >
              <span class="truncate" :title="entry.text">{{ formatDate(entry.createdAt) }} — {{ entry.text }}</span>
              <span class="shrink-0 text-text-500">{{ entry.recipientCount }} получ.</span>
            </button>
            <div v-if="expandedId === entry.id" class="mt-2 border-t border-border pt-2">
              <p v-if="loadingMissingId === entry.id" class="text-text-500">Считаю…</p>
              <template v-else-if="missingCache[entry.id]">
                <p v-if="missingCache[entry.id].length === 0" class="text-success">Все получили</p>
                <div v-else class="flex items-center justify-between gap-2">
                  <span>Не получили: {{ missingCache[entry.id].length }}</span>
                  <button
                    type="button"
                    class="rounded bg-primary-600 px-2 py-1 text-xs font-medium text-white"
                    @click="resendToMissing(entry)"
                  >
                    Отправить недостающим
                  </button>
                </div>
              </template>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { BroadcastCandidate, BroadcastHistoryEntry } from '../types';
import { listCandidates, listBroadcastHistory, missingRecipients, sendBroadcast } from '../api/broadcasts';

const candidates = ref<BroadcastCandidate[]>([]);
const history = ref<BroadcastHistoryEntry[]>([]);
const loading = ref(true);
const error = ref('');
const historyError = ref('');

const sendToAll = ref(true);
const search = ref('');
const selectedIds = ref<string[]>([]);
const text = ref('');

const confirming = ref(false);
const pendingCount = ref(0);
const sendError = ref('');
const sentCount = ref<number | null>(null);

const expandedId = ref<string | null>(null);
const loadingMissingId = ref<string | null>(null);
const missingCache = ref<Record<string, BroadcastCandidate[]>>({});

onMounted(async () => {
  try {
    [candidates.value, history.value] = await Promise.all([listCandidates(), listBroadcastHistory()]);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
});

const filteredCandidates = computed(() =>
  candidates.value.filter((c) => c.name.toLowerCase().includes(search.value.toLowerCase())),
);

const canSend = computed(() => text.value.trim() !== '' && (sendToAll.value || selectedIds.value.length > 0));

function recipients() {
  return sendToAll.value ? { mode: 'all' as const } : { mode: 'users' as const, userIds: selectedIds.value };
}

async function openConfirm() {
  sendError.value = '';
  sentCount.value = null;
  try {
    const { count } = await sendBroadcast(text.value, recipients(), true);
    pendingCount.value = count;
    confirming.value = true;
  } catch (e) {
    sendError.value = e instanceof Error ? e.message : String(e);
  }
}

async function confirmSend() {
  try {
    const { count } = await sendBroadcast(text.value, recipients(), false);
    sentCount.value = count;
    confirming.value = false;
    text.value = '';
    selectedIds.value = [];
  } catch (e) {
    sendError.value = e instanceof Error ? e.message : String(e);
    confirming.value = false;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU');
}

async function toggleHistoryRow(entry: BroadcastHistoryEntry) {
  if (expandedId.value === entry.id) {
    expandedId.value = null;
    return;
  }
  expandedId.value = entry.id;
  if (entry.id in missingCache.value) return;
  loadingMissingId.value = entry.id;
  try {
    missingCache.value[entry.id] = await missingRecipients(entry.id);
  } catch (e) {
    historyError.value = e instanceof Error ? e.message : String(e);
  } finally {
    loadingMissingId.value = null;
  }
}

function resendToMissing(entry: BroadcastHistoryEntry) {
  const missing = missingCache.value[entry.id];
  if (!missing || missing.length === 0) return;
  text.value = entry.text;
  sendToAll.value = false;
  selectedIds.value = missing.map((c) => c.id);
  sentCount.value = null;
  sendError.value = '';
}
</script>
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd web && npx vitest run BroadcastsView`
Expected: PASS

- [ ] **Step 5: Run the full frontend suite and commit**

Run: `cd web && npm run test:unit`
Expected: PASS

```bash
git add web/src/types.ts web/src/api/broadcasts.ts web/src/views/BroadcastsView.vue web/src/views/BroadcastsView.spec.ts
git commit -m "$(cat <<'EOF'
Add broadcast history + resend-to-missing to the composer page

Clicking a past broadcast fetches (and caches) which current
Telegram-linked candidates it never reached, then "Отправить
недостающим" pre-fills the same composer/dry-run/confirm flow with
that set — no new send path.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part C — ucimo-content-admin: notifications (in-app)

### Task 4: `notifications` backend — history list + missing-recipients

**Files:**
- Modify: `server/src/notifications/notifications.service.ts`
- Modify: `server/src/notifications/notifications.service.spec.ts`
- Modify: `server/src/notifications/notifications.controller.ts`
- Modify: `server/src/notifications/notifications.controller.spec.ts`

**Interfaces:**
- Consumes: `ProdDbService` (existing); `notifications`/`notification_recipients` (already exist, no schema change).
- Produces: `GET /api/notifications` → `NotificationHistoryEntry[]`; `GET /api/notifications/:id/missing` → `NotificationCandidate[]`. `NotificationsService.resolveMissing(all, targetedUserIds)` — pure, unit-tested. Task 5's frontend depends on these exact shapes. `send()`'s behavior is unchanged (it already returns `notifications.id` from `RETURNING id`, just never used it further).

**Testing note:** same as Task 2 — a real `INSERT` into `notifications`/`notification_recipients` against the tunneled prod DB is a live in-app notification to real users. `resolveMissing` is pure and unit-tested directly; `list()`/`missing()` are only exercised through the controller's "unreachable DB → 503" pattern.

- [ ] **Step 1: Write the failing tests**

Append to `server/src/notifications/notifications.service.spec.ts` (after the existing `resolveTargets` describe block — `NotificationCandidate`/`NotificationsService` are already imported at the top of this file):
```typescript
describe('NotificationsService.resolveMissing (pure — no DB involved)', () => {
  const service = new NotificationsService({} as any); // db is never touched by resolveMissing
  const all: NotificationCandidate[] = [
    { id: 'u1', name: 'Аня' },
    { id: 'u2', name: 'Боря' },
  ];

  it('returns everyone when nobody was targeted', () => {
    expect(service.resolveMissing(all, [])).toEqual(all);
  });

  it('excludes candidates already targeted', () => {
    expect(service.resolveMissing(all, ['u1'])).toEqual([all[1]]);
  });

  it('returns nothing when everyone was targeted', () => {
    expect(service.resolveMissing(all, ['u1', 'u2'])).toEqual([]);
  });

  it('ignores targeted ids that match no current candidate', () => {
    expect(service.resolveMissing(all, ['does-not-exist'])).toEqual(all);
  });
});
```

Append to `server/src/notifications/notifications.controller.spec.ts` (after the existing `describe('POST /notifications', ...)` block):
```typescript
describe('GET /notifications', () => {
  let app: INestApplication;
  const originalUrl = process.env.PROD_DATABASE_URL;

  beforeAll(async () => {
    delete process.env.PROD_DATABASE_URL;
    const moduleRef: TestingModule = await Test.createTestingModule({
      imports: [NotificationsModule],
    }).compile();
    app = moduleRef.createNestApplication();
    app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));
    await app.init();
  });

  afterAll(async () => {
    process.env.PROD_DATABASE_URL = originalUrl;
    await app.close();
  });

  it('returns 503 when the prod DB is unreachable', async () => {
    await request(app.getHttpServer()).get('/notifications').expect(503);
  });
});

describe('GET /notifications/:id/missing', () => {
  let app: INestApplication;
  const originalUrl = process.env.PROD_DATABASE_URL;

  beforeAll(async () => {
    delete process.env.PROD_DATABASE_URL;
    const moduleRef: TestingModule = await Test.createTestingModule({
      imports: [NotificationsModule],
    }).compile();
    app = moduleRef.createNestApplication();
    app.useGlobalPipes(new ValidationPipe({ whitelist: true, transform: true }));
    await app.init();
  });

  afterAll(async () => {
    process.env.PROD_DATABASE_URL = originalUrl;
    await app.close();
  });

  it('rejects a non-numeric id with 400 before ever touching the DB', async () => {
    await request(app.getHttpServer()).get('/notifications/not-a-number/missing').expect(400);
  });

  it('returns 503 for a numeric id when the prod DB is unreachable', async () => {
    await request(app.getHttpServer()).get('/notifications/7/missing').expect(503);
  });
});
```

- [ ] **Step 2: Run to verify they fail**

Run: `cd server && npx jest notifications`
Expected: FAIL — `resolveMissing` doesn't exist; `GET /notifications` and `GET /notifications/:id/missing` 404.

- [ ] **Step 3: Implement**

Replace `server/src/notifications/notifications.service.ts` in full:
```typescript
import { Injectable } from '@nestjs/common';
import { ProdDbService } from '../prod-db/prod-db.service';
import { SendNotificationDto, NotificationRecipientsDto } from './dto/send-notification.dto';

export interface NotificationCandidate {
  id: string;
  name: string;
}

export interface NotificationHistoryEntry {
  id: string;
  text: string;
  createdAt: string;
  recipientCount: number;
}

interface CandidateRow {
  id: string;
  name: string;
}

interface HistoryRow {
  id: string;
  text: string;
  created_at: string;
  recipient_count: string; // COUNT(...) comes back from pg as a string
}

function rfc3339(d: Date): string {
  return d.toISOString().replace(/\.\d{3}Z$/, 'Z');
}

@Injectable()
export class NotificationsService {
  constructor(private readonly db: ProdDbService) {}

  // Unlike broadcasts/candidates (Telegram-linked users only), every
  // registered user is a valid in-app notification recipient.
  async candidates(): Promise<NotificationCandidate[]> {
    const rows = await this.db.query<CandidateRow>(`SELECT id, name FROM users ORDER BY name`);
    return rows.map((r) => ({ id: r.id, name: r.name }));
  }

  // The pure part of send() — no DB write — so it's unit-testable without
  // touching Postgres.
  resolveTargets(all: NotificationCandidate[], recipients: NotificationRecipientsDto): NotificationCandidate[] {
    if (recipients.mode === 'all') return all;
    const wanted = new Set(recipients.userIds ?? []);
    return all.filter((c) => wanted.has(c.id));
  }

  // The pure part of missing() — no DB call — so it's unit-testable without
  // touching Postgres.
  resolveMissing(all: NotificationCandidate[], targetedUserIds: string[]): NotificationCandidate[] {
    const targeted = new Set(targetedUserIds);
    return all.filter((c) => !targeted.has(c.id));
  }

  async send(dto: SendNotificationDto): Promise<{ count: number }> {
    const all = await this.candidates();
    const targets = this.resolveTargets(all, dto.recipients);

    if (dto.dryRun || targets.length === 0) {
      return { count: targets.length };
    }

    const createdAt = rfc3339(new Date());
    const inserted = await this.db.query<{ id: string }>(
      `INSERT INTO notifications (text, created_at) VALUES ($1, $2) RETURNING id`,
      [dto.text, createdAt],
    );
    const notificationId = inserted[0].id;

    const values: string[] = [];
    const params: unknown[] = [notificationId];
    targets.forEach((t, i) => {
      values.push(`($1, $${i + 2}, '')`);
      params.push(t.id);
    });
    await this.db.query(
      `INSERT INTO notification_recipients (notification_id, user_id, read_at) VALUES ${values.join(', ')}`,
      params,
    );
    return { count: targets.length };
  }

  async list(): Promise<NotificationHistoryEntry[]> {
    const rows = await this.db.query<HistoryRow>(`
      SELECT n.id AS id, n.text AS text, n.created_at AS created_at, COUNT(r.id) AS recipient_count
      FROM notifications n
      LEFT JOIN notification_recipients r ON r.notification_id = n.id
      GROUP BY n.id, n.text, n.created_at
      ORDER BY n.created_at DESC
    `);
    return rows.map((r) => ({
      id: r.id,
      text: r.text,
      createdAt: r.created_at,
      recipientCount: Number(r.recipient_count),
    }));
  }

  async missing(notificationId: string): Promise<NotificationCandidate[]> {
    const all = await this.candidates();
    const rows = await this.db.query<{ user_id: string }>(
      `SELECT user_id FROM notification_recipients WHERE notification_id = $1`,
      [notificationId],
    );
    return this.resolveMissing(
      all,
      rows.map((r) => r.user_id),
    );
  }
}
```

Replace `server/src/notifications/notifications.controller.ts` in full:
```typescript
import { BadRequestException, Body, Controller, Get, Param, Post } from '@nestjs/common';
import { NotificationsService } from './notifications.service';
import { SendNotificationDto } from './dto/send-notification.dto';

@Controller('notifications')
export class NotificationsController {
  constructor(private readonly notifications: NotificationsService) {}

  @Get('candidates')
  candidates() {
    return this.notifications.candidates();
  }

  @Get()
  list() {
    return this.notifications.list();
  }

  @Get(':id/missing')
  missing(@Param('id') id: string) {
    if (!/^\d+$/.test(id)) {
      throw new BadRequestException('id must be numeric');
    }
    return this.notifications.missing(id);
  }

  @Post()
  send(@Body() dto: SendNotificationDto) {
    return this.notifications.send(dto);
  }
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd server && npx jest notifications`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add server/src/notifications
git commit -m "$(cat <<'EOF'
Add notification history + resend-to-missing to the notifications backend

list() surfaces past notifications with recipient counts; missing()
computes which current users weren't targeted by a given one (e.g.
registered after it was sent). resolveMissing is pure and
unit-tested directly — no test performs a real send.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: `notifications` frontend — history section + resend-to-missing

**Files:**
- Modify: `web/src/types.ts`
- Modify: `web/src/api/notifications.ts`
- Modify: `web/src/views/NotificationsView.vue`
- Create: `web/src/views/NotificationsView.spec.ts`

**Interfaces:**
- Consumes: `GET /api/notifications` → `NotificationHistoryEntry[]`, `GET /api/notifications/:id/missing` → `NotificationCandidate[]` (Task 4).
- Produces: nothing further downstream — leaf of the notifications feature.

- [ ] **Step 1: Write the failing test**

Create `web/src/views/NotificationsView.spec.ts`:
```typescript
import { afterEach, describe, expect, it, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import NotificationsView from './NotificationsView.vue';
import * as notificationsApi from '../api/notifications';

afterEach(() => vi.restoreAllMocks());

function stubApi() {
  vi.spyOn(notificationsApi, 'listCandidates').mockResolvedValue([
    { id: 'u1', name: 'Аня' },
    { id: 'u2', name: 'Боря' },
  ]);
  vi.spyOn(notificationsApi, 'listNotificationHistory').mockResolvedValue([
    { id: 'n1', text: 'Новый урок доступен', createdAt: '2026-09-20T10:00:00Z', recipientCount: 2 },
  ]);
  const missingSpy = vi
    .spyOn(notificationsApi, 'missingRecipients')
    .mockResolvedValue([{ id: 'u3', name: 'Вася' }]);
  return { missingSpy };
}

describe('NotificationsView history', () => {
  it('fetches missing recipients only once per row, on first expand', async () => {
    const { missingSpy } = stubApi();
    const wrapper = mount(NotificationsView);
    await flushPromises();

    const row = wrapper.findAll('button').find((b) => b.text().includes('Новый урок доступен'));
    if (!row) throw new Error('history row not found');
    await row.trigger('click'); // expand — fetches
    await flushPromises();
    await row.trigger('click'); // collapse — no fetch
    await row.trigger('click'); // expand again — cached, no fetch
    await flushPromises();

    expect(missingSpy).toHaveBeenCalledTimes(1);
    expect(missingSpy).toHaveBeenCalledWith('n1');
  });

  it('prefills the composer with the notification text and the missing recipients', async () => {
    stubApi();
    const wrapper = mount(NotificationsView);
    await flushPromises();

    const row = wrapper.findAll('button').find((b) => b.text().includes('Новый урок доступен'));
    await row!.trigger('click');
    await flushPromises();

    const resend = wrapper.findAll('button').find((b) => b.text().includes('Отправить недостающим'));
    await resend!.trigger('click');

    const textarea = wrapper.find('textarea');
    expect((textarea.element as HTMLTextAreaElement).value).toBe('Новый урок доступен');
    const allCheckbox = wrapper.find('input[type="checkbox"]');
    expect((allCheckbox.element as HTMLInputElement).checked).toBe(false);
  });
});
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd web && npx vitest run NotificationsView`
Expected: FAIL — `listNotificationHistory`/`missingRecipients` don't exist on the `notifications` API module; no history rendered.

- [ ] **Step 3: Implement**

Append to `web/src/types.ts` (after the existing `NotificationCandidate` interface):
```typescript
export interface NotificationHistoryEntry {
  id: string;
  text: string;
  createdAt: string;
  recipientCount: number;
}
```

Replace `web/src/api/notifications.ts` in full:
```typescript
import type { NotificationCandidate, NotificationHistoryEntry } from '../types';

const BASE = '/api/notifications';

async function handle<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.message ?? `Request failed: ${res.status}`);
  }
  return res.json();
}

export function listCandidates(): Promise<NotificationCandidate[]> {
  return fetch(`${BASE}/candidates`).then((r) => handle(r));
}

export function listNotificationHistory(): Promise<NotificationHistoryEntry[]> {
  return fetch(BASE).then((r) => handle(r));
}

export function missingRecipients(id: string): Promise<NotificationCandidate[]> {
  return fetch(`${BASE}/${id}/missing`).then((r) => handle(r));
}

export type NotificationRecipients = { mode: 'all' } | { mode: 'users'; userIds: string[] };

export function sendNotification(
  text: string,
  recipients: NotificationRecipients,
  dryRun: boolean,
): Promise<{ count: number }> {
  return fetch(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, recipients, dryRun }),
  }).then((r) => handle(r));
}
```

Replace `web/src/views/NotificationsView.vue` in full:
```vue
<template>
  <div>
    <h1 class="mb-6 text-2xl font-semibold text-primary-700">Уведомления</h1>
    <p v-if="loading" class="text-sm text-text-500">Загрузка…</p>
    <p v-else-if="error" class="text-sm text-error">{{ error }}</p>
    <div v-else class="flex max-w-xl flex-col gap-4">
      <label class="flex items-center gap-2 text-sm text-text-700">
        <input type="checkbox" v-model="sendToAll" />
        Всем пользователям ({{ candidates.length }})
      </label>

      <div v-if="!sendToAll">
        <input
          v-model="search"
          type="text"
          placeholder="Поиск по имени"
          class="mb-2 w-full rounded border border-border px-2 py-1 text-sm"
        />
        <div class="max-h-48 overflow-y-auto rounded border border-border">
          <label
            v-for="c in filteredCandidates"
            :key="c.id"
            class="flex items-center gap-2 border-b border-border px-2 py-1 text-sm last:border-b-0"
          >
            <input type="checkbox" :value="c.id" v-model="selectedIds" />
            {{ c.name }}
          </label>
        </div>
      </div>

      <div>
        <textarea
          v-model="text"
          rows="5"
          placeholder="Текст уведомления"
          class="w-full rounded border border-border p-2 text-sm"
        ></textarea>
        <p class="mt-1 text-xs text-text-500">Чтобы добавить ссылку: [текст ссылки](https://...)</p>
      </div>

      <button
        type="button"
        class="self-start rounded bg-primary-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
        :disabled="!canSend"
        @click="openConfirm"
      >
        Отправить
      </button>
      <p v-if="sendError" class="text-sm text-error">{{ sendError }}</p>
      <p v-if="sentCount !== null" class="text-sm text-success">Отправлено: {{ sentCount }}</p>

      <div v-if="confirming" class="rounded border border-border bg-white p-4 shadow-sm">
        <p class="mb-3 text-sm text-text-800">
          Отправить {{ pendingCount }} получател{{ pendingCount === 1 ? 'ю' : 'ям' }}?
        </p>
        <div class="flex gap-3">
          <button
            type="button"
            class="rounded bg-primary-600 px-3 py-1.5 text-sm font-medium text-white"
            @click="confirmSend"
          >
            Да, отправить {{ sendToAll ? `всем ${pendingCount}` : pendingCount }}
          </button>
          <button type="button" class="rounded px-3 py-1.5 text-sm text-text-600" @click="confirming = false">
            Отмена
          </button>
        </div>
      </div>

      <div class="mt-4">
        <h2 class="mb-3 text-lg font-semibold text-primary-700">История</h2>
        <p v-if="historyError" class="text-sm text-error">{{ historyError }}</p>
        <p v-else-if="history.length === 0" class="text-sm text-text-500">Уведомлений ещё не было.</p>
        <ul v-else class="flex flex-col gap-2">
          <li v-for="entry in history" :key="entry.id" class="rounded border border-border p-2 text-sm">
            <button
              type="button"
              class="flex w-full items-center justify-between gap-2 text-left"
              @click="toggleHistoryRow(entry)"
            >
              <span class="truncate" :title="entry.text">{{ formatDate(entry.createdAt) }} — {{ entry.text }}</span>
              <span class="shrink-0 text-text-500">{{ entry.recipientCount }} получ.</span>
            </button>
            <div v-if="expandedId === entry.id" class="mt-2 border-t border-border pt-2">
              <p v-if="loadingMissingId === entry.id" class="text-text-500">Считаю…</p>
              <template v-else-if="missingCache[entry.id]">
                <p v-if="missingCache[entry.id].length === 0" class="text-success">Все получили</p>
                <div v-else class="flex items-center justify-between gap-2">
                  <span>Не получили: {{ missingCache[entry.id].length }}</span>
                  <button
                    type="button"
                    class="rounded bg-primary-600 px-2 py-1 text-xs font-medium text-white"
                    @click="resendToMissing(entry)"
                  >
                    Отправить недостающим
                  </button>
                </div>
              </template>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { NotificationCandidate, NotificationHistoryEntry } from '../types';
import { listCandidates, listNotificationHistory, missingRecipients, sendNotification } from '../api/notifications';

const candidates = ref<NotificationCandidate[]>([]);
const history = ref<NotificationHistoryEntry[]>([]);
const loading = ref(true);
const error = ref('');
const historyError = ref('');

const sendToAll = ref(true);
const search = ref('');
const selectedIds = ref<string[]>([]);
const text = ref('');

const confirming = ref(false);
const pendingCount = ref(0);
const sendError = ref('');
const sentCount = ref<number | null>(null);

const expandedId = ref<string | null>(null);
const loadingMissingId = ref<string | null>(null);
const missingCache = ref<Record<string, NotificationCandidate[]>>({});

onMounted(async () => {
  try {
    [candidates.value, history.value] = await Promise.all([listCandidates(), listNotificationHistory()]);
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e);
  } finally {
    loading.value = false;
  }
});

const filteredCandidates = computed(() =>
  candidates.value.filter((c) => c.name.toLowerCase().includes(search.value.toLowerCase())),
);

const canSend = computed(() => text.value.trim() !== '' && (sendToAll.value || selectedIds.value.length > 0));

function recipients() {
  return sendToAll.value ? { mode: 'all' as const } : { mode: 'users' as const, userIds: selectedIds.value };
}

async function openConfirm() {
  sendError.value = '';
  sentCount.value = null;
  try {
    const { count } = await sendNotification(text.value, recipients(), true);
    pendingCount.value = count;
    confirming.value = true;
  } catch (e) {
    sendError.value = e instanceof Error ? e.message : String(e);
  }
}

async function confirmSend() {
  try {
    const { count } = await sendNotification(text.value, recipients(), false);
    sentCount.value = count;
    confirming.value = false;
    text.value = '';
    selectedIds.value = [];
  } catch (e) {
    sendError.value = e instanceof Error ? e.message : String(e);
    confirming.value = false;
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString('ru-RU');
}

async function toggleHistoryRow(entry: NotificationHistoryEntry) {
  if (expandedId.value === entry.id) {
    expandedId.value = null;
    return;
  }
  expandedId.value = entry.id;
  if (entry.id in missingCache.value) return;
  loadingMissingId.value = entry.id;
  try {
    missingCache.value[entry.id] = await missingRecipients(entry.id);
  } catch (e) {
    historyError.value = e instanceof Error ? e.message : String(e);
  } finally {
    loadingMissingId.value = null;
  }
}

function resendToMissing(entry: NotificationHistoryEntry) {
  const missing = missingCache.value[entry.id];
  if (!missing || missing.length === 0) return;
  text.value = entry.text;
  sendToAll.value = false;
  selectedIds.value = missing.map((c) => c.id);
  sentCount.value = null;
  sendError.value = '';
}
</script>
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd web && npx vitest run NotificationsView`
Expected: PASS

- [ ] **Step 5: Run the full frontend suite and commit**

Run: `cd web && npm run test:unit`
Expected: PASS

```bash
git add web/src/types.ts web/src/api/notifications.ts web/src/views/NotificationsView.vue web/src/views/NotificationsView.spec.ts
git commit -m "$(cat <<'EOF'
Add notification history + resend-to-missing to the composer page

Same pattern as the broadcasts page (Task 3): clicking a past
notification fetches (and caches) which current users it never
reached, then "Отправить недостающим" pre-fills the same
composer/dry-run/confirm flow with that set.

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
EOF
)"
```

---

## Part D — Deploy

### Task 6: Deploy serbian-app, verify migration 012, grant new permissions

**Files:** none (operational task)

- [ ] **Step 1: Push and deploy serbian-app**

```bash
cd /Users/grisha/plans/serbian-app
git push origin main
```
Deploy via the Dokploy API (`POST /api/application.deploy` with serbian-app's `applicationId` — see your deploy notes/memory for the base URL, API key, and id; do not paste them into this plan file). This runs migration 012 automatically on boot (`store.Open` applies pending migrations before the server starts serving).

- [ ] **Step 2: Verify migration 012 applied**

Using the SSH access documented in your deploy notes, connect to the prod Postgres and confirm:
```sql
SELECT count(*) FROM broadcasts;  -- want 0 (freshly created, empty)
SELECT column_name FROM information_schema.columns
  WHERE table_name = 'bot_outbox' AND column_name = 'broadcast_id';  -- want 1 row
```

- [ ] **Step 3: Grant `ucimo_admin_ro` its new permissions**

Still connected as an admin role, run:
```sql
GRANT SELECT, INSERT ON broadcasts TO ucimo_admin_ro;
GRANT SELECT ON bot_outbox TO ucimo_admin_ro;
GRANT SELECT ON notifications TO ucimo_admin_ro;
GRANT SELECT ON notification_recipients TO ucimo_admin_ro;
```
This is additive to that role's existing `INSERT`-only access on these four tables — nothing else about it changes, per the spec's "Права на проде" section. No sequence grant is needed for `broadcasts.id` (Postgres identity columns don't require one, unlike `serial`).

- [ ] **Step 4: (Optional, manual, only if you want it) backfill the one broadcast you already sent**

The history list only shows `broadcasts` rows created from now on — the message you already sent has no `broadcast_id` on its `bot_outbox` rows and won't retroactively appear. If you want it to show up (and get a working "Отправить недостающим" for it) immediately, back it in by hand using its **exact** text — do not use a blanket query grouping by `text` alone, since `bot_outbox` priority 1 is shared with periodic reminders (`reminders.go`'s random message pools), which would get swept in too:
```sql
INSERT INTO broadcasts (text, created_at)
SELECT text, MIN(created_at) FROM bot_outbox
WHERE text = '<paste the exact broadcast text here>' AND broadcast_id IS NULL
GROUP BY text
RETURNING id;
-- then, using the id just returned:
UPDATE bot_outbox SET broadcast_id = <id from above>
WHERE text = '<the same exact text>' AND broadcast_id IS NULL;
```
Skip this entirely if you'd rather just let the next broadcast you send through the updated composer be the first one tracked.

- [ ] **Step 5: Verify the deployed binary is actually live**

Per your deploy notes' "Verify after deploy" section: confirm the `deployments[]` entry for this commit shows `status:"done"`, then poll `https://ucimo.ru/` once more and diff the hashed JS asset filename against a fresh local `npm run build` (there's a documented ~1-2 min lag between "done" and the new container actually serving).

No commit for this task (nothing in the repo changes).

---

### Task 7: Deploy ucimo-content-admin

**Files:** none (operational task)

- [ ] **Step 1: Build and push the image, redeploy**

Follow the existing no-git-remote build-on-host flow (rsync the repo to the Dokploy host, `docker build`, tag/push to the local registry, `POST /api/application.deploy` with ucimo-content-admin's `applicationId`) — see your deploy notes for the exact commands and credentials; do not paste them into this plan file.

- [ ] **Step 2: Verify**

```bash
curl -s -o /dev/null -w '%{http_code}\n' https://admin.ucimo.ru/api/broadcasts                       # 401, no creds
curl -s -o /dev/null -w '%{http_code}\n' -u "admin:<password>" https://admin.ucimo.ru/api/broadcasts  # 200
curl -s -o /dev/null -w '%{http_code}\n' -u "admin:<password>" https://admin.ucimo.ru/api/notifications  # 200
```
Both authenticated calls should return a JSON array — empty, or containing the backfilled broadcast if Task 6 Step 4 was done.

No commit for this task (nothing in the repo changes).

---

### Task 8: End-to-end manual smoke test

**Files:** none (verification task)

- [ ] **Step 1: History renders**

Open `https://admin.ucimo.ru/bot/broadcasts`. Confirm the "История" section appears below the composer, listing past broadcasts (or "Рассылок ещё не было." if none/no backfill).

- [ ] **Step 2: Missing-count and resend, on a real (small) broadcast**

Send a short distinctive test broadcast to just your own account (uncheck "всем", select only yourself, confirm, send). Reload the page — the new entry should appear in "История" with "1 получ.". Click it: expect "Все получили" (you were the only candidate and you were targeted).

- [ ] **Step 3: Missing recipients after a new registration**

If you can create or use a second Telegram-linked test account: after step 2's broadcast, link that second account to a test user (or use one already linked but excluded from step 2's send). Reopen the same history row — expect "Не получили: 1". Click "Отправить недостающим", confirm the composer now shows the original text with only that one recipient selected, then send it and confirm delivery.

- [ ] **Step 4: Repeat steps 1-3 for notifications**

Same checks at `https://admin.ucimo.ru/notifications` — in-app notification bell should show the resent notification for the previously-missing user on their next poll (per `serbian-app`'s existing `NotificationBell.vue`, unrelated to this plan).

No commit for this task (verification only). If any step fails, file it as a bug against the relevant task above rather than patching ad hoc — reopen that task's checklist.

---

## Self-Review Notes

**Spec coverage:** `broadcasts` table + `bot_outbox.broadcast_id` migration (Task 1); `ucimo_admin_ro` grants (Task 6); `BroadcastsService.list()`/`missing()`/`resolveMissing()`, `send()` grouping into `broadcasts` (Task 2); mirrored for notifications (Task 4); history UI with on-demand cached missing-count and "Отправить недостающим" pre-filling the existing composer/dry-run/confirm flow, for both pages (Tasks 3, 5); no pagination / no delivery-or-read-state columns / no edit-delete (respected by design, not built).

**Deviation from the spec's testing section, and why:** the spec says to extend `broadcasts.service.spec.ts`/`notifications.service.spec.ts` "через реальный Postgres по образцу bot-messages/links, без моков." Investigating the actual codebase during planning showed this restates a mis-paraphrase already caught and corrected once before (see `2026-09-22-bot-messages-and-broadcasts.md`'s own Self-Review Notes): `links.service.spec.ts` tests for real against **Prisma** (the admin's own local content DB), not `ProdDbService`, and no `ProdDbService`-backed module in this codebase has ever run a real round-trip test against the tunneled serbian-app DB — for `broadcasts`/`notifications` specifically that would mean a live Telegram send or a live in-app notification from a test run. Tasks 2 and 4 follow the actually-established pattern instead (pure-function unit tests for `resolveMissing`, plus the controller's existing "unreachable DB → 503" / validation-400 tests), and call this out in each task's testing note.

**Extra beyond the spec:** Task 6 Step 4 (optional manual backfill of the one broadcast already sent) isn't in the spec — it exists because the spec's "no automatic backfill" YAGNI call was about *future* registrations, not about making the feature immediately useful for the specific message that motivated this whole plan. It's explicitly optional, manual, and scoped to an exact-text match (not a heuristic query) to avoid misclassifying `reminders.go`'s periodic messages, which share `bot_outbox` priority 1.
