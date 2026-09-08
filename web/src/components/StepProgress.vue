<script setup lang="ts">
import type { Step } from '../types'

defineProps<{ steps: Step[]; current: number }>()
</script>

<template>
  <div class="flex items-center gap-1">
    <span
      v-for="(s, i) in steps"
      :key="s.id"
      class="h-1.5 flex-1 rounded-full transition-colors duration-300"
      :class="
        s.status === 'done'
          ? 'bg-[var(--accent)]'
          : i === current
            ? 'bg-[color-mix(in_srgb,var(--accent)_45%,transparent)]'
            : 'bg-[var(--ring-track)]'
      "
    />
  </div>
</template>
