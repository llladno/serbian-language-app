<script setup lang="ts">
// A dialogue step: the conversation grows downwards, one turn at a time. A
// "me" turn shows its exercise; once answered — right or wrong — the canonical
// line takes its place as a bubble, so the thread of the conversation never
// breaks.
import { computed, reactive, ref } from 'vue'
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
const emit = defineEmits<{ graded: [exerciseId: string, ok: boolean] }>()

const showTranslations = ref(localStorage.getItem('dialogue.translations') === '1')
function toggleTranslations() {
  showTranslations.value = !showTranslations.value
  localStorage.setItem('dialogue.translations', showTranslations.value ? '1' : '0')
}

// Autoplay is opt-in: mobile browsers mute audio without a user gesture, and
// an unexpected voice in a quiet room is worse than a missing one.
const autoplay = ref(localStorage.getItem('dialogue.autoplay') === '1')
function toggleAutoplay() {
  autoplay.value = !autoplay.value
  localStorage.setItem('dialogue.autoplay', autoplay.value ? '1' : '0')
}
function played(clip?: string) {
  if (!autoplay.value || !clip) return
  new Audio(`/audio/${clip}`).play().catch(() => {})
}

const turns = computed(() => props.step.turns ?? [])
const byID = computed(() => Object.fromEntries(props.exercises.map((e) => [e.id, e])))

// Lines learned during this session, keyed by exercise id: the server sends
// them with the check result. Turns answered in an earlier session already
// carry their sr/ru in the lesson payload.
const answered = reactive<Record<string, CheckResult>>({})

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

function onGraded(exerciseId: string, ok: boolean, result: CheckResult) {
  answered[exerciseId] = result
  emit('graded', exerciseId, ok)
}

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
    note: res?.explain,
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
          class="rounded-full border border-[var(--border)] px-3 py-1.5 text-xs text-[var(--muted)] transition hover:border-[var(--accent)] hover:text-[var(--accent)]"
          :class="{ 'border-[var(--accent)] text-[var(--accent)]': showTranslations }"
          @click="toggleTranslations"
        >
          переводы
        </button>
        <button
          type="button"
          data-test="toggle-autoplay"
          class="rounded-full border border-[var(--border)] px-3 py-1.5 text-xs text-[var(--muted)] transition hover:border-[var(--accent)] hover:text-[var(--accent)]"
          :class="{ 'border-[var(--accent)] text-[var(--accent)]': autoplay }"
          @click="toggleAutoplay"
        >
          звук
        </button>
      </div>
    </header>

    <div class="space-y-3">
      <template v-for="v in visible" :key="v.index">
        <DialogueBubble
          v-if="turns[v.index].who === 'npc'"
          who="npc"
          :sr="turns[v.index].sr ?? ''"
          :ru="turns[v.index].ru"
          :audio="turns[v.index].audio"
          :show-translation="showTranslations"
          @vue:mounted="played(turns[v.index].audio)"
        />

        <template v-else>
          <DialogueBubble
            v-if="!v.active"
            who="me"
            :sr="lineOf(turns[v.index]).sr"
            :ru="lineOf(turns[v.index]).ru"
            :audio="lineOf(turns[v.index]).audio"
            :wrong="lineOf(turns[v.index]).wrong"
            :note="lineOf(turns[v.index]).note"
            :show-translation="showTranslations"
          />

          <ChoiceAnswer
            v-else-if="byID[turns[v.index].exercise_id!]?.type === 'choice'"
            :lesson="lesson"
            :exercise-id="turns[v.index].exercise_id!"
            :prompt="byID[turns[v.index].exercise_id!].prompt"
            :options="byID[turns[v.index].exercise_id!].options ?? []"
            @graded="(ok, res) => onGraded(turns[v.index].exercise_id!, ok, res)"
          />

          <TextAnswer
            v-else-if="byID[turns[v.index].exercise_id!]"
            :lesson="lesson"
            :exercise-id="turns[v.index].exercise_id!"
            :type="byID[turns[v.index].exercise_id!].type as 'translate' | 'fill_blank'"
            :prompt="byID[turns[v.index].exercise_id!].prompt"
            @graded="(ok, res) => onGraded(turns[v.index].exercise_id!, ok, res)"
          />
        </template>
      </template>
    </div>
  </section>
</template>
