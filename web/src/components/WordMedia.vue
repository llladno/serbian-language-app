<script setup lang="ts">
// Shows a curated photo, or an emoji, or nothing.
withDefaults(defineProps<{ image?: string; emoji?: string; size?: number; alt?: string }>(), {
  size: 96,
})
</script>

<template>
  <img
    v-if="image"
    :src="'/img/' + image"
    :alt="alt || ''"
    loading="lazy"
    class="rounded-xl object-cover"
    :style="{ width: size + 'px', height: size + 'px' }"
  />
  <span
    v-else-if="emoji"
    class="grid place-items-center"
    :style="{ width: size + 'px', height: size + 'px', fontSize: Math.round(size * 0.55) + 'px' }"
    >{{ emoji }}</span
  >
</template>
