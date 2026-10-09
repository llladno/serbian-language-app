<script setup lang="ts">
// The currency glyph. Decorative everywhere it appears — the amount next to it
// always carries the meaning — so it stays out of the accessibility tree.
import { SEED } from '../lib/seeds'

// `size` is the width of a slot the icon used to fill; the seed is taller than
// it is wide, so it is sized by height to keep the same visual weight next to
// text.
withDefaults(defineProps<{ size?: number }>(), { size: 18 })
</script>

<template>
  <img :src="SEED" alt="" aria-hidden="true" class="seed" :style="{ height: Math.round(size * 1.2) + 'px' }" />
</template>

<style scoped>
.seed {
  image-rendering: pixelated;
  display: block;
  flex: none;
}
</style>
