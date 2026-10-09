<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { RouterLink } from 'vue-router'
import { useCourseStore } from '../stores/course'
import StatusBadge from '../components/StatusBadge.vue'
import SeedIcon from '../components/SeedIcon.vue'
import UnlockLevelModal from '../components/UnlockLevelModal.vue'
import { useWalletStore } from '../stores/wallet'
import { useUnlockModal } from '../lib/unlockModal'
import type { LessonRef, Phase } from '../types'

const store = useCourseStore()
const { course, loading, error } = storeToRefs(store)

const wallet = useWalletStore()
const unlock = useUnlockModal()

// The balance and the shop are what the unlock modal needs; the course itself
// does not wait for them.
onMounted(() => {
  store.load()
  wallet.refresh()
  wallet.loadShop().catch(() => {})
})

const byId = computed(() => {
  const m = new Map<string, LessonRef>()
  course.value?.lessons.forEach((l) => m.set(l.id, l))
  return m
})

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
  <h1 class="mb-4 text-2xl font-extrabold">Курс</h1>

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
        <div class="mb-3 flex items-center justify-between gap-3">
          <h2 class="text-sm font-bold uppercase tracking-wide text-[var(--muted)]">{{ phase.title }}</h2>
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
        <div class="space-y-3">
          <component
            :is="forSale(phase) || comingSoon(phase) ? 'div' : RouterLink"
            v-for="lid in phase.lessons"
            :key="lid"
            v-bind="cardAttrs(phase, lid)"
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
              <span class="block text-sm font-semibold leading-snug sm:text-base">{{ byId.get(lid)?.title || 'Урок ' + lid }}</span>
              <span class="block truncate text-xs text-[var(--muted)] sm:text-sm">{{ byId.get(lid)?.subtitle }}</span>
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
