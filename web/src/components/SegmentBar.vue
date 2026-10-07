<script setup lang="ts">
// Progress as a row of hard-edged blocks rather than a smooth bar: the rest of
// the app is rounded and soft, and the quests screen is the one place where the
// pixel-art side of the brand gets to speak. Blocks also read at a glance on a
// phone — six of twelve is countable, 60% of a gradient is not.
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    value: number
    max: number
    /** 'accent' while working towards it, 'feather' once it is worth money. */
    tone?: 'accent' | 'feather'
    segments?: number
  }>(),
  { tone: 'accent', segments: 12 },
)

// Never full unless it really is done, and never empty unless nothing has been
// done: a bar that reads "finished" one answer early is worse than a coarse one.
const filled = computed(() => {
  if (props.max <= 0 || props.value <= 0) return 0
  if (props.value >= props.max) return props.segments
  const exact = Math.floor((props.value / props.max) * props.segments)
  return Math.min(props.segments - 1, Math.max(1, exact))
})
</script>

<template>
  <div
    class="flex gap-[2px]"
    role="progressbar"
    :aria-valuenow="value"
    :aria-valuemin="0"
    :aria-valuemax="max"
  >
    <span
      v-for="i in segments"
      :key="i"
      class="block h-2.5 flex-1"
      :class="[i <= filled ? (tone === 'feather' ? 'seg-feather' : 'seg-accent') : 'seg-empty']"
    />
  </div>
</template>

<style scoped>
.seg-accent {
  background: var(--accent);
}
.seg-feather {
  background: var(--feather);
}
.seg-empty {
  background: var(--ring-track);
}
</style>
