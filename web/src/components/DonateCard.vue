<script setup lang="ts">
import { Heart } from 'lucide-vue-next'
import { useDonateModal, TRIBUTE_TELEGRAM_LINK } from '../lib/donateModal'
import { isTelegram } from '../telegram'
import heartIcon from '../assets/donate/heart.png'

const { openModal } = useDonateModal()

function onClick() {
  if (isTelegram()) {
    window.open(TRIBUTE_TELEGRAM_LINK, '_blank')
  } else {
    openModal()
  }
}
</script>

<template>
  <div class="card flex flex-col items-center gap-3 p-5 text-center sm:flex-row sm:text-left">
    <img :src="heartIcon" alt="" class="h-16 w-16 shrink-0 object-contain" />
    <div class="flex-1">
      <p class="font-extrabold">Поддержать проект</p>
      <p class="text-sm text-[var(--muted)]">Помогите развитию курса — новые уроки, слова и улучшения.</p>
    </div>
    <button class="btn btn-primary flex shrink-0 items-center gap-1.5" data-test="open-donate" @click="onClick">
      <Heart :size="17" :stroke-width="2.25" />
      Podrži
    </button>
  </div>
</template>
