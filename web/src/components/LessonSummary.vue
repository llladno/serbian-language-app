<script setup lang="ts">
// The screen that closes a lesson. One sequence, played once: the arc is drawn
// (ArcGauge), and only when it is done does the rest arrive — the line of
// encouragement, then the figures counting up, then the way on. Everything is
// held back until then so the eye has a single place to be at a time.
//
// The figures are first-try, not final: a step cannot be finished with an
// exercise still wrong, so "how many did you get right in the end" is always
// all of them and would say nothing.
import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useWalletStore } from '../stores/wallet'
import { cheerFor } from '../lib/cheers'
import { SEED } from '../lib/seeds'
import { pluralRu } from '../lib/plural'
import type { LessonRef, LessonStats } from '../types'
import ArcGauge from './ArcGauge.vue'
import AnimatedNumber from './AnimatedNumber.vue'
import SparkleBurst from './SparkleBurst.vue'

const props = defineProps<{
  title: string
  reward: number
  stats?: LessonStats
  next: LessonRef | null
}>()
const wallet = useWalletStore()

// A lesson with nothing to answer has no percentage to draw an arc for.
const percent = computed(() => (props.stats && props.stats.answered > 0 ? props.stats.percent : null))
const cheer = computed(() => (percent.value === null ? '' : cheerFor(percent.value)))

// What to count, in this order. A figure that would read "0" is left out,
// except mistakes: "0 ошибок" is the best thing the screen can say.
const tiles = computed(() => {
  const out: { key: 'reward' | 'words' | 'mistakes'; value: number; label: (n: number) => string; gold?: boolean }[] = []
  if (props.reward > 0) {
    out.push({ key: 'reward', value: props.reward, label: (n) => wallet.currencyWord(n), gold: true })
  }
  if ((props.stats?.new_words ?? 0) > 0) {
    out.push({
      key: 'words',
      value: props.stats!.new_words,
      label: (n) => pluralRu(n, 'новое слово', 'новых слова', 'новых слов'),
    })
  }
  if (percent.value !== null) {
    out.push({
      key: 'mistakes',
      value: props.stats!.mistakes,
      label: (n) => pluralRu(n, 'ошибка', 'ошибки', 'ошибок'),
    })
  }
  return out
})

const revealed = ref(false)
const counts = reactive({ reward: 0, words: 0, mistakes: 0 })
const timers: number[] = []

function reveal() {
  if (revealed.value) return
  revealed.value = true
  // The figures count one after another, each just as its tile appears. For
  // someone who asked for less motion, or in a tab nobody is looking at, they
  // are simply there.
  const calm =
    (typeof document !== 'undefined' && document.hidden) ||
    !!window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
  tiles.value.forEach((t, i) => {
    if (calm) counts[t.key] = t.value
    else timers.push(window.setTimeout(() => (counts[t.key] = t.value), 380 + i * 160))
  })
}
onBeforeUnmount(() => timers.forEach(clearTimeout))

// With no arc to wait for, the sequence starts by itself.
if (percent.value === null) timers.push(window.setTimeout(reveal, 500))
</script>

<template>
  <section class="summary" :class="{ revealed }" data-test="lesson-summary">
    <header class="head">
      <h1 class="text-[1.375rem] font-extrabold leading-tight">Урок пройден</h1>
      <p class="mt-1 text-sm text-[var(--muted)]">{{ title }}</p>
    </header>

    <!-- Room above the arc: the sparkles that float up from it need it. -->
    <ArcGauge
      v-if="percent !== null"
      class="mt-12"
      :percent="percent"
      :gold="percent >= 100"
      @landed="reveal"
    />
    <div v-else class="relative mx-auto mt-6 h-40 w-64">
      <SparkleBurst v-if="revealed" :count="8" />
      <img :src="SEED" alt="" aria-hidden="true" class="lone" />
    </div>

    <p v-if="cheer" class="cheer" data-test="summary-cheer">{{ cheer }}</p>

    <dl v-if="tiles.length" class="plate stats" data-test="summary-stats">
      <div
        v-for="(t, i) in tiles"
        :key="t.key"
        class="tile"
        :class="{ gold: t.gold }"
        :style="{ '--i': i }"
        :data-test="'stat-' + t.key"
      >
        <dd class="serbian val">
          <AnimatedNumber :value="counts[t.key]" :duration="900" />
        </dd>
        <dt class="lab">{{ t.label(t.value) }}</dt>
      </div>
    </dl>

    <div class="actions">
      <RouterLink
        v-if="next"
        :to="`/lesson/${next.id}`"
        class="btn btn-primary w-full"
        data-test="summary-next"
      >
        Следующий урок
      </RouterLink>
      <RouterLink
        to="/course"
        class="btn w-full"
        :class="next ? 'btn-ghost' : 'btn-primary'"
        data-test="summary-course"
      >
        К курсу
      </RouterLink>
    </div>
  </section>
</template>

<style scoped>
.summary {
  width: 100%;
  max-width: 24rem;
  margin: 0 auto;
  text-align: center;
}

/* The header is there from the first frame; everything after the arc waits. */
.head {
  animation: rise 0.5s cubic-bezier(0.2, 0.7, 0.2, 1) both;
}

.cheer,
.stats,
.actions {
  opacity: 0;
  transform: translateY(14px);
  transition: opacity 0.55s ease, transform 0.55s cubic-bezier(0.2, 0.7, 0.2, 1);
}
.revealed .cheer,
.revealed .stats,
.revealed .actions {
  opacity: 1;
  transform: none;
}
.revealed .stats {
  transition-delay: 0.18s;
}
.revealed .actions {
  transition-delay: 0.7s;
}

.cheer {
  margin: 1.5rem auto 0;
  max-width: 20rem;
  font-size: 1.1875rem;
  font-weight: 500;
  line-height: 1.45;
}

/* One plate, three columns: the figures belong together, so they are not
   three cards. Gold is only the seeds' — it is the money. */
.stats {
  display: flex;
  margin-top: 1.5rem;
  padding: 0.5rem 0;
  border-radius: 20px;
  background: var(--card);
  box-shadow: var(--shadow);
}
.tile {
  flex: 1 1 0;
  padding: 0.85rem 0.5rem;
  opacity: 0;
  transform: scale(0.94);
  transition: opacity 0.45s ease, transform 0.5s cubic-bezier(0.3, 1.5, 0.5, 1);
  transition-delay: calc(0.3s + var(--i) * 0.16s);
}
.revealed .tile {
  opacity: 1;
  transform: none;
}
.tile + .tile {
  border-left: 1px solid var(--border);
}
.val {
  font-size: 2rem;
  font-weight: 600;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}
.tile.gold .val {
  color: color-mix(in srgb, var(--seed) 60%, var(--fg));
}
.lab {
  margin-top: 0.45rem;
  font-size: 0.8125rem;
  color: var(--muted);
}

.actions {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 1.5rem;
}

.lone {
  position: absolute;
  left: 50%;
  top: 50%;
  height: 130px;
  transform: translate(-50%, -50%);
  image-rendering: pixelated;
}

@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
}

@media (prefers-reduced-motion: reduce) {
  .head {
    animation: none;
  }
  .cheer,
  .stats,
  .actions,
  .tile {
    transition: none;
  }
}
</style>
