<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { api } from '../../api'
import { useWrongGoesToEnd } from '../../lib/lessonRules'
import { useScript } from '../../lib/useScript'
import type { CheckResult, LessonAttempt } from '../../types'
import SerbianKeys from '../SerbianKeys.vue'
import SpeakButton from '../SpeakButton.vue'
import GlossedText from '../GlossedText.vue'
import BottomBar from '../BottomBar.vue'
import HintButton from '../HintButton.vue'
import PhraseSpeak from '../PhraseSpeak.vue'
import PromptSpeak from '../PromptSpeak.vue'
import { fillSentence } from '../../lib/phraseAudio'
import { speakClip } from '../../lib/sound'
import SuccessCheck from '../SuccessCheck.vue'

const props = defineProps<{
  lesson: string
  exerciseId: string
  type: 'translate' | 'fill_blank' | 'fix_error' | 'listen'
  prompt: string
  audio?: string
  explain?: string
  // What kind of text is wanted (see Exercise.shape): the «?» says so in words.
  shape?: 'word' | 'phrase' | 'blank' | 'sentence'
  prior?: LessonAttempt
  // A dictation for someone who cannot listen: the sentence arrives as text
  // and is translated into Russian instead of being written down. Everything
  // about the screen follows from this being set — no speaker, no Serbian
  // keys, and the answer is Russian.
  text?: string
  // Mounted but not on screen (see ExerciseBlock): the dictation waits until it is.
  offscreen?: boolean
}>()

// fill_blank / fix_error prompts are Serbian sentences — make their words
// clickable. A translate prompt is Russian (and its Serbian answer must not be
// spoiled), so it stays plain. A listen prompt is just an instruction; the
// sentence to transcribe lives only in the audio.
const glossPrompt = computed(() => props.type === 'fill_blank' || props.type === 'fix_error')

const emit = defineEmits<{ graded: [ok: boolean, result: CheckResult, answer: string]; ungraded: []; cantListen: [] }>()
// Display only: the typed answer goes to the server exactly as typed, and the
// server compares it with the Latin answer (it accepts Cyrillic input too).
const { show } = useScript()

// fix_error: start from the wrong sentence so the learner edits it instead of
// retyping (people were typing only the missing word — «ja sam» — and failing).
// Prompts with Russian notes in them can't be edited as-is, so those start empty.
function startingAnswer() {
  return props.type === 'fix_error' && !/[А-Яа-яЁё]/.test(props.prompt) ? props.prompt : ''
}
// The «?» next to the prompt: first what kind of answer is wanted (a word, a phrase, the missing word or
// the whole sentence: people lost points over exactly this), then the exercise's own hint if it has one.
const formatLine = computed(() => {
  if (props.text) return 'Напиши перевод на русский.'
  switch (props.shape) {
    case 'blank':
      return 'Впиши только пропущенное слово. Можно и всё предложение: это тоже засчитается.'
    case 'word':
      return 'Напиши одно слово.'
    case 'phrase':
      return 'Напиши фразу целиком, не одно слово.'
    case 'sentence':
      return 'Перепиши предложение целиком, исправив ошибку, а не одно слово.'
    default:
      return ''
  }
})
const helpText = computed(() => [formatLine.value, props.explain].filter(Boolean).join(' '))

const answer = ref(props.prior?.answer ?? startingAnswer())
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)

// A dictation starts by itself when it comes on screen, so the learner does not have to find the speaker first
// (an answered one shown again, and the text-only mode, stay silent). The speaker stays for hearing it again.
watch(
  () => props.offscreen,
  (off) => {
    if (off || props.type !== 'listen' || !props.audio || props.text || fromPrior.value || result.value) return
    speakClip(props.audio)
  },
  { immediate: true },
)

// After a mistake the way forward is the lesson's own button (the exercise
// comes back at the end of the step), not an instant redo.
const wrongGoesToEnd = useWrongGoesToEnd()
const showRetry = computed(
  () => (fromPrior.value || !!result.value) && !(wrongGoesToEnd.value && result.value && !result.value.ok),
)
const pending = ref(false)

// The answer was only a piece of the right one (just the corrected word, the first words). It is a slip
// of format, not of knowledge, so nothing is graded: the learner is told and finishes it, for free.
const tooShort = ref(false)
watch(answer, () => {
  tooShort.value = false
})

async function submit() {
  if (pending.value || !answer.value.trim()) return
  pending.value = true
  try {
    const res = await api.check(props.lesson, props.exerciseId, {
      answer: answer.value,
      ...(props.text ? { ru: true } : {}),
    })
    if (res.format === 'too_short') {
      tooShort.value = true
      return
    }
    result.value = res
    fromPrior.value = false
    emit('graded', !!res.ok, res, answer.value)
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
  answer.value = startingAnswer()
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
      <p class="mb-1.5 text-sm text-[var(--muted)]">{{ show(prompt) }}</p>
      <p class="serbian whitespace-pre-wrap text-xl font-medium">{{ show(text) }}</p>
    </div>
    <div v-else-if="type === 'listen'" class="mb-5 flex flex-col items-center gap-3 px-8">
      <SpeakButton :src="audio" :size="52" />
      <span class="flex flex-wrap items-center justify-center gap-1.5 text-sm text-[var(--muted)]">
        {{ show(prompt) || 'Напиши, что слышишь' }}
        <HintButton v-if="helpText && !fromPrior && !result" :text="show(helpText)" />
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
        найди и исправь ошибку — перепиши предложение целиком
      </p>
      <p class="flex flex-wrap items-center justify-center gap-1.5 whitespace-pre-wrap text-xl font-medium">
        <GlossedText v-if="glossPrompt" :text="prompt" />
        <template v-else>{{ show(prompt) }}</template>
        <PromptSpeak :prompt="prompt" />
        <HintButton v-if="helpText && !fromPrior && !result" :text="helpText" />
      </p>
    </div>

    <!-- previously answered -->
    <div v-if="fromPrior" class="text-sm">
      <p class="mb-1 flex items-center justify-center gap-1.5 font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        <span>{{ prior!.correct ? '✓' : '✗' }}</span>
        <span>{{ prior!.correct ? 'Отвечено верно' : 'Был ответ с ошибкой' }}</span>
      </p>
      <p class="text-[var(--muted)]">ты писал: <span class="text-[var(--fg)]" :class="{ serbian: !text }">{{ show(prior!.answer) }}</span></p>
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
        :placeholder="text ? 'перевод…' : type === 'fix_error' ? 'исправь предложение…' : 'ответ…'"
      />
      <SerbianKeys v-if="!text" class="mt-1.5 justify-center" />
    </form>
    <p v-if="tooShort" class="mt-3 text-sm text-[var(--warning)]" data-test="too-short">
      <template v-if="type === 'fix_error'">Это только часть. Перепиши предложение целиком, исправив ошибку.</template>
      <template v-else>Это только часть ответа. Напиши фразу целиком.</template>
    </p>
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
        <SuccessCheck v-if="result.ok" />
        <span v-else>✗</span>
        <span>{{ result.ok ? 'Верно' : result.near_miss ? 'Почти — опечатка?' : 'Не совсем' }}</span>
      </p>
      <div v-if="result.ok && result.typo && result.expected" class="mb-1 text-sm text-[var(--warning)]" data-test="typo-note">
        <p>Засчитано, но проверь написание.</p>
        <p>Вы написали: <span class="font-semibold" :class="{ serbian: !text }">{{ show(answer.trim()) }}</span></p>
        <p>Правильно: <span class="font-semibold" :class="{ serbian: !text }">{{ show(result.expected) }}</span></p>
      </div>
      <p v-if="result.diff?.length && !result.ok" class="mb-1" data-test="wrote">
        <span class="text-sm">Вы написали: </span
        ><span v-for="(c, i) in result.diff" :key="i" :class="c.ok ? '' : 'chunk-wrong'">{{ show(c.text)
        }}{{ i < result.diff.length - 1 ? ' ' : '' }}</span>
      </p>
      <p v-if="!result.ok && result.expected" class="text-sm">
        Правильно: <span class="font-semibold" :class="{ serbian: !text }">{{ show(result.expected) }}</span>
      </p>
      <p v-if="resultPhrase" class="mt-1 flex justify-center">
        <PhraseSpeak :text="resultPhrase" autoplay />
      </p>
      <p v-if="result.explain" class="mt-1 text-sm text-[var(--muted)]">{{ show(result.explain) }}</p>
    </div>
  </div>
</template>
