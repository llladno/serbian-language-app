<script setup lang="ts">
// A dialogue step: the conversation grows downwards, one turn at a time. A
// "me" turn shows its exercise; once answered — right or wrong — the canonical
// line takes its place as a bubble, so the thread of the conversation never
// breaks.
//
// What follows an answer arrives like a conversation does: the other person's
// lines come in one at a time, each fading up into place with the page scrolling
// smoothly to it, and the next question waits until they have all been said.
// Whatever was already on the screen when the step opened (a dialogue resumed,
// or looked back at) is simply there.
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import type { CheckResult, Exercise, LessonAttempt, Step } from '../types'
import DialogueBubble from './DialogueBubble.vue'
import ChoiceAnswer from './exercises/ChoiceAnswer.vue'
import TextAnswer from './exercises/TextAnswer.vue'

const props = defineProps<{
  lesson: string
  step: Step
  exercises: Exercise[]
  priors: Record<string, LessonAttempt>
}>()
const emit = defineEmits<{ graded: [exerciseId: string, ok: boolean]; ungraded: [exerciseId: string] }>()

const calm = () => typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

const showTranslations = ref(localStorage.getItem('dialogue.translations') === '1')
function toggleTranslations() {
  showTranslations.value = !showTranslations.value
  localStorage.setItem('dialogue.translations', showTranslations.value ? '1' : '0')
}

const turns = computed(() => props.step.turns ?? [])
const byID = computed(() => Object.fromEntries(props.exercises.map((e) => [e.id, e])))

// Lines learned during this session, keyed by exercise id: the server sends
// them with the check result. Turns answered in an earlier session already
// carry their sr/ru in the lesson payload.
const answered = reactive<Record<string, CheckResult>>({})
// What the learner actually answered, for showing it next to the right line.
const given = reactive<Record<string, string>>({})

function isDone(exerciseId: string) {
  return !!answered[exerciseId] || exerciseId in props.priors
}

// The conversation is revealed up to and including the first unanswered turn.
const visible = computed(() => {
  const out: { index: number; active: boolean }[] = []
  for (let i = 0; i < turns.value.length; i++) {
    const t = turns.value[i]
    const pending = t.who === 'me' && !!t.exercise_id && !isDone(t.exercise_id)
    out.push({ index: i, active: pending })
    if (pending) break
  }
  return out
})

function onGraded(exerciseId: string, ok: boolean, result: CheckResult, answer: string) {
  answered[exerciseId] = result
  given[exerciseId] = answer
  emit('graded', exerciseId, ok)
}

// ---- pacing -----------------------------------------------------------
// `visible` is what the conversation has got to; `shown` is how much of it is
// on the screen. They differ for a moment after each answer.
const initial = visible.value.length
const shown = ref(initial)
const onScreen = computed(() => visible.value.slice(0, shown.value))
let timer = 0

// How long the next line takes to "type": longer lines take longer, within
// limits that keep it from dragging.
function gapBefore(index: number) {
  const t = turns.value[index]
  const prev = turns.value[index - 1]
  if (t.who === 'me') return 450 // the next question, after the last line was read
  return Math.min(1300, Math.max(600, 450 + (prev?.sr?.length ?? 0) * 8))
}

watch(
  () => visible.value.length,
  (len) => {
    clearTimeout(timer)
    if (len <= shown.value || calm()) {
      shown.value = len
      scrollToEnd()
      return
    }
    const reveal = () => {
      shown.value++
      scrollToEnd()
      if (shown.value < len) timer = window.setTimeout(reveal, gapBefore(visible.value[shown.value].index))
    }
    timer = window.setTimeout(reveal, gapBefore(visible.value[shown.value].index))
  },
)
onBeforeUnmount(() => clearTimeout(timer))

// True once every question has been answered and the last line has been said:
// only then does the lesson offer its way on.
const finished = computed(
  () =>
    shown.value >= visible.value.length &&
    turns.value.every((t) => t.who !== 'me' || !t.exercise_id || isDone(t.exercise_id)),
)

const listEl = ref<HTMLElement | null>(null)
async function scrollToEnd() {
  await nextTick()
  const last = listEl.value?.lastElementChild as HTMLElement | null | undefined
  if (last && typeof last.scrollIntoView === 'function') {
    last.scrollIntoView({ behavior: calm() ? 'auto' : 'smooth', block: 'nearest' })
  }
}

// Things that arrive after the step opened — new lines, and the bubble an
// answer turns into — fade up. What was there from the start does not.
function arrives(index: number) {
  const t = turns.value[index]
  return index >= initial || (t.who === 'me' && !!t.exercise_id && t.exercise_id in answered)
}

// The lesson's back button steps a dialogue one turn at a time rather than
// leaving the whole step in one click: it undoes the last turn answered
// *this session* (an id in `answered`), reopening its exercise. A turn
// already recorded in an earlier session (`priors`, no local `answered`
// entry) can't be un-asked, so once undoing runs out of local answers —
// including when the dialogue was already fully done before this visit —
// stepBack reports false and the lesson moves to the previous step instead.
function stepBack(): boolean {
  for (let i = turns.value.length - 1; i >= 0; i--) {
    const t = turns.value[i]
    if (t.who === 'me' && t.exercise_id && t.exercise_id in answered) {
      const exId = t.exercise_id
      delete answered[exId]
      emit('ungraded', exId)
      return true
    }
  }
  return false
}

defineExpose({ stepBack, finished })

// A settled "me" turn: the canonical line, plus — when the answer was wrong —
// the miss marker and its explanation, so a mistake is never silently swallowed
// by the conversation moving on.
function lineOf(t: { exercise_id?: string; sr?: string; ru?: string; audio?: string }) {
  const res = t.exercise_id ? answered[t.exercise_id] : undefined
  const prior = t.exercise_id ? props.priors[t.exercise_id] : undefined
  const wrong = res ? !res.ok : prior ? !prior.correct : false
  return {
    sr: res?.line ?? t.sr ?? '',
    ru: res?.line_ru ?? t.ru,
    audio: res?.line_audio ?? t.audio,
    wrong,
    given: t.exercise_id ? (given[t.exercise_id] ?? prior?.answer) : undefined,
    // The server's note when the answer was just checked; after a reload only
    // the exercise's own explanation is left, which says the same thing.
    note: res?.explain ?? (t.exercise_id ? byID.value[t.exercise_id]?.explain : undefined),
  }
}
</script>

<template>
  <section class="space-y-4">
    <header class="flex items-start justify-between gap-3">
      <p class="rounded-xl bg-[var(--bg-soft)] px-4 py-2.5 text-sm text-[var(--muted)]">
        {{ step.scene }}
      </p>
      <div class="flex shrink-0 gap-2">
        <button
          type="button"
          data-test="toggle-translations"
          class="rounded-full bg-[var(--bg-soft)] px-3 py-1.5 text-xs text-[var(--muted)] transition hover:text-[var(--accent)]"
          :class="{ 'bg-[var(--accent-soft)] text-[var(--accent)]': showTranslations }"
          @click="toggleTranslations"
        >
          переводы
        </button>
      </div>
    </header>

    <div ref="listEl" class="space-y-3">
      <div
        v-for="v in onScreen"
        :key="v.index"
        class="msg"
        :class="[turns[v.index].who, arrives(v.index) ? (v.active ? 'msg-fade' : 'msg-in') : '']"
        :data-index="v.index"
      >
        <DialogueBubble
          v-if="turns[v.index].who === 'npc'"
          who="npc"
          :sr="turns[v.index].sr ?? ''"
          :ru="turns[v.index].ru"
          :audio="turns[v.index].audio"
          :show-translation="showTranslations"
        />

        <template v-else>
          <DialogueBubble
            v-if="!v.active"
            who="me"
            :sr="lineOf(turns[v.index]).sr"
            :ru="lineOf(turns[v.index]).ru"
            :audio="lineOf(turns[v.index]).audio"
            :wrong="lineOf(turns[v.index]).wrong"
            :given="lineOf(turns[v.index]).given"
            :note="lineOf(turns[v.index]).note"
            :show-translation="showTranslations"
          />

          <ChoiceAnswer
            v-else-if="byID[turns[v.index].exercise_id!]?.type === 'choice'"
            :lesson="lesson"
            :exercise-id="turns[v.index].exercise_id!"
            :prompt="byID[turns[v.index].exercise_id!].prompt"
            :options="byID[turns[v.index].exercise_id!].options ?? []"
            :explain="byID[turns[v.index].exercise_id!].explain"
            @graded="(ok, res, ans) => onGraded(turns[v.index].exercise_id!, ok, res, ans)"
          />

          <TextAnswer
            v-else-if="byID[turns[v.index].exercise_id!]"
            :lesson="lesson"
            :exercise-id="turns[v.index].exercise_id!"
            :type="byID[turns[v.index].exercise_id!].type as 'translate' | 'fill_blank'"
            :prompt="byID[turns[v.index].exercise_id!].prompt"
            :explain="byID[turns[v.index].exercise_id!].explain"
            @graded="(ok, res, ans) => onGraded(turns[v.index].exercise_id!, ok, res, ans)"
          />
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* The page scrolls to a new line so that it ends up clear of the fixed bar the
   answer fields bring with them. */
.msg {
  scroll-margin-bottom: 9rem;
}

/* A line coming in: it rises out of where the speaker's bubble sits, so the
   two sides of the conversation do not arrive from the same place.

   Two things here are deliberate. The animation holds no end state
   (`backwards`, not `both`): an element that carries a transform animation, even
   one that has finished and is only filling, becomes the containing block of any
   `position: fixed` child, and the pinned "Проверить" bar would then stick to the
   line instead of the bottom of the screen. And the wrapper of an open question,
   which contains exactly that bar, fades only — it never gets a transform. */
.msg-in {
  animation: msg-in 0.45s cubic-bezier(0.2, 0.8, 0.2, 1) backwards;
}
.msg-in.npc {
  transform-origin: 0% 100%;
}
.msg-in.me {
  transform-origin: 100% 100%;
}
.msg-fade {
  animation: msg-fade 0.4s ease backwards;
}
@keyframes msg-in {
  from {
    opacity: 0;
    transform: translateY(14px) scale(0.96);
  }
}
@keyframes msg-fade {
  from {
    opacity: 0;
  }
}
@media (prefers-reduced-motion: reduce) {
  .msg-in,
  .msg-fade {
    animation: none;
  }
}
</style>
