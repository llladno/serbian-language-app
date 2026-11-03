<script setup lang="ts">
// The picture of a story: one frame per scene, cross-faded when the scene changes, each with a slow zoom while it is
// on the screen. Plain CSS, no video: the frames are generated once and weigh little. It fills the box it is put in
// (the parent gives the 4:3 shape and lets it shrink on a short screen; the picture is cropped, not squeezed).
import type { StoryScene } from '../types'

defineProps<{ scenes: StoryScene[]; active: number }>()
</script>

<template>
  <div class="story-frames" data-test="story-frames">
    <img
      v-for="sc in scenes"
      :key="sc.id"
      :src="sc.image"
      alt=""
      class="story-frame"
      :class="{ 'story-frame-on': sc.id === active }"
      draggable="false"
    />
  </div>
</template>

<style scoped>
.story-frames {
  position: absolute;
  inset: 0;
  overflow: hidden;
  border-radius: 1rem;
  background: var(--bg-soft);
}
.story-frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  image-rendering: pixelated;
  opacity: 0;
  transform: scale(1);
  transition: opacity 0.9s ease;
}
.story-frame-on {
  opacity: 1;
  animation: story-zoom 16s ease-out forwards;
}
@keyframes story-zoom {
  from {
    transform: scale(1);
  }
  to {
    transform: scale(1.07);
  }
}
@media (prefers-reduced-motion: reduce) {
  .story-frame {
    transition: none;
  }
  .story-frame-on {
    animation: none;
  }
}
</style>
