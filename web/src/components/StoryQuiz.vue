<script setup lang="ts">
// A question of a story with options to pick from: the quick ones of pass 3, and the final ones that are choices
// (about the story, translate, listen and pick, who said it, the missing word). Built like the lesson's «выбери»:
// the prompt, the options as ghost buttons, a ring when answered. A right answer is replaced by a big «Верно»
// (StoryVerdict); a wrong one shows the right answer. The checking is the server's job (the parent passes `check`).
import { computed, onMounted, ref, watch } from 'vue'
import { Volume2 } from 'lucide-vue-next'
import { useScript } from '../lib/useScript'
import { shuffled } from '../lib/shuffle'
import { SAY_DELAY_MS, useStoryVerdict } from '../lib/useStoryVerdict'
import type { StoryVerdict as Verdict } from '../types'
import StoryGap from './StoryGap.vue'
import StoryVerdict from './StoryVerdict.vue'

const props = withDefaults(defineProps<{
  prompt: string
  /** A small line above the prompt ("Какое слово прозвучало?"). */
  eyebrow?: string
  options: string[]
  /** Which parts are Serbian and so are shown in the learner's script. */
  promptSerbian?: boolean
  optionsSerbian?: boolean
  /** A clip to listen to (a question that asks to listen): played once on arrival and again by the speaker button. */
  audio?: string
  play?: (url: string) => void
  /** A clip said once the question is answered, like the lessons say the right phrase, and its length in ms. */
  say?: string
  sayMs?: number
  /** The prompt is a sentence with a gap («_____»): a right answer is written into the gap, in green. */
  fillGap?: boolean
  /** The right answer is written in place elsewhere on the screen (the subtitle's gap): «Верно» does not repeat it. */
  answerInPlace?: boolean
  /** Mix the options (default). The right one is written first in the story, so it must not stay first. */
  shuffle?: boolean
  check: (choice: string) => Promise<Verdict>
}>(), { shuffle: true }) // an absent Boolean prop is false in Vue, so the default has to be said
const emit = defineEmits<{ done: [correct: boolean] }>()
const { show } = useScript()

const { verdict, lockedHeight, setControls, settle, reset } = useStoryVerdict((ok) => emit('done', ok))
let picked = ''
let busy = false

// The prompt cut at its gap(s): the gap is a real element, so the right word can grow into it.
const promptParts = computed(() => (props.fillGap ? props.prompt.split(/(_{3,})/) : [props.prompt]))
const order = () => (props.shuffle === false ? [...props.options] : shuffled(props.options))
const shown = ref(order())

watch(
  () => props.prompt,
  () => {
    reset()
    picked = ''
    busy = false
    shown.value = order()
  },
)
onMounted(() => {
  if (props.audio) setTimeout(() => props.play?.(props.audio!), 350)
})

async function choose(opt: string) {
  if (busy || verdict.value) return
  busy = true
  picked = opt
  let v: Verdict
  try {
    v = await props.check(opt)
  } catch {
    v = { correct: false, answer: '' }
  }
  settle(v, props.say ? (props.sayMs ?? 0) : 0)
  if (props.say) setTimeout(() => props.play?.(props.say!), SAY_DELAY_MS)
}
</script>

<template>
  <div class="text-center" data-test="story-quiz">
    <p v-if="eyebrow" class="mb-1 text-xs font-semibold text-[var(--muted)]">{{ eyebrow }}</p>
    <p v-if="fillGap" class="mb-4 text-xl font-medium leading-snug" :class="{ serbian: promptSerbian }" data-test="story-prompt">
      <template v-for="(part, i) in promptParts" :key="i">
        <StoryGap v-if="/^_{3,}$/.test(part)" :filled="verdict?.correct" :word="verdict?.answer" data-test="story-gap" />
        <template v-else>{{ promptSerbian ? show(part) : part }}</template>
      </template>
    </p>
    <p v-else class="mb-4 text-xl font-medium leading-snug" :class="{ serbian: promptSerbian }">
      {{ promptSerbian ? show(prompt) : prompt }}
    </p>

    <Transition name="verdict-swap" mode="out-in">
      <div
        v-if="verdict?.correct"
        key="ok"
        class="flex items-center justify-center"
        :style="{ minHeight: lockedHeight + 'px' }"
      >
        <StoryVerdict :answer="optionsSerbian && !fillGap && !answerInPlace ? verdict.answer : undefined" :serbian="optionsSerbian" />
      </div>

      <div v-else key="ask" :ref="setControls">
        <div v-if="audio" class="mb-3 flex justify-center">
          <button type="button" class="btn btn-ghost h-[52px] w-[52px] rounded-full !p-0" aria-label="Послушать ещё раз" data-test="story-listen-again" @click="play?.(audio)">
            <Volume2 :size="22" :stroke-width="2.25" />
          </button>
        </div>
        <div class="grid gap-2">
          <button
            v-for="o in shown"
            :key="o"
            type="button"
            class="btn btn-ghost w-full"
            :class="{
              serbian: optionsSerbian,
              'ring-2 ring-[var(--good)]': verdict && o === verdict.answer,
              'ring-2 ring-[var(--bad)]': verdict && !verdict.correct && o === picked,
            }"
            :disabled="!!verdict"
            data-test="story-option"
            @click="choose(o)"
          >
            {{ optionsSerbian ? show(o) : o }}
          </button>
        </div>
        <div v-if="verdict && !verdict.correct" class="pop mt-3 text-sm">
          <p class="font-semibold text-[var(--bad)]">✗ Не то</p>
          <p v-if="verdict.answer">
            Правильно: <span class="font-semibold" :class="{ serbian: optionsSerbian }">{{ optionsSerbian ? show(verdict.answer) : verdict.answer }}</span>
          </p>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
/* Opacity only on the way out, a small grow on the way in: the pinned bar stays pinned (see BottomBar.vue). */
.verdict-swap-leave-active {
  transition: opacity 0.18s ease;
}
.verdict-swap-leave-to {
  opacity: 0;
}
</style>
