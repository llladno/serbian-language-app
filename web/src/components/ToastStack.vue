<script setup lang="ts">
// Mounted once, in AppNav. Hangs below the header rather than above the tab
// bar — the bottom of a phone screen is already the navigation — and clears
// the header's own height so a toast about a reward does not cover the balance
// it just changed.
import { RouterLink } from 'vue-router'
import { useToasts } from '../lib/toasts'
import FeatherIcon from './FeatherIcon.vue'

const { items, dismiss } = useToasts()
</script>

<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed inset-x-0 top-0 z-40 flex flex-col items-center gap-2 px-3"
      style="padding-top: calc(env(safe-area-inset-top) + 3.75rem)"
    >
      <TransitionGroup name="toast">
        <RouterLink
          v-for="t in items"
          :key="t.id"
          :to="t.to ?? ''"
          class="card pointer-events-auto flex w-full max-w-sm items-center gap-3 p-3 text-left"
          data-test="toast"
          @click="dismiss(t.id)"
        >
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-bold">{{ t.title }}</p>
            <p class="truncate text-sm text-[var(--muted)]">{{ t.text }}</p>
          </div>
          <span v-if="t.reward" class="flex shrink-0 items-center gap-1 font-extrabold text-[var(--accent)]">
            +{{ t.reward }}<FeatherIcon :size="15" />
          </span>
        </RouterLink>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.22s ease, transform 0.22s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
.toast-move {
  transition: transform 0.22s ease;
}
</style>
