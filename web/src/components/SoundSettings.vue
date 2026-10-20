<script setup lang="ts">
// The two switches for the app's own sounds (lib/sound.ts). They live on this
// device, not in the account: sound that is welcome on a laptop at home is not
// on a phone in a queue. The vibration is a Telegram thing, so outside the
// Mini App there is nothing to switch.
import { hapticsOn, setHaptics, setSound, soundOn } from '../lib/sound'
import { inMiniApp } from '../telegram'
</script>

<template>
  <div data-test="sound-settings">
    <label class="flex cursor-pointer items-center gap-3 rounded-[14px] px-1 py-2">
      <span class="min-w-0 flex-1">
        <span class="block text-sm font-semibold">Звук</span>
        <span class="block text-xs text-[var(--muted)]">
          Аккорд в конце урока и озвучка верного предложения после каждого задания
        </span>
      </span>
      <input
        type="checkbox"
        role="switch"
        class="switch"
        aria-label="Звук"
        data-test="setting-sound"
        :checked="soundOn"
        @change="setSound(($event.target as HTMLInputElement).checked)"
      />
    </label>

    <label v-if="inMiniApp()" class="flex cursor-pointer items-center gap-3 rounded-[14px] px-1 py-2">
      <span class="min-w-0 flex-1">
        <span class="block text-sm font-semibold">Вибрация</span>
        <span class="block text-xs text-[var(--muted)]">Короткий отклик телефона, когда урок пройден</span>
      </span>
      <input
        type="checkbox"
        role="switch"
        class="switch"
        aria-label="Вибрация"
        data-test="setting-haptics"
        :checked="hapticsOn"
        @change="setHaptics(($event.target as HTMLInputElement).checked)"
      />
    </label>
  </div>
</template>
