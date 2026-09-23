<script setup lang="ts">
defineProps<{ title?: string; mascot?: string }>()
</script>

<template>
  <div class="mx-auto mt-10 max-w-sm px-4">
    <img v-if="mascot" :src="mascot" alt="" class="mx-auto h-28 w-28 object-contain" />
    <p class="serbian text-center text-4xl font-semibold">ucimo</p>
    <p v-if="title" class="mt-1 text-center text-[var(--muted)]">{{ title }}</p>
    <div class="mt-6">
      <slot />
    </div>
  </div>
</template>
