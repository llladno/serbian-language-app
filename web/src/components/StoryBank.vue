<script setup lang="ts">
// «Соберите по-сербски»: the sentence is built from word tiles, like the lesson's word bank. A right answer is
// replaced by a big «Верно»; a wrong one shows the right sentence.
import { computed, ref } from 'vue'
import { useScript } from '../lib/useScript'
import { shuffled } from '../lib/shuffle'
import { SAY_DELAY_MS, useStoryVerdict } from '../lib/useStoryVerdict'
import type { StoryVerdict as Verdict } from '../types'
import StoryVerdict from './StoryVerdict.vue'

const props = defineProps<{
  prompt: string
  eyebrow?: string
  bank: string[]
  play?: (url: string) => void
  /** The sentence, said once the question is answered, and its length in ms. */
  say?: string
  sayMs?: number
  check: (choice: string) => Promise<Verdict>
}>()
const emit = defineEmits<{ done: [correct: boolean] }>()
const { show } = useScript()
const { verdict, lockedHeight, setControls, settle } = useStoryVerdict((ok) => emit('done', ok))

const tiles = ref(shuffled(props.bank.map((text, id) => ({ id, text }))))
const picked = ref<number[]>([])
const pending = ref(false)

const available = computed(() => tiles.value.filter((t) => !picked.value.includes(t.id)))
const built = computed(() => picked.value.map((id) => tiles.value.find((t) => t.id === id)!.text))

function add(id: number) {
  if (!verdict.value) picked.value.push(id)
}
function removeAt(i: number) {
  if (!verdict.value) picked.value.splice(i, 1)
}
async function submit() {
  if (pending.value || verdict.value || !picked.value.length) return
  pending.value = true
  let v: Verdict
  try {
    v = await props.check(built.value.join(' '))
  } catch {
    v = { correct: false, answer: '' }
  }
  settle(v, props.say ? (props.sayMs ?? 0) : 0)
  if (props.say) setTimeout(() => props.play?.(props.say!), SAY_DELAY_MS)
}
</script>

<template>
  <div class="text-center" data-test="story-bank">
    <p v-if="eyebrow" class="mb-1 text-xs font-semibold text-[var(--muted)]">{{ eyebrow }}</p>
    <p class="mb-3 text-xl font-medium leading-snug">{{ prompt }}</p>

    <Transition name="verdict-swap" mode="out-in">
      <div
        v-if="verdict?.correct"
        key="ok"
        class="flex items-center justify-center"
        :style="{ minHeight: lockedHeight + 'px' }"
      >
        <StoryVerdict :answer="verdict.answer" serbian />
      </div>

      <div v-else key="ask" :ref="setControls">
        <div class="mb-3 flex min-h-14 flex-wrap items-center justify-center gap-2 rounded-2xl bg-[var(--bg-soft)] p-3">
          <button
            v-for="(id, i) in picked"
            :key="id"
            type="button"
            class="serbian min-h-9 rounded-xl bg-[var(--card)] px-3.5 py-1.5 text-sm text-[var(--fg)] shadow-[var(--shadow)]"
            :disabled="!!verdict"
            data-test="story-chosen"
            @click="removeAt(i)"
          >
            {{ show(tiles.find((t) => t.id === id)!.text) }}
          </button>
          <span v-if="!picked.length" class="text-sm text-[var(--muted)]">нажимайте слова по порядку</span>
        </div>
        <div class="mb-3 flex flex-wrap justify-center gap-2">
          <button
            v-for="t in available"
            :key="t.id"
            type="button"
            class="btn btn-ghost serbian"
            :disabled="!!verdict"
            data-test="story-tile"
            @click="add(t.id)"
          >
            {{ show(t.text) }}
          </button>
        </div>

        <div v-if="verdict && !verdict.correct" class="pop text-sm">
          <p class="font-semibold text-[var(--bad)]">✗ Не совсем</p>
          <p>Правильно: <span class="serbian font-semibold">{{ show(verdict.answer) }}</span></p>
        </div>
        <button v-else type="button" class="btn btn-primary w-full" :disabled="pending || !picked.length" data-test="story-submit" @click="submit">
          Проверить
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.verdict-swap-leave-active {
  transition: opacity 0.18s ease;
}
.verdict-swap-leave-to {
  opacity: 0;
}
</style>
