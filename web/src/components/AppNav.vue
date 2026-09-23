<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router'
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  GraduationCap,
  Headphones,
  Languages,
  MonitorSmartphone,
  Moon,
  Repeat,
  SunMedium,
  Trophy,
  User,
} from 'lucide-vue-next'
import { theme, cycleTheme, THEME_META } from '../theme'
import { useSupportModal } from '../lib/supportModal'
import SupportModal from './SupportModal.vue'
import NotificationBell from './NotificationBell.vue'

const { openModal } = useSupportModal()

defineProps<{ name: string; hideTabBar?: boolean }>()

const route = useRoute()

// iOS Safari's collapsing address bar leaves window.innerHeight taller than
// what's actually on screen (visualViewport.height) while it's animating -
// a `position: fixed; bottom: 0` element resolves against the former, so it
// visibly floats below the real viewport edge until the two resync. Track
// the gap and cancel it out with a transform tied to the real visual
// viewport instead.
const tabBar = ref<HTMLElement | null>(null)

function syncTabBarOffset() {
  const vv = window.visualViewport
  if (!vv || !tabBar.value) return
  const offset = Math.max(0, window.innerHeight - vv.height - vv.offsetTop)
  tabBar.value.style.transform = offset ? `translateY(-${offset}px)` : ''
}

onMounted(() => {
  syncTabBarOffset()
  window.visualViewport?.addEventListener('resize', syncTabBarOffset)
  window.visualViewport?.addEventListener('scroll', syncTabBarOffset)
})
onBeforeUnmount(() => {
  window.visualViewport?.removeEventListener('resize', syncTabBarOffset)
  window.visualViewport?.removeEventListener('scroll', syncTabBarOffset)
})

const links = [
  { to: '/profile', label: 'Профиль', icon: User },
  { to: '/course', label: 'Курс', icon: GraduationCap },
  { to: '/review', label: 'Слова', icon: Repeat },
  { to: '/vocab', label: 'Словарь', icon: Languages },
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
  <header class="sticky top-0 z-20 bg-[var(--bg)]" style="padding-top: env(safe-area-inset-top)">
    <div class="mx-auto flex max-w-3xl items-center gap-1 px-3 py-2">
      <nav class="hidden gap-1 sm:flex">
        <RouterLink
          v-for="l in links"
          :key="l.to"
          :to="l.to"
          class="flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-semibold transition"
          :class="
            isActive(l.to)
              ? 'bg-[var(--accent)] text-white'
              : 'text-[var(--muted)] hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]'
          "
        >
          <component :is="l.icon" :size="15" :stroke-width="2.25" />{{ l.label }}
        </RouterLink>
      </nav>

      <span class="serbian text-lg font-semibold sm:hidden">ucimo</span>

      <button
        class="ml-auto flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
        :title="`Тема: ${themeMeta.label}`"
        @click="cycleTheme()"
      >
        <component :is="themeIcon" :size="17" :stroke-width="2.25" />
      </button>

      <NotificationBell />

      <button
        class="flex h-11 w-11 shrink-0 items-center justify-center rounded-full text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
        title="Поддержка"
        aria-label="Поддержка"
        data-test="open-support"
        @click="openModal"
      >
        <Headphones :size="17" :stroke-width="2.25" />
      </button>

      <RouterLink
        to="/profile"
        class="flex h-9 shrink-0 items-center gap-1.5 rounded-full px-2.5 text-sm text-[var(--muted)] transition hover:bg-[var(--bg-soft)] hover:text-[var(--fg)]"
        title="Профиль"
      >
        <span class="max-w-[7rem] truncate font-semibold text-[var(--fg)]">{{ name }}</span>
      </RouterLink>
    </div>
  </header>

  <nav
    v-if="!hideTabBar"
    ref="tabBar"
    class="fixed inset-x-3 z-20 flex justify-between gap-0.5 rounded-[28px] bg-[var(--card)] p-1.5 shadow-[var(--shadow)] transition-transform duration-150 ease-out sm:hidden"
    style="bottom: max(0.75rem, env(safe-area-inset-bottom))"
  >
    <RouterLink
      v-for="l in links"
      :key="l.to"
      :to="l.to"
      class="flex flex-1 flex-col items-center gap-0.5 rounded-3xl py-2 text-[10px] font-medium transition"
      :class="isActive(l.to) ? 'bg-[var(--accent)] text-white' : 'text-[var(--muted)]'"
    >
      <component :is="l.icon" :size="18" :stroke-width="2.25" />{{ l.label }}
    </RouterLink>
  </nav>

  <SupportModal />
</template>
