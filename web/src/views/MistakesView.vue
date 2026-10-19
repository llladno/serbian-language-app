<script setup lang="ts">
// "Работа над ошибками": the exercises the learner last got wrong, answered
// again one after another without going back into their lessons. A wrong answer
// goes to the back of the line (lib/mistakesQueue); answering goes through the
// ordinary check endpoint, which is what takes a fixed exercise off the list.
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, CircleCheckBig, RotateCcw, X } from 'lucide-vue-next'
import { api } from '../api'
import { provideWrongGoesToEnd } from '../lib/lessonRules'
import { pluralRu } from '../lib/plural'
import {
  createMistakesState,
  current,
  isDone,
  keyOf,
  next,
  record,
  type MistakesState,
} from '../lib/mistakesQueue'
import ExerciseItem from '../components/exercises/ExerciseItem.vue'
import BottomBar from '../components/BottomBar.vue'

const router = useRouter()
// As in a lesson: after a mistake the way forward is the bottom button (the
// exercise comes back later), not an instant redo that would show the answer.
provideWrongGoesToEnd(ref(true))

const state = ref<MistakesState | null>(null)
const loading = ref(true)
const failed = ref(false)
// Mistakes still waiting beyond the batch on screen.
const more = ref(0)

// A learner who cannot listen gets dictations as sentences to translate. One
// switch for the whole screen, remembered for next time.
const SILENT_KEY = 'mistakes.silent'
function readSilent() {
  try {
    return localStorage.getItem(SILENT_KEY) === '1'
  } catch {
    return false
  }
}
function writeSilent(on: boolean) {
  try {
    if (on) localStorage.setItem(SILENT_KEY, '1')
    else localStorage.removeItem(SILENT_KEY)
  } catch {
    /* private mode: the switch just won't outlive the screen */
  }
}
const silent = ref(readSilent())

async function load() {
  loading.value = true
  failed.value = false
  try {
    const res = await api.mistakes(silent.value ? { audioOff: true } : {})
    state.value = createMistakesState(res.items)
    more.value = Math.max(0, res.total - res.items.length)
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}
onMounted(load)

const cur = computed(() => (state.value ? current(state.value) : undefined))
const round = computed(() => (cur.value && state.value ? (state.value.rounds[keyOf(cur.value)] ?? 0) : 0))
const finished = computed(() => !!state.value && state.value.total > 0 && isDone(state.value))
const empty = computed(() => !!state.value && state.value.total === 0)
const pct = computed(() => (state.value?.total ? Math.round((state.value.fixed / state.value.total) * 100) : 0))

// Same as the lesson player: these types pin their own "Проверить" to the
// bottom of the screen until answered, so only one bottom action shows at once.
// Choice grades itself on tap.
const OWN_ACTION = new Set(['translate', 'fill_blank', 'fix_error', 'listen', 'word_bank', 'conjugate', 'match'])
const showBar = computed(() => {
  const s = state.value
  const it = cur.value
  if (!s || !it) return false
  return s.answered !== null || !OWN_ACTION.has(it.exercise.type)
})
const wrong = computed(() => state.value?.answered === false)
const last = computed(() => (state.value?.pending.length ?? 0) === 1)

function onGraded(ok: boolean) {
  if (state.value) record(state.value, ok)
}
function onUngraded() {
  if (state.value) state.value.answered = null
}
function advance() {
  if (state.value) next(state.value)
}

async function cantListen() {
  const s = state.value
  if (!s || silent.value) return
  silent.value = true
  writeSilent(true)
  try {
    const res = await api.mistakes({ audioOff: true })
    const fresh = new Map(res.items.map((m) => [keyOf(m), m]))
    s.pending = s.pending.map((m) => (m.exercise.type === 'listen' ? (fresh.get(keyOf(m)) ?? m) : m))
  } catch {
    // Without the text there is nothing to put in the audio's place.
    silent.value = false
    writeSilent(false)
  }
}

const close = () => router.push('/profile')
</script>

<template>
  <div class="flex min-h-[calc(100dvh-11rem)] flex-col">
    <header class="shrink-0 pb-4">
      <div class="flex w-full items-center">
        <div class="min-w-0 flex-1">
          <p class="truncate text-[14px] font-semibold">Работа над ошибками</p>
          <p class="truncate text-[11px] text-[var(--muted)]">те же упражнения, без возврата в уроки</p>
        </div>
        <button class="icon-btn ml-auto shrink-0" title="Закрыть" aria-label="Закрыть" @click="close">
          <X :size="20" :stroke-width="2.25" />
        </button>
      </div>
      <template v-if="state && state.total > 0">
        <div class="mt-3 h-1.5 overflow-hidden rounded-full bg-[var(--bg-soft)]">
          <div class="h-full rounded-full bg-[var(--accent)] transition-all" :style="{ width: pct + '%' }" />
        </div>
        <p class="mt-1.5 text-xs text-[var(--muted)]">{{ state.fixed }} из {{ state.total }}</p>
      </template>
    </header>

    <div class="flex flex-1 flex-col justify-center">
      <div v-if="loading" class="space-y-3">
        <div class="skel h-6 w-2/3"></div>
        <div class="skel h-12 w-full"></div>
        <div class="skel h-12 w-full"></div>
      </div>

      <div v-else-if="failed" class="card p-6 text-center">
        <p class="text-[var(--muted)]">Не получилось загрузить ошибки.</p>
        <button class="btn btn-ghost mt-3" @click="load">Повторить</button>
      </div>

      <div v-else-if="empty" class="card p-8 text-center">
        <p class="text-lg font-bold">Ошибок нет</p>
        <p class="mt-1 text-sm text-[var(--muted)]">Всё, что ты проходил, отвечено верно.</p>
        <button class="btn btn-primary mt-5" @click="close">К прогрессу</button>
      </div>

      <div v-else-if="finished && state" class="card p-8 text-center">
        <CircleCheckBig :size="36" :stroke-width="2" class="mx-auto text-[var(--good)]" />
        <p class="mt-3 text-lg font-bold">Все ошибки исправлены</p>
        <p class="mt-1 text-sm text-[var(--muted)]">
          Исправлено {{ state.fixed }}, с первого раза — {{ state.fixedFirstTry }}
        </p>
        <button v-if="more > 0" class="btn btn-primary mt-5" @click="load">
          Ещё {{ more }} {{ pluralRu(more, 'ошибка', 'ошибки', 'ошибок') }}
        </button>
        <button class="btn mt-3" :class="more > 0 ? 'btn-ghost' : 'btn-primary'" @click="close">Готово</button>
      </div>

      <div v-else-if="cur" :key="keyOf(cur) + ':' + round">
        <ExerciseItem
          :lesson="cur.lesson"
          :exercise="cur.exercise"
          @graded="onGraded"
          @ungraded="onUngraded"
          @cant-listen="cantListen"
        />
      </div>
    </div>
  </div>

  <BottomBar v-if="showBar">
    <button class="btn btn-primary w-full" :disabled="state?.answered === null" @click="advance">
      <template v-if="wrong">
        <template v-if="last"><RotateCcw :size="16" :stroke-width="2.5" />Ещё раз</template>
        <template v-else>Вернёмся позже<ArrowRight :size="16" :stroke-width="2.5" /></template>
      </template>
      <template v-else-if="last && state?.answered">
        <CircleCheckBig :size="16" :stroke-width="2.5" />Завершить
      </template>
      <template v-else>Дальше<ArrowRight :size="16" :stroke-width="2.5" /></template>
    </button>
  </BottomBar>
</template>
