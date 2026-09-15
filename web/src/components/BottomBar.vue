<script setup lang="ts">
// Shared "pinned to the bottom of the screen" action bar — the lesson's own
// Далее/Завершить bar and each answer component's own Проверить button use
// the exact same shell so only one is ever on screen and they read as one
// consistent control, not two different bars.
</script>

<template>
  <div class="fixed inset-x-0 bottom-0 z-20 border-t border-[var(--border)] bg-[var(--bg)]/90 backdrop-blur">
    <div
      class="mx-auto max-w-3xl px-4 py-3 sm:px-6"
      style="padding-bottom: calc(0.75rem + env(safe-area-inset-bottom))"
    >
      <slot />
    </div>
  </div>
</template>
