<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ activity: { date: string; count: number }[]; weeks?: number }>()

const WEEKS = props.weeks ?? 13

function iso(d: Date) {
  return d.toISOString().slice(0, 10)
}

const cells = computed(() => {
  const map = new Map(props.activity.map((a) => [a.date, a.count]))
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  // start on the Monday WEEKS-1 weeks back
  const start = new Date(today)
  const dow = (start.getDay() + 6) % 7 // Mon=0
  start.setDate(start.getDate() - dow - (WEEKS - 1) * 7)

  const grid: { date: string; count: number; future: boolean }[][] = []
  for (let w = 0; w < WEEKS; w++) {
    const col: { date: string; count: number; future: boolean }[] = []
    for (let d = 0; d < 7; d++) {
      const cur = new Date(start)
      cur.setDate(start.getDate() + w * 7 + d)
      const key = iso(cur)
      col.push({ date: key, count: map.get(key) ?? 0, future: cur > today })
    }
    grid.push(col)
  }
  return grid
})

function level(count: number) {
  if (count === 0) return 0
  if (count < 5) return 1
  if (count < 12) return 2
  if (count < 25) return 3
  return 4
}
const COLORS = [
  'var(--ring-track)',
  'color-mix(in srgb, var(--accent) 30%, var(--ring-track))',
  'color-mix(in srgb, var(--accent) 55%, transparent)',
  'color-mix(in srgb, var(--accent) 78%, transparent)',
  'var(--accent)',
]
</script>

<template>
  <div class="flex gap-[3px] overflow-x-auto">
    <div v-for="(col, i) in cells" :key="i" class="flex flex-col gap-[3px]">
      <div
        v-for="cell in col"
        :key="cell.date"
        class="h-[11px] w-[11px] rounded-[3px]"
        :style="{ background: cell.future ? 'transparent' : COLORS[level(cell.count)] }"
        :title="cell.future ? '' : `${cell.date}: ${cell.count}`"
      />
    </div>
  </div>
</template>
