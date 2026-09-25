<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import type { FalseFriend, Vocab } from '../types'
import WordMedia from '../components/WordMedia.vue'
import SpeakButton from '../components/SpeakButton.vue'
import SerbianKeys from '../components/SerbianKeys.vue'
import FieldSelect from '../components/FieldSelect.vue'

const route = useRoute()
const router = useRouter()

const TABS = [
  { v: 'words' as const, label: 'Слова' },
  { v: 'traps' as const, label: 'Ловушки' },
]
const GROUPS = [
  { value: 'all', label: 'все' },
  { value: 'top', label: 'бытовые' },
  { value: 'shop', label: 'магазин / еда' },
  { value: 'small', label: 'мелкие частые' },
]

const tab = ref<'words' | 'traps'>(route.query.tab === 'traps' ? 'traps' : 'words')
const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const lesson = ref('')
const tagFilter = ref('')
const group = ref('all')
const scriptMode = ref<'latin' | 'cyrillic'>('latin')
const error = ref<string | null>(null)

const allVocab = ref<Vocab[]>([])
const allTraps = ref<FalseFriend[]>([])
const initialLoading = ref(true)

watch(tab, (v) => {
  router.replace({ query: { ...route.query, tab: v === 'traps' ? 'traps' : undefined } })
  drilling.value = false
})

let t: ReturnType<typeof setTimeout>
watch(q, () => {
  clearTimeout(t)
  t = setTimeout(load, 200)
})

async function load() {
  try {
    ;[allVocab.value, allTraps.value] = await Promise.all([
      api.vocab({ q: q.value || undefined }),
      api.falseFriends({ q: q.value || undefined }),
    ])
    error.value = null
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    initialLoading.value = false
  }
}
onMounted(load)

const lessons = computed(() => [...new Set(allVocab.value.map((v) => v.lesson).filter(Boolean))].sort())
const tags = computed(() => [...new Set(allVocab.value.flatMap((v) => v.tags ?? []))].sort())
const lessonOptions = computed(() => [
  { value: '', label: 'урок: все' },
  ...lessons.value.map((l) => ({ value: l as string, label: `урок ${l}` })),
])
const tagOptions = computed(() => [
  { value: '', label: 'тег: все' },
  ...tags.value.map((tg) => ({ value: tg, label: tg })),
])

const vocabRows = computed(() =>
  allVocab.value.filter(
    (v) =>
      (!lesson.value || v.lesson === lesson.value) && (!tagFilter.value || (v.tags ?? []).includes(tagFilter.value)),
  ),
)
const trapRows = computed(() => allTraps.value.filter((f) => group.value === 'all' || f.group === group.value))

const drilling = ref(false)
const drillIdx = ref(0)
const drillShown = ref(false)
function startDrill() {
  drilling.value = true
  drillIdx.value = 0
  drillShown.value = false
}
function nextCard() {
  drillShown.value = false
  drillIdx.value++
  if (drillIdx.value >= vocabRows.value.length) drilling.value = false
}
</script>

<template>
  <div class="mb-4 flex items-baseline justify-between">
    <h1 class="text-2xl font-extrabold">Словарь</h1>
    <Transition name="fade">
      <button
        v-if="tab === 'words'"
        class="btn btn-ghost px-2.5 py-1 text-sm"
        @click="scriptMode = scriptMode === 'latin' ? 'cyrillic' : 'latin'"
      >
        {{ scriptMode === 'latin' ? 'латиница' : 'кириллица' }}
      </button>
    </Transition>
  </div>

  <div class="mb-3 flex gap-1 rounded-2xl bg-[var(--bg-soft)] p-1">
    <button
      v-for="tb in TABS"
      :key="tb.v"
      type="button"
      class="flex-1 rounded-xl px-3 py-1.5 text-sm font-semibold transition"
      :class="tab === tb.v ? 'bg-[var(--accent)] text-white' : 'text-[var(--muted)]'"
      @click="tab = tb.v"
    >
      {{ tb.label }}
      <span class="ml-1 text-xs opacity-75">{{ tb.v === 'words' ? vocabRows.length : trapRows.length }}</span>
    </button>
  </div>

  <p v-if="error" class="card mb-3 p-3 text-[var(--bad)]">{{ error }}</p>

  <div class="mb-2 flex flex-col gap-2 sm:flex-row sm:items-center">
    <input v-model="q" placeholder="поиск…" class="field w-full sm:flex-1" />
    <Transition name="fade" mode="out-in">
      <div :key="tab" class="flex flex-row gap-2">
        <template v-if="tab === 'words'">
          <FieldSelect v-model="lesson" :options="lessonOptions" class="w-full sm:w-auto flex-1" />
          <FieldSelect v-model="tagFilter" :options="tagOptions" class="w-full sm:w-auto flex-1" />
        </template>
        <FieldSelect v-else v-model="group" :options="GROUPS" class="w-full sm:w-auto" />
      </div>
    </Transition>
  </div>
  <SerbianKeys class="mb-3" />

  <div v-if="initialLoading" class="space-y-2">
    <div v-for="r in 6" :key="r" class="card flex gap-3 p-3">
      <div class="skel h-12 w-12 shrink-0 rounded-xl"></div>
      <div class="min-w-0 flex-1 space-y-1.5">
        <div class="skel h-5 w-2/5"></div>
        <div class="skel h-3.5 w-3/5"></div>
      </div>
    </div>
  </div>

  <Transition v-else name="fade" mode="out-in">
    <div :key="tab">
      <template v-if="tab === 'words'">
        <div class="mb-3 flex items-center gap-3 text-sm text-[var(--muted)]">
          <span>{{ vocabRows.length }} слов</span>
          <button v-if="vocabRows.length" class="font-semibold text-[var(--accent)]" @click="startDrill">
            Учить выборку →
          </button>
        </div>

        <div v-if="drilling && vocabRows[drillIdx]" class="card mb-4 p-6 text-center" @click="drillShown = true">
          <p class="text-xs text-[var(--muted)]">{{ drillIdx + 1 }} / {{ vocabRows.length }}</p>
          <div class="mt-2 flex items-center justify-center gap-2">
            <p class="serbian text-3xl font-semibold">
              {{ scriptMode === 'latin' ? vocabRows[drillIdx].latin : vocabRows[drillIdx].cyrillic }}
            </p>
            <SpeakButton :src="vocabRows[drillIdx].audio" :size="34" @click.stop />
          </div>
          <template v-if="drillShown">
            <div class="mt-3 flex justify-center">
              <WordMedia :image="vocabRows[drillIdx].image" :emoji="vocabRows[drillIdx].emoji" :size="112" />
            </div>
            <p class="mt-2 text-lg">{{ vocabRows[drillIdx].ru }}</p>
            <p v-if="vocabRows[drillIdx].note" class="text-sm text-[var(--muted)]">{{ vocabRows[drillIdx].note }}</p>
            <button class="btn btn-primary mt-3" @click.stop="nextCard">Дальше</button>
          </template>
          <p v-else class="mt-3 text-xs text-[var(--muted)]">нажми, чтобы показать перевод</p>
        </div>

        <div class="space-y-2">
          <div v-for="v in vocabRows" :key="v.id" class="card flex gap-3 p-3">
            <WordMedia :image="v.image" :emoji="v.emoji" :alt="v.ru" :size="48" class="mt-0.5 shrink-0" />
            <div class="min-w-0 flex-1">
              <div class="flex items-baseline justify-between gap-2">
                <span class="flex items-center gap-1.5">
                  <span class="serbian text-lg font-semibold">{{
                    scriptMode === 'latin' ? v.latin : v.cyrillic
                  }}</span>
                  <span v-if="v.transcription" class="text-sm text-[var(--muted)]">[{{ v.transcription }}]</span>
                  <SpeakButton :src="v.audio" :size="22" />
                </span>
                <span v-if="v.lesson" class="shrink-0 text-sm text-[var(--muted)]">урок {{ v.lesson }}</span>
              </div>
              <p class="text-sm">{{ v.ru }}</p>
              <p v-if="v.note" class="mt-0.5 text-sm text-[var(--muted)]">{{ v.note }}</p>
            </div>
          </div>
        </div>
      </template>

      <template v-else>
        <p class="mb-3 text-sm text-[var(--muted)]">Слова, которые звучат знакомо, но значат другое.</p>
        <p class="mb-2 text-sm text-[var(--muted)]">{{ trapRows.length }} записей</p>

        <div class="space-y-2">
          <div v-for="f in trapRows" :key="f.id" class="card flex gap-3 p-3">
            <WordMedia :image="f.image" :emoji="f.emoji" :alt="f.means" :size="48" class="mt-0.5 shrink-0" />
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-baseline justify-between gap-x-2 gap-y-0.5">
                <span class="serbian text-lg font-semibold">{{ f.sr }}</span>
                <span v-if="f.not" class="text-sm text-[var(--bad)]">≠ {{ f.not }}</span>
              </div>
              <p class="text-sm">{{ f.means }}</p>
              <p v-if="f.correct" class="mt-0.5 text-sm text-[var(--muted)]">
                «то самое» → <span class="serbian">{{ f.correct }}</span>
              </p>
            </div>
          </div>
        </div>
      </template>
    </div>
  </Transition>
</template>
