<script setup lang="ts">
// «Соедините пары»: a Serbian phrase on the left, its translation on the right; tap one, then the other, like the
// lesson's match. A right answer is replaced by a big «Верно»; a wrong one shows the right pairs.
import { computed, reactive, ref } from 'vue'
import { useScript } from '../lib/useScript'
import { shuffled } from '../lib/shuffle'
import { useStoryVerdict } from '../lib/useStoryVerdict'
import type { StoryVerdict as Verdict } from '../types'
import StoryVerdict from './StoryVerdict.vue'

const props = defineProps<{
  prompt: string
  eyebrow?: string
  left: string[]
  right: string[]
  check: (pairs: Record<string, string>) => Promise<Verdict>
}>()
const emit = defineEmits<{ done: [correct: boolean] }>()
const { show } = useScript()
const { verdict, lockedHeight, setControls, settle } = useStoryVerdict((ok) => emit('done', ok))

const choices = ref(shuffled(props.right))
const picks = reactive<Record<string, string>>({})
const selectedLeft = ref<string | null>(null)
const selectedRight = ref<string | null>(null)
const pending = ref(false)

const complete = computed(() => props.left.every((l) => l in picks))
const isRightUsed = (r: string) => Object.values(picks).includes(r)
const leftFor = (r: string) => props.left.find((l) => picks[l] === r)

function tapLeft(l: string) {
  if (verdict.value) return
  if (l in picks) {
    delete picks[l]
    return
  }
  if (selectedRight.value) {
    picks[l] = selectedRight.value
    selectedRight.value = null
    return
  }
  selectedLeft.value = selectedLeft.value === l ? null : l
}
function tapRight(r: string) {
  if (verdict.value) return
  if (isRightUsed(r)) {
    const l = leftFor(r)
    if (l) delete picks[l]
    return
  }
  if (selectedLeft.value) {
    picks[selectedLeft.value] = r
    selectedLeft.value = null
    return
  }
  selectedRight.value = selectedRight.value === r ? null : r
}

const PAIRED = 'border border-[var(--accent)] bg-[var(--accent-soft)] text-[var(--accent)]'
function leftClass(l: string) {
  if (l in picks) return PAIRED
  return selectedLeft.value === l ? 'btn-ghost ring-2 ring-[var(--accent)]' : 'btn-ghost'
}
function rightClass(r: string) {
  if (isRightUsed(r)) return PAIRED
  return selectedRight.value === r ? 'btn-ghost ring-2 ring-[var(--accent)]' : 'btn-ghost'
}

async function submit() {
  if (pending.value || verdict.value || !complete.value) return
  pending.value = true
  let v: Verdict
  try {
    v = await props.check({ ...picks })
  } catch {
    v = { correct: false, answer: '' }
  }
  settle(v)
}
</script>

<template>
  <div class="text-center" data-test="story-match">
    <p v-if="eyebrow" class="mb-1 text-xs font-semibold text-[var(--muted)]">{{ eyebrow }}</p>
    <p class="mb-3 text-xl font-medium leading-snug">{{ prompt }}</p>

    <Transition name="verdict-swap" mode="out-in">
      <div
        v-if="verdict?.correct"
        key="ok"
        class="flex items-center justify-center"
        :style="{ minHeight: lockedHeight + 'px' }"
      >
        <StoryVerdict />
      </div>

      <div v-else key="ask" :ref="setControls">
        <div v-if="verdict && !verdict.correct" class="pop mb-3 text-sm">
          <p class="mb-1 font-semibold text-[var(--bad)]">✗ Есть ошибки</p>
          <p v-for="p in verdict.pairs" :key="p[0]">
            <span class="serbian font-semibold">{{ show(p[0]) }}</span> — {{ p[1] }}
          </p>
        </div>
        <template v-else>
          <div class="mb-3 grid grid-cols-2 gap-2">
            <div class="flex flex-col gap-2">
              <button
                v-for="l in left"
                :key="l"
                type="button"
                class="btn serbian w-full !px-2"
                :class="leftClass(l)"
                data-test="story-left"
                @click="tapLeft(l)"
              >
                {{ show(l) }}
              </button>
            </div>
            <div class="flex flex-col gap-2">
              <button
                v-for="r in choices"
                :key="r"
                type="button"
                class="btn w-full !px-2"
                :class="rightClass(r)"
                data-test="story-right"
                @click="tapRight(r)"
              >
                {{ r }}
              </button>
            </div>
          </div>
          <button type="button" class="btn btn-primary w-full" :disabled="pending || !complete" data-test="story-submit" @click="submit">
            Проверить
          </button>
        </template>
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
