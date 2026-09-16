<script setup lang="ts">
// Shared "pinned to the bottom of the screen" action bar — the lesson's own
// Далее/Завершить bar and each answer component's own Проверить button use
// the exact same shell so only one is ever on screen and they read as one
// consistent control, not two different bars.
//
// Not teleported: this can render from deep inside LessonView's per-step
// Transition, and a `transform` on that ancestor while it's entering/leaving
// would give `position: fixed` a new containing block — pinning the bar to
// the animating step wrapper instead of the viewport (it visibly floats
// mid-page for the transition's duration). Fixed at the source instead: the
// step transition animates opacity only, no transform (see LessonView.vue).
</script>

<template>
  <div class="fixed inset-x-0 bottom-0 z-20 bg-[var(--bg)]">
    <div
      class="mx-auto max-w-3xl px-4 py-3 sm:px-6"
      style="padding-bottom: calc(0.75rem + env(safe-area-inset-bottom))"
    >
      <slot />
    </div>
  </div>
</template>
