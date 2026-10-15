<script setup lang="ts">
// The way into the tariffs screen, for the accounts that have it. Nothing in
// the main navigation points there while tariffs are closed to everyone else.
import { Sprout } from 'lucide-vue-next'
import { useTariffsStore } from '../stores/tariffs'

const tariffs = useTariffsStore()
</script>

<template>
  <div v-if="tariffs.enabled" class="card flex flex-col items-center gap-3 p-5 text-center sm:flex-row sm:text-left" data-test="tariffs-card">
    <div class="flex-1">
      <p class="font-extrabold">Тарифы</p>
      <p class="text-sm text-[var(--muted)]">Открыть уровни сразу, без накопления зёрнышек.</p>
    </div>
    <RouterLink to="/tariffs" class="btn btn-primary flex shrink-0 items-center gap-1.5" data-test="open-tariffs">
      <Sprout :size="17" :stroke-width="2.25" />
      Смотреть
    </RouterLink>
  </div>
</template>
