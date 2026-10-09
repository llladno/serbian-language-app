<script setup lang="ts">
// A half-circle gauge that is drawn, not filled: the stroke is laid down along
// the arc while the number in the middle counts up in step. One clock drives
// both, so they cannot drift apart, and the same clock stops them together on
// the final value.
//
// What happens round the arc depends on how it ended: a full one glitters
// (small sparkles behind the stroke), one in the eighties lets a few sparkles
// float up from it, anything less is left plain.
//
// Motion is skipped (final frame at once) for reduced motion, a hidden tab —
// requestAnimationFrame does not run there, and a count started in one freezes
// halfway — and a duration of 0, which is what the tests use.
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import SparkleBurst from './SparkleBurst.vue'

const props = withDefaults(
  defineProps<{
    percent: number
    // Gold when the arc is full, the accent otherwise: gold is the colour of
    // the seeds, so a full arc reads as the same kind of reward.
    gold?: boolean
    // Wait before the stroke starts, so the screen has settled first.
    delay?: number
    // 0 = no animation. Otherwise a base the stroke's length scales from.
    speed?: number
    // Sparkles round the arc when it lands (see above).
    sparkle?: boolean
  }>(),
  { gold: false, delay: 450, speed: 1, sparkle: true },
)
const emit = defineEmits<{ landed: [] }>()

const W = 280
const CX = 140
const CY = 140
const R = 112
const target = computed(() => Math.max(0, Math.min(100, Math.round(props.percent))))

const shown = ref(0) // 0..100, what is drawn right now
const landed = ref(false)

const arc = `M ${CX - R} ${CY} A ${R} ${R} 0 0 1 ${CX + R} ${CY}`

// Squared, so the glow blooms late in the stroke rather than lighting the whole
// arc dimly from the start.
const HALO = 0.8
const haloOpacity = computed(() => {
  const t = target.value ? Math.min(1, shown.value / target.value) : 0
  return HALO * t * t
})

let raf = 0
let timer = 0

const reduced = () =>
  typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
const instant = () =>
  props.speed <= 0 || typeof requestAnimationFrame !== 'function' || document.hidden || reduced()

function land() {
  shown.value = target.value
  landed.value = true
  emit('landed')
}

// Fast out of the gate and slow into the end, like a pen finishing a stroke.
const ease = (t: number) => 1 - Math.pow(1 - t, 3.2)

function run() {
  if (target.value === 0 || instant()) {
    land()
    return
  }
  // Longer arcs take longer, but not in proportion: a 30% stroke that took a
  // third of the time of a 100% one would look hurried.
  const ms = (900 + target.value * 14) / props.speed
  let started = 0
  const frame = (now: number) => {
    if (!started) started = now
    const t = Math.min(1, (now - started) / ms)
    shown.value = target.value * ease(t)
    if (t < 1) raf = requestAnimationFrame(frame)
    else land()
  }
  raf = requestAnimationFrame(frame)
}

onMounted(() => {
  if (instant() || props.delay <= 0) run()
  else timer = window.setTimeout(run, props.delay)
})
onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  clearTimeout(timer)
})

// The layer the sparkles live in is the gauge plus margins: below for the
// twinkle to spill over the sides, above for the rising ones to climb into.
const PAD_X = 40
const PAD_TOP = 70
const BOX_W = W + PAD_X * 2
const BOX_H = 160 + PAD_TOP

// Where a point at angle `theta` (0 = left end of the arc, 1 = right end) and
// distance `r` from the centre sits in that layer, in percent.
function spotAt(frac: number, r: number) {
  const theta = Math.PI * (1 - frac)
  const x = CX + r * Math.cos(theta) + PAD_X
  const y = CY - r * Math.sin(theta) + PAD_TOP
  return { left: (x / BOX_W) * 100, top: (y / BOX_H) * 100 }
}

type Sparkles = { mode: 'twinkle' | 'rise'; spots: { left: number; top: number }[] }
const sparkles = computed<Sparkles | null>(() => {
  if (!landed.value || !props.sparkle) return null
  if (target.value >= 100) {
    // Hugging the outer edge of the stroke, so each one is half-hidden by it.
    const n = 12
    return {
      mode: 'twinkle',
      spots: Array.from({ length: n }, (_, i) =>
        spotAt((i + 0.5 + (Math.random() - 0.5) * 0.5) / n, R + 6 + Math.random() * 22),
      ),
    }
  }
  if (target.value >= 80) {
    // Five, spread along the arc, starting on the stroke itself so they come
    // out from behind it.
    return {
      mode: 'rise',
      spots: [0.12, 0.31, 0.5, 0.69, 0.88].map((f) => spotAt(f + (Math.random() - 0.5) * 0.08, R)),
    }
  }
  return null
})
</script>

<template>
  <div
    class="gauge"
    :class="{ gold, landed }"
    role="img"
    :aria-label="`С первой попытки верно: ${target}%`"
  >
    <!-- Behind the arc: the stroke covers the part of a sparkle that sits on it. -->
    <div v-if="sparkles" class="burst-box">
      <SparkleBurst
        :spots="sparkles.spots"
        :mode="sparkles.mode"
        :base="2"
        :lift="46"
        :sizes="sparkles.mode === 'rise' ? ['m', 'm', 'l'] : ['s', 's', 's', 'm', 'm', 'l']"
        :colors="['gold', 'amber']"
      />
    </div>

    <svg :viewBox="`0 0 ${W} 160`" :width="W" height="160" class="svg" aria-hidden="true">
      <defs>
        <linearGradient id="gauge-gold" x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" style="stop-color: var(--seed)" />
          <stop offset="1" style="stop-color: var(--seed-hi)" />
        </linearGradient>
        <!-- The glow's blur is an SVG filter, not CSS `filter`: CSS filters on
             SVG shapes are not drawn everywhere (Safari), SVG ones are. -->
        <filter id="gauge-blur" x="-20%" y="-35%" width="140%" height="190%">
          <feGaussianBlur stdDeviation="7" />
        </filter>
      </defs>
      <path :d="arc" class="track" pathLength="100" />
      <!-- The glow of a full arc: a blurred copy of the stroke behind it, drawn
           by the same clock, so it swells with the arc and is at full strength
           exactly when the arc is — there is no moment at which it "switches on". -->
      <path
        v-if="gold"
        v-show="shown > 0.4"
        :d="arc"
        class="halo"
        pathLength="100"
        :stroke-dasharray="`${shown} 100`"
        :style="{ opacity: haloOpacity }"
        filter="url(#gauge-blur)"
      />
      <path
        v-show="shown > 0.4"
        :d="arc"
        class="fill"
        pathLength="100"
        :stroke-dasharray="`${shown} 100`"
      />
    </svg>

    <div class="readout" aria-hidden="true">
      <p class="num serbian">{{ Math.round(shown) }}<span class="pct">%</span></p>
      <p class="cap">с первой попытки</p>
    </div>
  </div>
</template>

<style scoped>
.gauge {
  position: relative;
  width: 280px;
  height: 160px;
  /* Horizontal only: the vertical room is the parent's to give, and a margin
     set here would beat its utility classes. */
  margin-left: auto;
  margin-right: auto;
}
.svg {
  position: relative;
  z-index: 1;
  display: block;
  overflow: visible;
}
.track,
.fill {
  fill: none;
  stroke-width: 20;
  stroke-linecap: round;
}
.track {
  stroke: var(--ring-track);
}
.fill {
  stroke: var(--accent);
}
.gold .fill {
  stroke: url(#gauge-gold);
}
.halo {
  fill: none;
  stroke: url(#gauge-gold);
  stroke-width: 26;
  stroke-linecap: round;
}

.burst-box {
  position: absolute;
  inset: -70px -40px 0;
  z-index: 0;
  pointer-events: none;
}

/* Inside the arc, sitting on its baseline. */
.readout {
  position: absolute;
  left: 50%;
  bottom: 2px;
  width: 180px;
  transform: translateX(-50%);
  z-index: 2;
  text-align: center;
}
.num {
  font-size: 4.25rem;
  font-weight: 600;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
  color: var(--fg);
}
.gold .num {
  color: color-mix(in srgb, var(--seed) 55%, var(--fg));
}
.pct {
  margin-left: 0.1em;
  font-size: 0.45em;
  font-weight: 500;
  color: var(--muted);
  letter-spacing: 0;
}
.cap {
  margin-top: 0.35rem;
  font-size: 0.8rem;
  color: var(--muted);
}

</style>
