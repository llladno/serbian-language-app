<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { computed } from 'vue'
import {
  GraduationCap,
  Languages,
  MonitorSmartphone,
  Moon,
  Repeat,
  SunMedium,
  TriangleAlert,
  Trophy,
  User,
} from 'lucide-vue-next'
import { theme, cycleTheme, THEME_META } from '../theme'

defineProps<{ name: string }>()

const route = useRoute()

const links = [
  { to: '/profile', label: 'Профиль', icon: User },
  { to: '/course', label: 'Курс', icon: GraduationCap },
  { to: '/review', label: 'Слова', icon: Repeat },
  { to: '/vocab', label: 'Словарь', icon: Languages },
  { to: '/false-friends', label: 'Ловушки', icon: TriangleAlert },
  { to: '/rating', label: 'Рейтинг', icon: Trophy },
]

const active = computed(() => route.path)
function isActive(to: string) {
  return active.value.startsWith(to)
}

const THEME_ICON = { system: MonitorSmartphone, light: SunMedium, dark: Moon }
const themeMeta = computed(() => THEME_META[theme.value])
const themeIcon = computed(() => THEME_ICON[theme.value])
</script>

<template>
  <header
    class="sticky top-0 z-20 border-b border-[var(--border)] bg-[var(--bg)]/85 backdrop-blur"
    style="padding-top: env(safe-area-inset-top)"
  >
    <div class="mx-auto flex max-w-3xl items-center gap-1 px-3 py-2">
      <nav class="hidden gap-1 sm:flex">
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
          <component :is="l.icon" :size="15" :stroke-width="2.25" />{{ l.label }}
        </RouterLink>
      </nav>

      <span class="serbian text-lg font-semibold sm:hidden">ucimo</span>

      <button
        class="ml-auto flex h-9 w-9 shrink-0 items-center justify-center rounded-lg text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
        :title="`Тема: ${themeMeta.label}`"
        @click="cycleTheme()"
      >
        <component :is="themeIcon" :size="17" :stroke-width="2.25" />
      </button>

      <RouterLink
        to="/profile"
        class="flex h-9 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm text-[var(--muted)] transition hover:text-[var(--fg)]"
        title="Профиль"
      >
        <span class="max-w-[7rem] truncate font-semibold text-[var(--fg)]">{{ name }}</span>
      </RouterLink>
    </div>
  </header>

  <nav
    class="fixed inset-x-0 bottom-0 z-20 grid grid-cols-6 border-t border-[var(--border)] bg-[var(--bg)]/95 backdrop-blur sm:hidden"
    style="padding-bottom: env(safe-area-inset-bottom)"
  >
    <RouterLink
      v-for="l in links"
      :key="l.to"
      :to="l.to"
      class="flex flex-col items-center gap-1 py-2 text-[10px] font-medium transition"
      :class="isActive(l.to) ? 'text-[var(--accent)]' : 'text-[var(--muted)]'"
    >
      <component :is="l.icon" :size="19" :stroke-width="2.25" />{{ l.label }}
    </RouterLink>
  </nav>
</template>
