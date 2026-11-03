<script setup lang="ts">
// The little slides before a story, each showing what the next part will look like: the picture, the
// subtitle, the sound. They are drawn with the story's own first frame and sentence, so what is shown is what comes.
// Everything loops with CSS only; with reduced motion the slide stands still in its finished state.
import { computed } from 'vue'
import { useScript } from '../lib/useScript'

const props = defineProps<{
  kind: 'listen' | 'translate' | 'fast' | 'scrub'
  image: string
  sr: string
  ru: string
  /** Three Serbian lines from the beginning, the middle and the end of the story, for the slide about the bar. */
  lines?: string[]
  /** A quick question to play out in the third slide. */
  quiz: { mask: string; options: string[] }
}>()
const { show } = useScript()

const words = computed(() => props.sr.split(/\s+/).filter(Boolean))
const blanked = computed(() => {
  const target = props.quiz.mask.toLowerCase()
  return words.value.map((w) => (w.toLowerCase().replace(/[^\p{L}]/gu, '') === target ? '_____' : w))
})
</script>

<template>
  <div class="slide" :class="`slide-${kind}`" data-test="story-slide">
    <div class="pic">
      <img :src="image" alt="" draggable="false" />
      <span v-if="kind === 'listen' || kind === 'fast'" class="eq" aria-hidden="true"><i /><i /><i /><i /><i /></span>
      <div v-if="kind === 'scrub'" class="mock" aria-hidden="true">
        <div class="row"><i /><i /><i /><i /></div>
        <div class="row fill"><i /><i /><i /><i /></div>
        <span class="finger"><b /></span>
      </div>
    </div>

    <div class="lines">
      <template v-if="kind === 'listen'">
        <p class="sr serbian">
          <template v-for="(w, i) in words" :key="i"><span class="w" :style="{ '--i': i, '--n': words.length }">{{ show(w) }}</span>{{ ' ' }}</template>
        </p>
      </template>

      <template v-else-if="kind === 'translate'">
        <p class="sr serbian line-a">{{ show(sr) }}</p>
        <p class="ru line-b">{{ ru }}</p>
      </template>

      <template v-else-if="kind === 'scrub'">
        <p v-for="(l, i) in (lines ?? []).slice(0, 3)" :key="i" class="sr serbian scrub-line" :class="`scrub-line-${i}`">{{ show(l) }}</p>
      </template>

      <template v-else>
        <p class="sr serbian">
          <template v-for="(w, i) in blanked" :key="i"><span :class="{ blank: w === '_____' }">{{ w === '_____' ? w : show(w) }}</span>{{ ' ' }}</template>
        </p>
        <p class="ru-small">{{ ru }}</p>
        <div class="opts" aria-hidden="true">
          <span v-for="(o, i) in quiz.options" :key="o" class="opt serbian" :class="{ right: o.toLowerCase() === quiz.mask.toLowerCase() }" :style="{ '--i': i }">{{ show(o) }}</span>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.slide {
  --loop: 6s;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.pic {
  position: relative;
  flex: 0 1 auto;
  min-height: 4rem;
  width: 100%;
  aspect-ratio: 4 / 3;
  overflow: hidden;
  border-radius: 1rem;
  background: var(--bg-soft);
}
.pic img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  image-rendering: pixelated;
  animation: pic-zoom var(--loop) ease-in-out infinite alternate;
}
@keyframes pic-zoom {
  to {
    transform: scale(1.07);
  }
}

/* The sound: five bars that breathe. */
.eq {
  position: absolute;
  left: 0.75rem;
  bottom: 0.75rem;
  display: flex;
  align-items: flex-end;
  gap: 3px;
  height: 1.5rem;
  padding: 0.35rem 0.55rem;
  border-radius: 999px;
  background: color-mix(in srgb, var(--card) 80%, transparent);
  backdrop-filter: blur(6px);
  box-sizing: content-box;
}
.eq i {
  display: block;
  width: 4px;
  height: 100%;
  border-radius: 2px;
  background: var(--accent);
  transform-origin: bottom;
  animation: eq 0.9s ease-in-out infinite;
}
.eq i:nth-child(2) { animation-delay: -0.2s; }
.eq i:nth-child(3) { animation-delay: -0.5s; }
.eq i:nth-child(4) { animation-delay: -0.1s; }
.eq i:nth-child(5) { animation-delay: -0.4s; }
.slide-fast .eq i {
  animation-duration: 0.55s;
}
@keyframes eq {
  0%, 100% { transform: scaleY(0.25); }
  50% { transform: scaleY(1); }
}


.lines {
  flex-shrink: 0;
  min-height: 6.5rem;
  padding-top: 1rem;
  text-align: center;
}
.sr {
  font-size: 1.25rem;
  font-weight: 800;
  line-height: 1.3;
}
.ru-small {
  margin-top: 0.4rem;
  font-size: 0.85rem;
  color: var(--muted);
}
.ru {
  margin-top: 0.4rem;
  font-size: 1.1rem;
  font-weight: 700;
}

/* Slide 1: the word that is being spoken lights up, one after another. */
.w {
  border-radius: 0.35rem;
  padding: 0 0.15em;
  animation: word-on var(--loop) linear infinite;
  animation-delay: calc(var(--i) * (var(--loop) / (var(--n) + 1)));
}
@keyframes word-on {
  0%, 24% { background: var(--accent-soft); color: var(--accent); }
  25%, 100% { background: transparent; color: inherit; }
}

/* Slide 2: the Serbian line, then its Russian one, taking turns. */
.line-a {
  animation: turn-a var(--loop) ease-in-out infinite;
}
.line-b {
  animation: turn-b var(--loop) ease-in-out infinite;
}
@keyframes turn-a {
  0%, 30% { opacity: 1; }
  42%, 92% { opacity: 0.4; }
  100% { opacity: 1; }
}
@keyframes turn-b {
  0%, 28% { opacity: 0; transform: translateY(6px); }
  42%, 92% { opacity: 1; transform: none; }
  100% { opacity: 0; transform: translateY(6px); }
}

/* Slide 3: the blank, and the options, one of which is found. */
.blank {
  animation: blank-fill var(--loop) ease-in-out infinite;
  border-radius: 0.35rem;
  padding: 0 0.15em;
}
@keyframes blank-fill {
  0%, 60% { background: transparent; }
  75%, 95% { background: color-mix(in srgb, var(--good) 22%, transparent); }
  100% { background: transparent; }
}
.opts {
  display: flex;
  justify-content: center;
  gap: 0.5rem;
  margin-top: 0.9rem;
}
.opt {
  border-radius: 1rem;
  background: var(--bg-soft);
  padding: 0.45rem 0.9rem;
  font-size: 0.9rem;
  font-weight: 600;
  animation: opt-in 0.5s ease-out both;
  animation-delay: calc(var(--i) * 0.12s);
}
.opt.right {
  animation: opt-in 0.5s ease-out both, opt-right var(--loop) ease-in-out infinite;
  animation-delay: calc(var(--i) * 0.12s), 0s;
}
@keyframes opt-in {
  from { opacity: 0; transform: translateY(8px); }
}
@keyframes opt-right {
  0%, 55% { box-shadow: none; }
  70%, 95% { box-shadow: 0 0 0 2px var(--good); }
  100% { box-shadow: none; }
}

/* Slide 4: a finger along the bar, and the line under it follows. */
.mock {
  position: absolute;
  left: 0.75rem;
  right: 0.75rem;
  top: 0.9rem;
  height: 0.5rem;
}
.mock .row {
  position: absolute;
  inset: 0;
  display: flex;
  gap: 2px;
}
.mock .row i {
  display: block;
  flex: 1;
  height: 100%;
  border-radius: 999px;
  background: color-mix(in srgb, var(--card) 70%, transparent);
}
.mock .fill i {
  background: var(--accent);
}
.mock .fill {
  animation: mock-fill var(--loop) ease-in-out infinite;
}
@keyframes mock-fill {
  0%, 8% { clip-path: inset(0 88% 0 0); }
  28%, 42% { clip-path: inset(0 52% 0 0); }
  62%, 78% { clip-path: inset(0 12% 0 0); }
  92%, 100% { clip-path: inset(0 88% 0 0); }
}
.finger {
  position: absolute;
  top: 50%;
  width: 1.7rem;
  height: 1.7rem;
  margin: -0.85rem 0 0 -0.85rem;
  border-radius: 999px;
  background: color-mix(in srgb, var(--card) 85%, transparent);
  box-shadow: 0 0 0 2px var(--accent);
  animation: finger-move var(--loop) ease-in-out infinite;
}
.finger b {
  position: absolute;
  inset: 0.5rem;
  border-radius: 999px;
  background: var(--accent);
}
@keyframes finger-move {
  0%, 8% { left: 12%; transform: scale(1); }
  4% { transform: scale(0.8); }
  28%, 42% { left: 48%; }
  62%, 78% { left: 88%; }
  92%, 100% { left: 12%; }
}
.scrub-line {
  position: absolute;
  left: 0;
  right: 0;
  opacity: 0;
  animation: scrub-line var(--loop) ease-in-out infinite;
}
.scrub-line-1 { animation-delay: calc(var(--loop) * 0.28); }
.scrub-line-2 { animation-delay: calc(var(--loop) * 0.6); }
@keyframes scrub-line {
  0% { opacity: 1; }
  24% { opacity: 1; }
  30%, 100% { opacity: 0; }
}
.slide-scrub .lines {
  position: relative;
}

@media (prefers-reduced-motion: reduce) {
  .mock .fill, .finger, .scrub-line,
  .pic img, .eq i, .w, .line-a, .line-b, .blank, .opt, .opt.right {
    animation: none;
  }
  .eq i { transform: scaleY(0.7); }
  .mock .fill { clip-path: inset(0 52% 0 0); }
  .finger { left: 48%; }
  .scrub-line-1 { opacity: 1; }
  .line-b { opacity: 1; transform: none; }
  .opt.right { box-shadow: 0 0 0 2px var(--good); }
}
</style>
