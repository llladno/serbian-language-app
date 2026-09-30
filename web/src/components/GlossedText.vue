<script setup lang="ts">
// Renders Serbian text with every word clickable: a click looks the word up in
// the course dictionary (lib/lookup) and shows a small card next to it. Used by
// the reading text and by exercise prompts.
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import type { LookupResult } from '../types'
import { tokenize } from '../lib/reading'
import { lookupWord } from '../lib/lookup'
import { addWordToReview, isAddedToReview } from '../lib/review'
import SpeakButton from './SpeakButton.vue'

const props = defineProps<{ text: string }>()

const adding = ref('')
async function addToReview(vocabId: string) {
  if (adding.value || isAddedToReview(vocabId)) return
  adding.value = vocabId
  try {
    await addWordToReview(vocabId)
  } catch {
    /* leave the button as-is; user can retry */
  } finally {
    adding.value = ''
  }
}

const tokens = computed(() => tokenize(props.text))

const open = ref(false)
// Pixel-exact placement (left/top-or-bottom/width/maxHeight), not a CSS
// -translate-x-1/2 trick — that centers on the anchor with no idea how wide
// the card actually is, so it happily runs off-screen for a word near
// either edge (see computeCardStyle). Also clamps height so a long list of
// matches scrolls inside the card instead of spilling past the viewport.
const cardStyle = ref<Record<string, string>>({})
const word = ref('')
const loading = ref(false)
const result = ref<LookupResult | null>(null)
let anchor: HTMLElement | null = null

const CARD_MARGIN = 8
const CARD_MAX_W = 320 // ~20rem
const CARD_MIN_SIDE_SPACE = 140 // below this, flip to placing the card above the word

function computeCardStyle(anchorEl: HTMLElement): Record<string, string> {
  const r = anchorEl.getBoundingClientRect()
  const vw = window.innerWidth
  const vh = window.innerHeight

  const width = Math.min(CARD_MAX_W, vw - CARD_MARGIN * 2)
  const left = Math.max(CARD_MARGIN, Math.min(r.left + r.width / 2 - width / 2, vw - width - CARD_MARGIN))

  const spaceBelow = vh - r.bottom - CARD_MARGIN
  const spaceAbove = r.top - CARD_MARGIN
  const below = spaceBelow >= CARD_MIN_SIDE_SPACE || spaceBelow >= spaceAbove
  const maxHeight = Math.max(120, below ? spaceBelow : spaceAbove)

  return {
    left: `${left}px`,
    width: `${width}px`,
    maxHeight: `${maxHeight}px`,
    ...(below ? { top: `${r.bottom + CARD_MARGIN}px` } : { bottom: `${vh - r.top + CARD_MARGIN}px` }),
  }
}

// Keep the card pinned under (or above) its word as the page scrolls;
// dismiss it once the word leaves the viewport.
function reposition() {
  if (!open.value || !anchor) return
  const r = anchor.getBoundingClientRect()
  if (r.bottom < 0 || r.top > window.innerHeight) {
    close()
    return
  }
  cardStyle.value = computeCardStyle(anchor)
}

async function onWordClick(w: string, e: MouseEvent) {
  anchor = e.target as HTMLElement
  cardStyle.value = computeCardStyle(anchor)
  word.value = w
  open.value = true
  loading.value = true
  result.value = null
  const res = await lookupWord(w)
  if (open.value && word.value === w) {
    result.value = res
    loading.value = false
  }
}

function close() {
  open.value = false
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') close()
}

// Global listeners live only while the card is open — a lesson page can hold
// dozens of GlossedText instances.
watch(open, (isOpen) => {
  const fn = isOpen ? window.addEventListener : window.removeEventListener
  fn('keydown', onKey as EventListener)
  fn('resize', reposition)
  fn('scroll', reposition, true)
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey as EventListener)
  window.removeEventListener('resize', reposition)
  window.removeEventListener('scroll', reposition, true)
})
</script>

<template>
  <span class="glossed">
    <template v-for="(t, i) in tokens" :key="i"
      ><span
        v-if="t.word"
        class="cursor-pointer rounded px-[1px] transition hover:bg-[var(--ring-track)] hover:text-[var(--accent)]"
        :class="{ 'bg-[var(--ring-track)] text-[var(--accent)]': open && word === t.text }"
        @click="onWordClick(t.text, $event)"
        >{{ t.text }}</span
      ><template v-else>{{ t.text }}</template></template
    >

    <!-- lookup popover -->
    <div v-if="open" class="fixed inset-0 z-40" @click="close">
      <div
        class="card absolute z-50 overflow-y-auto p-3 text-sm shadow-lg"
        :style="cardStyle"
        @click.stop
      >
        <p v-if="loading" class="text-[var(--muted)]">…</p>

        <template v-else-if="result && result.matches.length">
          <p v-if="result.partial" class="mb-1 text-xs text-[var(--muted)]">
            похоже на «{{ word }}» — возможно:
          </p>
          <div
            v-for="m in result.matches"
            :key="m.id"
            class="flex items-start gap-2 border-[var(--border)] py-1 [&:not(:first-child)]:border-t"
          >
            <SpeakButton v-if="m.audio" :src="m.audio" :size="24" class="mt-[2px]" />
            <div class="min-w-0">
              <div class="font-semibold">
                {{ m.latin }}
                <span v-if="m.cyrillic" class="font-normal text-[var(--muted)]">· {{ m.cyrillic }}</span>
                <span v-if="m.transcription" class="font-normal text-[var(--muted)]">[{{ m.transcription }}]</span>
              </div>
              <div>{{ m.ru }}</div>
              <div v-if="m.note" class="text-xs text-[var(--muted)]">{{ m.note }}</div>
              <button
                type="button"
                class="mt-1 text-xs font-medium"
                :class="
                  isAddedToReview(m.id)
                    ? 'text-[var(--good)]'
                    : 'text-[var(--accent)] hover:underline disabled:opacity-50'
                "
                :disabled="isAddedToReview(m.id) || adding === m.id"
                @click="addToReview(m.id)"
              >
                {{ isAddedToReview(m.id) ? '✓ в очереди на повторение' : '＋ в повторение' }}
              </button>
            </div>
          </div>
        </template>

        <template v-else>
          <p class="text-[var(--muted)]">«{{ word }}» — нет в словаре курса</p>
          <RouterLink
            :to="{ name: 'vocab', query: { q: word } }"
            class="mt-1 inline-block text-[var(--accent)] hover:underline"
            @click="close"
          >
            искать в словаре →
          </RouterLink>
        </template>
      </div>
    </div>
  </span>
</template>
