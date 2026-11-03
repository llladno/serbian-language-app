<script setup lang="ts">
// The two subtitle lines under the picture. The Serbian one lights the word being spoken; while a quick question is
// open the answer is blanked out, so the learner recalls it instead of reading it.
import { computed } from 'vue'
import { useScript } from '../lib/useScript'
import { maskAnswer, type Subtitle } from '../lib/storyPlayer'
import StoryGap from './StoryGap.vue'

const props = defineProps<{
  subtitle: Subtitle
  wordIdx: number
  /** Blank this part of the Serbian line (the open question's mask), or null. */
  mask: string | null
  /** The question was answered right: the blanked word is written back into its gap, in green. */
  reveal?: boolean
}>()
const { show } = useScript()

const plainWords = computed(() => props.subtitle.sr?.text.split(/\s+/).filter(Boolean) ?? [])
const srWords = computed(() => {
  const sr = props.subtitle.sr
  if (!sr) return []
  return props.mask ? maskAnswer(sr.text, props.mask) : plainWords.value
})
// Which words of the line are the gap (the masked ones differ from the plain ones).
const isGap = (i: number) => !!props.mask && srWords.value[i] !== plainWords.value[i]
// The gap holds the letters of the word; the punctuation stuck to it stays outside, after the gap.
const core = (w: string) => w.replace(/[^\p{L}]+$/u, '')
const tail = (w: string) => w.slice(core(w).length)
</script>

<template>
  <div class="min-h-[6.5rem] space-y-1.5 text-center" aria-live="polite">
    <p
      v-if="subtitle.sr"
      class="text-xl font-extrabold leading-snug transition-opacity duration-300 sm:text-2xl"
      :class="subtitle.sr.active ? '' : 'opacity-45'"
      data-test="story-sr"
    >
      <template v-for="(w, i) in srWords" :key="i">
        <template v-if="isGap(i)">
          <StoryGap :filled="reveal" :word="core(plainWords[i])" data-test="story-gap" />{{ tail(plainWords[i]) }}
        </template>
        <span v-else class="story-word" :class="{ 'story-word-on': subtitle.sr.active && i === wordIdx && !mask }">{{ show(w) }}</span
        >{{ ' ' }}
      </template>
    </p>
    <p
      v-if="subtitle.ru"
      class="leading-snug transition-all duration-300"
      :class="subtitle.ru.active ? 'text-lg font-bold' : 'text-sm text-[var(--muted)]'"
      data-test="story-ru"
    >
      {{ subtitle.ru.text }}
    </p>
  </div>
</template>

<style scoped>
.story-word {
  border-radius: 0.35rem;
  padding: 0 0.15em;
  transition: background-color 0.15s ease, color 0.15s ease;
}
.story-word-on {
  background: var(--accent-soft);
  color: var(--accent);
}
</style>
