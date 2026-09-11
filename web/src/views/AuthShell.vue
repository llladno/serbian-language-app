<script setup lang="ts">
defineProps<{ title?: string }>()
</script>

<template>
  <div class="mx-auto mt-16 max-w-sm px-4">
    <p class="serbian text-center text-4xl font-semibold">Српски</p>
    <p v-if="title" class="mt-1 text-center text-[var(--muted)]">{{ title }}</p>
    <div class="mt-6">
      <slot />
    </div>
  </div>
</template>
