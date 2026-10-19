<script setup lang="ts">
import { computed, nextTick, onMounted, reactive } from 'vue'
import { storeToRefs } from 'pinia'
import { RouterLink } from 'vue-router'
import { Check, ChevronDown } from 'lucide-vue-next'
import { useCourseStore } from '../stores/course'
import StatusBadge from '../components/StatusBadge.vue'
import SeedIcon from '../components/SeedIcon.vue'
import UnlockLevelModal from '../components/UnlockLevelModal.vue'
import { useWalletStore } from '../stores/wallet'
import { useUnlockModal } from '../lib/unlockModal'
import { useScript } from '../lib/useScript'
import { lastStudiedLesson } from '../lib/lastStudied'
import { levelPassed, levelProgress, readCollapsed, writeCollapsed } from '../lib/courseCollapse'
import type { LessonRef, Phase } from '../types'

const store = useCourseStore()
// Display only: lesson titles follow the learner's script.
const { show } = useScript()
const { course, loading, error } = storeToRefs(store)

const wallet = useWalletStore()
const unlock = useUnlockModal()

// The balance and the shop are what the unlock modal needs; the course itself
// does not wait for them.
onMounted(async () => {
  wallet.refresh()
  wallet.loadShop().catch(() => {})
  // A course remembered from an earlier visit can be out of date (a lesson started
  // since is not in it), and where to scroll depends on the statuses, so read it again.
  await (course.value ? store.refresh() : store.load())
  void scrollToLastStudied()
})

// Scrolls the course to the lesson the learner was last at (see lastStudiedLesson),
// once, when the screen opens. If that lesson sits in a folded level, the level stays
// folded (a finished level folds itself, and a fold is the learner's own choice) and
// the screen goes to the level's header instead.
async function scrollToLastStudied() {
  const c = course.value
  if (!c) return
  const target = lastStudiedLesson(c.lessons)
  const phase = target && c.phases.find((p) => p.lessons.includes(target.id))
  if (!target || !phase) return
  const selector = collapsed(phase) ? `[data-phase="${phase.id}"]` : `[data-lesson="${target.id}"]`
  await nextTick()
  // The cards appear after the loading placeholder has faded out.
  for (let i = 0; i < 40; i++) {
    const found = document.querySelector(selector)
    if (found) {
      found.scrollIntoView?.({ block: 'center' })
      return
    }
    await new Promise((r) => setTimeout(r, 50))
  }
}

const byId = computed(() => {
  const m = new Map<string, LessonRef>()
  course.value?.lessons.forEach((l) => m.set(l.id, l))
  return m
})

// Folded levels. The learner's own choice (remembered in this browser) wins; with
// none, a level folds once it is passed. `choices` is read from storage the first
// time a level is looked at, then kept here so a click re-renders at once.
const choices = reactive<Record<string, boolean | null>>({})
function choice(phase: Phase): boolean | null {
  if (!(phase.id in choices)) choices[phase.id] = readCollapsed(phase.id)
  return choices[phase.id]
}
function passed(phase: Phase): boolean {
  return levelPassed(phase, byId.value)
}
function collapsed(phase: Phase): boolean {
  return choice(phase) ?? passed(phase)
}
function setCollapsed(phase: Phase, value: boolean) {
  choices[phase.id] = value
  writeCollapsed(phase.id, value)
}
function toggle(phase: Phase) {
  setCollapsed(phase, !collapsed(phase))
}
// One control for the whole course: fold everything, or unfold it again.
const allCollapsed = computed(() => !!course.value?.phases.every(collapsed))
function toggleAll() {
  const fold = !allCollapsed.value
  course.value?.phases.forEach((p) => setCollapsed(p, fold))
}

// A level the learner has not bought. It looks like any other level; tapping a
// lesson in it asks about the price instead of opening the lesson. A locked
// level whose lessons are all still "скоро" has nothing to sell yet, so it
// stays quiet: no price, no tap target.
function forSale(phase: Phase): boolean {
  return !!phase.locked && phase.lessons.some((id) => !byId.value.get(id)?.planned)
}
function comingSoon(phase: Phase): boolean {
  return !!phase.locked && !forSale(phase)
}

// A bought or free level links to its lesson. A level for sale is a div that
// opens the unlock modal instead: a router link would navigate before any click
// handler of ours could stop it.
function cardAttrs(phase: Phase, lid: string) {
  if (forSale(phase)) return { role: 'button', tabindex: 0, 'data-test': `locked-${lid}` }
  if (comingSoon(phase)) return {}
  return { to: `/lesson/${lid}` }
}

function badgeClass(lesson?: LessonRef) {
  if (lesson?.status === 'done') return 'bg-[color-mix(in_srgb,var(--good)_16%,transparent)] text-[var(--good)]'
  if (lesson?.status === 'in_progress') return 'bg-[var(--accent)] text-white'
  return 'bg-[var(--bg-soft)] text-[var(--muted)]'
}
</script>

<template>
  <div class="mb-4 flex items-center justify-between gap-3">
    <h1 class="text-2xl font-extrabold">Курс</h1>
    <button
      v-if="course && course.phases.length > 1"
      type="button"
      class="rounded-full bg-[var(--bg-soft)] px-3 py-1 text-sm font-semibold text-[var(--muted)] transition hover:text-[var(--accent)]"
      data-test="toggle-all"
      @click="toggleAll"
    >
      {{ allCollapsed ? 'Развернуть всё' : 'Свернуть всё' }}
    </button>
  </div>

  <Transition name="fade" mode="out-in">
    <div v-if="loading" key="skel" class="space-y-8">
      <section v-for="s in 2" :key="s">
        <div class="skel mb-3 h-3.5 w-24"></div>
        <div class="space-y-3">
          <div v-for="r in 3" :key="r" class="card flex items-center gap-3.5 p-4">
            <div class="skel h-11 w-11 shrink-0 rounded-full"></div>
            <span class="min-w-0 flex-1 space-y-1.5">
              <span class="skel block h-4 w-3/5"></span>
              <span class="skel block h-3 w-4/5"></span>
            </span>
            <div class="skel h-6 w-14 shrink-0 rounded-full"></div>
          </div>
        </div>
      </section>
    </div>
    <p v-else-if="error" key="error" class="card p-4 text-[var(--bad)]">
      {{ error }} <button class="font-semibold text-[var(--accent)]" @click="store.load(true)">повторить</button>
    </p>

    <div v-else-if="course" key="course" class="space-y-8">
      <section v-for="phase in course.phases" :key="phase.id">
        <div class="flex items-center justify-between gap-3" :class="collapsed(phase) ? '' : 'mb-3'" :data-phase="phase.id">
          <h2 class="min-w-0 text-sm font-bold uppercase tracking-wide text-[var(--muted)]">
            <button
              type="button"
              class="flex max-w-full items-center gap-1.5 text-left uppercase"
              :aria-expanded="!collapsed(phase)"
              :aria-controls="`phase-${phase.id}`"
              :data-test="`toggle-${phase.id}`"
              @click="toggle(phase)"
            >
              <ChevronDown
                :size="18"
                :stroke-width="2.5"
                class="shrink-0 transition-transform"
                :class="collapsed(phase) ? '-rotate-90' : ''"
              />
              <span class="min-w-0">{{ phase.title }}</span>
            </button>
          </h2>
          <div class="flex shrink-0 items-center gap-3">
            <span
              v-if="passed(phase)"
              class="flex shrink-0 items-center gap-1 text-xs font-bold text-[var(--good)]"
              :data-test="`passed-${phase.id}`"
            >
              <Check :size="15" :stroke-width="3" /> пройден
            </span>
            <span
              v-else-if="collapsed(phase) && levelProgress(phase, byId).total"
              class="shrink-0 text-xs font-semibold tabular-nums text-[var(--muted)]"
              :data-test="`progress-${phase.id}`"
            >
              {{ levelProgress(phase, byId).done }} из {{ levelProgress(phase, byId).total }}
            </span>
            <button
              v-if="forSale(phase) && phase.price_effective"
              type="button"
              class="flex shrink-0 items-center gap-1.5 rounded-full bg-[var(--bg-soft)] px-3 py-1 text-sm font-bold tabular-nums"
              :data-test="`price-${phase.id}`"
              @click="unlock.openFor(phase.id)"
            >
              <SeedIcon :size="16" />
              {{ phase.price_effective }}
            </button>
          </div>
        </div>
        <div v-if="!collapsed(phase)" :id="`phase-${phase.id}`" class="space-y-3">
          <component
            :is="forSale(phase) || comingSoon(phase) ? 'div' : RouterLink"
            v-for="lid in phase.lessons"
            :key="lid"
            v-bind="cardAttrs(phase, lid)"
            :data-lesson="lid"
            class="card flex items-center gap-3.5 p-4 transition"
            :class="comingSoon(phase) ? '' : 'cursor-pointer active:scale-[0.99] hover:-translate-y-0.5'"
            @click="forSale(phase) && unlock.openFor(phase.id)"
            @keydown.enter="forSale(phase) && unlock.openFor(phase.id)"
          >
            <span
              class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-sm font-bold tabular-nums"
              :class="badgeClass(byId.get(lid))"
            >
              {{ lid }}
            </span>
            <span class="min-w-0 flex-1">
              <span class="block text-sm font-semibold leading-snug sm:text-base">{{ show(byId.get(lid)?.title || 'Урок ' + lid) }}</span>
              <span class="block truncate text-xs text-[var(--muted)] sm:text-sm">{{ show(byId.get(lid)?.subtitle) }}</span>
            </span>
            <StatusBadge
              v-if="byId.get(lid)"
              :status="byId.get(lid)!.status"
              :planned="byId.get(lid)!.planned"
            />
          </component>
        </div>
      </section>
    </div>
  </Transition>

  <UnlockLevelModal />
</template>
