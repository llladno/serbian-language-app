<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { api } from '../api'
import { useSessionStore } from '../stores/session'
import type { LeaderboardMe, LeaderRow } from '../types'

const PAGE_SIZE = 30

const rows = ref<LeaderRow[]>([])
const error = ref<string | null>(null)
const loading = ref(true)
const loadingMore = ref(false)
const hasMore = ref(true)
const session = useSessionStore()

const me = ref<LeaderboardMe | null>(null)
const meLoading = ref(true)

const sentinel = ref<HTMLElement | null>(null)
const sentinelVisible = ref(false)
let observer: IntersectionObserver | null = null

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    const page = await api.leaderboard({ limit: PAGE_SIZE, offset: rows.value.length })
    rows.value.push(...page.rows)
    hasMore.value = page.has_more
  } catch (e) {
    error.value = (e as Error).message
    hasMore.value = false // stop retrying against a broken endpoint
  } finally {
    loadingMore.value = false
  }
  // IntersectionObserver only fires on a visibility *change*. If the
  // sentinel is still in view once this page lands - the viewport fits
  // several pages, or the user was already scrolled to the very bottom -
  // keep going instead of waiting for a scroll event that may never come.
  if (sentinelVisible.value) loadMore()
}

// Separate from the paginated list: "you" might be hundreds of rows down,
// so this is fetched independently instead of waiting for that page to load.
async function loadMe() {
  try {
    me.value = await api.leaderboardMe()
  } catch {
    // non-critical - the page still works without the pinned "you" card
  } finally {
    meLoading.value = false
  }
}

// The list sits behind a <Transition mode="out-in"> (skeleton -> list), which
// mounts the new branch only after the skeleton's CSS leave-transition
// actually finishes - a `nextTick()` after flipping `loading` resolves long
// before that (it just waits for the next reactive flush, not a transition
// event), so `sentinel.value` was still null when observe() used to run and
// the observer silently watched nothing. Watching the ref itself instead
// fires exactly when the element really lands in the DOM, however that
// happens - including if the list ever unmounts and remounts later (error
// retry, etc.), not just on first load.
watch(sentinel, (el) => {
  observer?.disconnect()
  if (!el) return
  // Fetch the next 30 a bit before the sentinel actually reaches the
  // viewport, so scrolling doesn't stall waiting on the request.
  observer = new IntersectionObserver(
    ([entry]) => {
      sentinelVisible.value = entry.isIntersecting
      if (entry.isIntersecting) loadMore()
    },
    { rootMargin: '400px' },
  )
  observer.observe(el)
})

onMounted(async () => {
  loadMe() // runs concurrently with the list below, not awaited
  await loadMore()
  loading.value = false
})

onBeforeUnmount(() => observer?.disconnect())

function activeLabel(d: string) {
  if (!d) return 'ещё не занимался'
  const today = new Date().toISOString().slice(0, 10)
  const y = new Date(Date.now() - 86400000).toISOString().slice(0, 10)
  if (d === today) return 'сегодня'
  if (d === y) return 'вчера'
  return d
}
</script>

<template>
  <h1 class="mb-1 text-2xl font-extrabold">Рейтинг</h1>
  <p class="mb-4 text-sm text-[var(--muted)]">Прогресс всех, кто занимается по этому курсу.</p>

  <template v-if="meLoading || me">
    <p class="mb-2 text-sm font-bold uppercase tracking-wide text-[var(--muted)]">Твой результат</p>
    <Transition name="fade" mode="out-in">
      <div v-if="meLoading" key="me-skel" class="card mb-6 flex items-center gap-3 p-4">
        <div class="skel h-5 w-5 shrink-0 rounded"></div>
        <div class="min-w-0 flex-1 space-y-1.5">
          <div class="skel h-4 w-2/5"></div>
          <div class="skel h-3.5 w-3/5"></div>
        </div>
        <div class="shrink-0 space-y-1.5 text-right">
          <div class="skel ml-auto h-4 w-8"></div>
          <div class="skel h-3 w-16"></div>
        </div>
      </div>

      <div
        v-else-if="me"
        key="me"
        class="card mb-6 flex items-center gap-3 p-4 ring-2 ring-[var(--accent)]"
        style="background: color-mix(in srgb, var(--accent) 7%, var(--card))"
      >
        <span class="w-5 shrink-0 text-center font-mono text-sm font-bold text-[var(--accent)]">{{ me.rank }}</span>
        <div class="min-w-0 flex-1">
          <p class="font-bold">{{ me.row.name }}</p>
          <p class="text-sm text-[var(--muted)]">
            {{ me.row.lessons_done }}/{{ me.row.lessons_total }} уроков · {{ me.row.cards_known }} слов ·
            {{ activeLabel(me.row.last_active) }}
          </p>
        </div>
        <div class="shrink-0 text-right">
          <p class="font-bold">{{ me.row.streak_days }}<span v-if="me.row.streak_days" class="ml-0.5">🔥</span></p>
          <p class="text-[10px] text-[var(--muted)]">дней подряд</p>
        </div>
      </div>
    </Transition>
  </template>

  <p class="mb-2 text-sm font-bold uppercase tracking-wide text-[var(--muted)]">Все участники</p>

  <Transition name="fade" mode="out-in">
    <p v-if="error && !rows.length" key="error" class="card p-4 text-[var(--bad)]">{{ error }}</p>

    <div v-else-if="loading" key="skel" class="space-y-2">
      <div v-for="i in 6" :key="i" class="card flex items-center gap-3 p-4">
        <div class="skel h-5 w-5 shrink-0 rounded"></div>
        <div class="min-w-0 flex-1 space-y-1.5">
          <div class="skel h-4 w-2/5"></div>
          <div class="skel h-3.5 w-3/5"></div>
        </div>
        <div class="shrink-0 space-y-1.5 text-right">
          <div class="skel ml-auto h-4 w-8"></div>
          <div class="skel h-3 w-16"></div>
        </div>
      </div>
    </div>

    <div v-else key="list" class="space-y-2">
      <div
        v-for="(r, i) in rows"
        :key="r.name"
        class="card flex items-center gap-3 p-4"
        :class="r.name === session.user?.name ? 'ring-2 ring-[var(--accent)]' : ''"
      >
        <span class="w-5 shrink-0 text-center font-mono text-sm text-[var(--muted)]">{{ i + 1 }}</span>
        <div class="min-w-0 flex-1">
          <p class="font-bold">
            {{ r.name }}
            <span v-if="r.name === session.user?.name" class="text-xs font-normal text-[var(--accent)]">— это ты</span>
          </p>
          <p class="text-sm text-[var(--muted)]">
            {{ r.lessons_done }}/{{ r.lessons_total }} уроков · {{ r.cards_known }} слов ·
            {{ activeLabel(r.last_active) }}
          </p>
        </div>
        <div class="shrink-0 text-right">
          <p class="font-bold">{{ r.streak_days }}<span v-if="r.streak_days" class="ml-0.5">🔥</span></p>
          <p class="text-[10px] text-[var(--muted)]">дней подряд</p>
        </div>
      </div>

      <p v-if="!rows.length" class="card p-6 text-center text-[var(--muted)]">Пока никого.</p>

      <div ref="sentinel"></div>
      <p v-if="loadingMore" class="p-2 text-center text-sm text-[var(--muted)]">загрузка…</p>
      <p v-if="error && rows.length" class="card p-4 text-sm text-[var(--bad)]">{{ error }}</p>
    </div>
  </Transition>
</template>
