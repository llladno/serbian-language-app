<script setup lang="ts">
const props = defineProps<{ current: number; total: number }>()
const pct = () => (props.total > 0 ? Math.round((props.current / props.total) * 100) : 0)
</script>

<template>
  <div class="flex items-center gap-2 text-xs text-[var(--muted)]">
    <div class="h-1.5 flex-1 rounded-full bg-[var(--border)]">
      <div class="h-full rounded-full bg-[var(--accent)] transition-all" :style="{ width: pct() + '%' }" />
    </div>
    <span class="tabular-nums">{{ current }} / {{ total }}</span>
  </div>
</template>
