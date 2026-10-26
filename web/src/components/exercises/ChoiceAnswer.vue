<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { api } from '../../api'
import { useWrongGoesToEnd } from '../../lib/lessonRules'
import { useScript } from '../../lib/useScript'
import { speakClip } from '../../lib/sound'
import type { CheckResult, LessonAttempt } from '../../types'
import HintButton from '../HintButton.vue'
import PhraseSpeak from '../PhraseSpeak.vue'
import PromptSpeak from '../PromptSpeak.vue'
import SpeakButton from '../SpeakButton.vue'
import SuccessCheck from '../SuccessCheck.vue'

const props = defineProps<{
  lesson: string
  exerciseId: string
  prompt: string
  options: string[]
  explain?: string
  // «Послушай и выбери»: a clip instead of text. With `text` (audio off) the
  // sentence is shown instead of the clip, for a learner who cannot listen.
  audio?: string
  text?: string
  prior?: LessonAttempt
}>()
const emit = defineEmits<{
  graded: [ok: boolean, result: CheckResult, answer: string]
  ungraded: []
  cantListen: []
}>()
// Display only: what is sent to the server and compared stays the Latin original.
const { show } = useScript()

const shuffled = ref([...props.options].sort(() => Math.random() - 0.5))
const picked = ref<string | null>(props.prior?.answer ?? null)
const result = ref<CheckResult | null>(null)
const fromPrior = ref(!!props.prior)

// After a mistake the way forward is the lesson's own button (the exercise
// comes back at the end of the step), not an instant redo.
const wrongGoesToEnd = useWrongGoesToEnd()
const showRetry = computed(
  () => (fromPrior.value || !!result.value) && !(wrongGoesToEnd.value && result.value && !result.value.ok),
)
const pending = ref(false)

// «Послушай и выбери» starts the clip by itself; the speaker stays for hearing it again.
onMounted(() => {
  if (props.audio && !props.text && !props.prior) speakClip(props.audio)
})

async function choose(opt: string) {
  if (pending.value || result.value || fromPrior.value) return
  picked.value = opt
  pending.value = true
  try {
    result.value = await api.check(props.lesson, props.exerciseId, { answer: opt })
    emit('graded', !!result.value.ok, result.value, opt)
  } finally {
    pending.value = false
  }
}

function retry() {
  result.value = null
  fromPrior.value = false
  picked.value = null
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

    <div v-if="audio || text" class="mb-5 flex flex-col items-center gap-3 px-8">
      <p v-if="text" class="serbian whitespace-pre-wrap text-xl font-medium" data-test="listen-as-text">{{ show(text) }}</p>
      <SpeakButton v-else :src="audio" :size="52" />
      <p v-if="result?.say" class="serbian text-xl font-medium" data-test="listen-said">{{ show(result.say) }}</p>
      <span class="flex items-center gap-1.5 text-sm text-[var(--muted)]">
        {{ show(prompt) }}
        <HintButton v-if="explain && !result && !fromPrior" :text="show(explain)" />
      </span>
      <button
        v-if="audio && !fromPrior && !result"
        type="button"
        class="btn btn-ghost text-sm"
        data-test="cant-listen"
        @click="emit('cantListen')"
      >
        не могу прослушать :(
      </button>
    </div>
    <p v-else class="mb-5 flex flex-wrap items-center justify-center gap-1.5 whitespace-pre-wrap px-8 text-xl font-medium">
      {{ show(prompt) }}
      <PromptSpeak :prompt="prompt" />
      <HintButton v-if="explain && !result && !fromPrior" :text="show(explain)" />
    </p>

    <div class="flex flex-wrap justify-center gap-2.5">
      <button
        v-for="o in shuffled"
        :key="o"
        class="btn btn-ghost serbian"
        :class="{
          'ring-2 ring-[var(--good)]': (result?.ok || (fromPrior && prior?.correct)) && picked === o,
          'ring-2 ring-[var(--bad)]': ((result && !result.ok) || (fromPrior && !prior?.correct)) && picked === o,
        }"
        :disabled="!!result || fromPrior"
        @click="choose(o)"
      >
        {{ show(o) }}
      </button>
    </div>

    <div v-if="fromPrior" class="mt-4 text-sm">
      <p class="font-semibold" :class="prior!.correct ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        {{ prior!.correct ? '✓ Отвечено верно' : '✗ Был неверный ответ' }}
      </p>
      <p v-if="prior!.correct" class="mt-1 flex justify-center"><PhraseSpeak :text="prior!.answer" /></p>
    </div>

    <div v-else-if="result" class="pop mt-4 text-sm">
      <p class="flex items-center justify-center gap-2 font-semibold" :class="result.ok ? 'text-[var(--good)]' : 'text-[var(--bad)]'">
        <SuccessCheck v-if="result.ok" />
        {{ result.ok ? 'Верно' : '✗ Не то' }}
      </p>
      <p v-if="!result.ok && result.expected">
        Правильно: <span class="serbian font-semibold">{{ show(result.expected) }}</span>
      </p>
      <!-- only ever the right phrase: the options themselves stay silent, a wrong one must not be voiced -->
      <p class="mt-1 flex justify-center">
        <PhraseSpeak :text="result.ok ? picked! : result.expected" autoplay />
      </p>
      <p v-if="result.explain" class="mt-1 text-[var(--muted)]">{{ show(result.explain) }}</p>
    </div>
  </div>
</template>
