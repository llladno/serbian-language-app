<script setup lang="ts">
import { computed, ref } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { api } from '../../api'
import { useWrongGoesToEnd } from '../../lib/lessonRules'
import type { CheckResult, LessonAttempt } from '../../types'
import SerbianKeys from '../SerbianKeys.vue'
import SpeakButton from '../SpeakButton.vue'
import GlossedText from '../GlossedText.vue'
import BottomBar from '../BottomBar.vue'
import HintButton from '../HintButton.vue'
import PhraseSpeak from '../PhraseSpeak.vue'
import PromptSpeak from '../PromptSpeak.vue'
import { fillSentence } from '../../lib/phraseAudio'

const props = defineProps<{
  lesson: string
  exerciseId: string
  type: 'translate' | 'fill_blank' | 'fix_error' | 'listen'
  prompt: string
  audio?: string
  explain?: string
  prior?: LessonAttempt
  // A dictation for someone who cannot listen: the sentence arrives as text
  // and is translated into Russian instead of being written down. Everything
  // about the screen follows from this being set — no speaker, no Serbian
  // keys, and the answer is Russian.
  text?: string
}>()

// fill_blank / fix_error prompts are Serbian sentences — make their words
// clickable. A translate prompt is Russian (and its Serbian answer must not be
// spoiled), so it stays plain. A listen prompt is just an instruction; the
// sentence to transcribe lives only in the audio.
const glossPrompt = computed(() => props.type === 'fill_blank' || props.type === 'fix_error')

const emit = defineEmits<{ graded: [ok: boolean, result: CheckResult, answer: string]; ungraded: []; cantListen: [] }>()

const answer = ref(props.prior?.answer ?? '')
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)

// After a mistake the way forward is the lesson's own button (the exercise
// comes back at the end of the step), not an instant redo.
const wrongGoesToEnd = useWrongGoesToEnd()
const showRetry = computed(
  () => (fromPrior.value || !!result.value) && !(wrongGoesToEnd.value && result.value && !result.value.ok),
)
const pending = ref(false)

async function submit() {
  if (pending.value || !answer.value.trim()) return
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, {
      answer: answer.value,
      ...(props.text ? { ru: true } : {}),
    })
    fromPrior.value = false
    emit('graded', !!result.value.ok, result.value, answer.value)
  } finally {
    pending.value = false
  }
}

// What the speaker under the verdict reads: the whole right sentence (for a
// gap-fill the prompt's sentence with the gap filled, not just the missing
// word). A dictation already has its own speaker on top, and the text-only
// "can't listen" mode answers in Russian.
const resultPhrase = computed(() => {
  if (!result.value || props.text || props.type === 'listen') return undefined
  const word = result.value.ok ? answer.value : result.value.expected
  if (!word) return undefined
  return props.type === 'fill_blank' ? fillSentence(props.prompt, word) : word
})

function retry() {
  result.value = null
  fromPrior.value = false
  answer.value = ''
  emit('ungraded')
}
</script>

<template>
  <div class="relative text-center">
    <button
      v-if="showRetry"
      class="icon-btn absolute right-0 top-0"
      title="Переделать"
      @click="retry"
    >
      <RefreshCw :size="15" :stroke-width="2.25" />
    </button>

    <div v-if="type === 'listen' && text" class="mb-5 px-8" data-test="listen-as-text">
      <p class="mb-1.5 text-sm text-[var(--muted)]">{{ prompt }}</p>
      <p class="serbian whitespace-pre-wrap text-xl font-medium">{{ text }}</p>
    </div>
    <div v-else-if="type === 'listen'" class="mb-5 flex flex-col items-center gap-3 px-8">
      <SpeakButton :src="audio" :size="52" />
      <span class="flex items-center gap-1.5 text-sm text-[var(--muted)]">
        {{ prompt || 'Напиши, что слышишь' }}
        <HintButton v-if="explain && !fromPrior && !result" :text="explain" />
      </span>
      <button
        v-if="!fromPrior && !result"
        type="button"
        class="btn btn-ghost text-sm"
        data-test="cant-listen"
        @click="emit('cantListen')"
      >
        не могу прослушать :(
      </button>
    </div>
    <div v-else class="mb-5 px-8 text-center">
      <p v-if="type === 'fix_error'" class="mb-1.5 text-[10px] uppercase tracking-widest text-[var(--accent)]">
        найди и исправь ошибку
      </p>
      <p class="flex flex-wrap items-center justify-center gap-1.5 whitespace-pre-wrap text-xl font-medium">
        <GlossedText v-if="glossPrompt" :text="prompt" />
        <template v-else>{{ prompt }}</template>
        <PromptSpeak :prompt="prompt" />
        <HintButton v-if="explain && !fromPrior && !result" :text="explain" />
      </p>
    </div>

    <!-- previously answered -->
    <div v-if="fromPrior" class="text-sm">
      <p class="mb-1 flex items-center justify-center gap-1.5 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        <span>{{ prior!.correct ? '✓' : '✗' }}</span>
        <span>{{ prior!.correct ? 'Отвечено верно' : 'Был ответ с ошибкой' }}</span>
      </p>
      <p class="text-[var(--muted)]">ты писал: <span class="text-[var(--fg)]" :class="{ serbian: !text }">{{ prior!.answer }}</span></p>
    </div>

    <form v-if="!fromPrior && !result" class="mx-auto max-w-sm" @submit.prevent="submit">
      <input
        v-model="answer"
        type="text"
        autocomplete="off"
        autocapitalize="off"
        autocorrect="off"
        spellcheck="false"
        class="field w-full text-center"
        :placeholder="text ? 'перевод…' : 'ответ…'"
      />
      <SerbianKeys v-if="!text" class="mt-1.5 justify-center" />
    </form>
    <BottomBar v-if="!fromPrior && !result">
      <button class="btn btn-primary w-full" :disabled="pending || !answer.trim()" @click="submit">
        Проверить
      </button>
    </BottomBar>

    <div v-if="result" class="pop mt-4">
      <p
        class="mb-1 flex items-center justify-center gap-1.5 font-semibold"
        :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'"
      >
        <span>{{ result.ok ? '✓' : '✗' }}</span>
        <span>{{ result.ok ? 'Верно' : result.near_miss ? 'Почти — опечатка?' : 'Не совсем' }}</span>
      </p>
      <p v-if="result.diff?.length && !result.ok" class="mb-1">
        <span v-for="(c, i) in result.diff" :key="i" :class="c.ok ? '' : 'chunk-wrong'">{{ c.text
        }}{{ i < result.diff.length - 1 ? ' ' : '' }}</span>
      </p>
      <p v-if="!result.ok && result.expected" class="text-sm">
        Правильно: <span class="font-semibold" :class="{ serbian: !text }">{{ result.expected }}</span>
      </p>
      <p v-if="resultPhrase" class="mt-1 flex justify-center">
        <PhraseSpeak :text="resultPhrase" />
      </p>
      <p v-if="result.explain" class="mt-1 text-sm text-[var(--muted)]">{{ result.explain }}</p>
    </div>
  </div>
</template>
