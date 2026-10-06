<script setup lang="ts">
// The currency glyph. Decorative everywhere it appears — the amount next to it
// always carries the meaning — so it stays out of the accessibility tree.
import { FEATHER } from '../lib/feathers'

withDefaults(defineProps<{ size?: number }>(), { size: 18 })
</script>

<template>
  <img :src="FEATHER" alt="" aria-hidden="true" class="feather" :style="{ width: size + 'px' }" />
</template>

<style scoped>
.feather {
  image-rendering: pixelated;
  display: block;
  flex: none;
}
</style>
