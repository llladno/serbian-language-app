<script setup lang="ts">
import { Headphones } from 'lucide-vue-next'
import { useSupportModal } from '../lib/supportModal'
import supportMascot from '../assets/support-mascot.webp'

const { openModal } = useSupportModal()
</script>

<template>
  <div class="card flex flex-col items-center gap-3 p-5 text-center sm:flex-row sm:text-left">
    <img :src="supportMascot" alt="" class="h-16 w-16 shrink-0 object-contain" />
    <div class="flex-1">
      <p class="font-extrabold">Есть вопрос или пожелание?</p>
      <p class="text-sm text-[var(--muted)]">
        Пишите в любой момент — про ошибки, идеи или просто чтобы поздороваться.
      </p>
    </div>
    <button class="btn btn-primary flex shrink-0 items-center gap-1.5" data-test="open-support" @click="openModal">
      <Headphones :size="17" :stroke-width="2.25" />
      Поддержка
    </button>
  </div>
</template>
