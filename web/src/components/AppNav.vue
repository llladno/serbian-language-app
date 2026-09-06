<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { computed } from 'vue'

const route = useRoute()

const links = [
  { to: '/', label: 'Главная', icon: '◈' },
  { to: '/course', label: 'Курс', icon: '≣' },
  { to: '/review', label: 'Слова', icon: '✦' },
  { to: '/vocab', label: 'Словарь', icon: '⌕' },
  { to: '/false-friends', label: 'Ловушки', icon: '⚠' },
]

const active = computed(() => route.path)
function isActive(to: string) {
  return to === '/' ? active.value === '/' : active.value.startsWith(to)
}
</script>

<template>
  <nav class="sticky top-0 z-20 border-b border-[var(--border)] bg-[var(--bg)]/85 backdrop-blur">
    <div class="mx-auto flex max-w-3xl gap-1 overflow-x-auto px-3 py-2">
      <RouterLink
        v-for="l in links"
        :key="l.to"
        :to="l.to"
        class="flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold transition"
        :class="
          isActive(l.to)
            ? 'bg-[var(--accent-soft)] text-[var(--accent)]'
            : 'text-[var(--muted)] hover:text-[var(--fg)]'
        "
      >
        <span class="text-xs opacity-70">{{ l.icon }}</span>{{ l.label }}
      </RouterLink>
    </div>
  </nav>
</template>
