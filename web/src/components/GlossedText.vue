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
const pos = ref({ x: 0, y: 0 })
const word = ref('')
const loading = ref(false)
const result = ref<LookupResult | null>(null)
let anchor: HTMLElement | null = null

// Keep the card pinned under its word as the page scrolls; dismiss it once the
// word leaves the viewport.
function reposition() {
  if (!open.value || !anchor) return
  const r = anchor.getBoundingClientRect()
  if (r.bottom < 0 || r.top > window.innerHeight) {
    close()
    return
  }
  pos.value = { x: r.left + r.width / 2, y: r.bottom }
}

async function onWordClick(w: string, e: MouseEvent) {
  anchor = e.target as HTMLElement
  const r = anchor.getBoundingClientRect()
  pos.value = { x: r.left + r.width / 2, y: r.bottom }
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
        class="card absolute z-50 max-w-[min(20rem,90vw)] -translate-x-1/2 p-3 text-sm shadow-lg"
        :style="{ left: pos.x + 'px', top: pos.y + 8 + 'px' }"
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
