<script setup lang="ts">
// Shown instead of the batch when today's free «Повторить» is used up. The button in the profile is
// never blocked: it leads here, and the way out is a subscription. «Повторять без ограничений» is
// always there and always active (an account without tariffs is sent on to its profile by the router).
// The picture is a full-width scene, like the one in the payment thanks: replace
// assets/repeat-limit-scene.webp to change it (a wide 3:2 pixel-art image, see the note in CLAUDE.md).
import { X } from 'lucide-vue-next'
import { watch } from 'vue'
import { useRouter } from 'vue-router'
import ModalShell from './ModalShell.vue'
import scene from '../assets/repeat-limit-scene.webp'
import { trackTariff } from '../lib/tariffEvents'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const router = useRouter()

// The admin's «Воронка тарифов» counts the showings: each time the window opens.
watch(() => props.open, (isOpen) => isOpen && trackTariff('view', 'repeat_limit'), { immediate: true })

function toTariffs() {
  trackTariff('click', 'repeat_limit')
  emit('close')
  router.push('/tariffs')
}
</script>

<template>
  <ModalShell :open="open" size="sm" panel-class="overflow-hidden text-center" @close="emit('close')">
    <button class="icon-btn close absolute right-3 top-3 z-10" title="Закрыть" aria-label="Закрыть" @click="emit('close')">
      <X :size="19" :stroke-width="2.25" />
    </button>

    <div class="art" aria-hidden="true">
      <img :src="scene" alt="" data-test="repeat-limit-scene" />
    </div>

    <div class="space-y-4 px-6 pb-6 pt-1" data-test="repeat-limit">
      <div class="space-y-1.5">
        <p class="text-xl font-extrabold">Серия на сегодня пройдена</p>
        <p class="text-sm text-[var(--muted)]">В день доступна одна серия из 15 заданий. Новая появится завтра.</p>
      </div>
      <p class="rounded-2xl bg-[var(--bg-soft)] px-4 py-3 text-sm">
        Можете оформить подписку, чтобы использовать повторения без ограничений.
      </p>
      <div class="space-y-2">
        <button class="btn btn-primary w-full" data-test="repeat-limit-tariffs" @click="toTariffs">
          Повторять без ограничений
        </button>
        <button class="btn btn-ghost w-full" data-test="repeat-limit-close" @click="emit('close')">Понятно</button>
      </div>
    </div>
  </ModalShell>
</template>

<style scoped>
/* A wide picture across the top of the card: it melts into the card instead of ending in an edge. */
.art {
  height: 13rem;
  overflow: hidden;
  background: linear-gradient(135deg, var(--accent-soft), var(--card));
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  /* The picture is 1264 px wide and shown at about 380: the browser's smooth scaling keeps the pixel edges
     clean, hard "pixelated" scaling would tear them. The bird and the sign stay clear of the fade. */
  object-position: 50% 40%;
  mask-image: linear-gradient(to bottom, #000 88%, transparent);
  -webkit-mask-image: linear-gradient(to bottom, #000 88%, transparent);
}
/* The cross sits on the picture, which can be bright in a dark theme: give it its own plate. */
.close {
  background: color-mix(in srgb, var(--card) 72%, transparent);
  backdrop-filter: blur(4px);
}
</style>
