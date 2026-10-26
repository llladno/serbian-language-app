<script setup lang="ts">
// The way back into the tariffs screen for a learner who already has a tariff
// (a running year, or a level paid for in rubles): that is where "Мой тариф" is.
// Everybody else sees the advert (TariffsPromo) in the same place instead.
// Nothing in the main navigation points to the tariffs while they are closed to
// the other accounts, so a card for them would be a dead end.
import { computed } from 'vue'
import { Sprout } from 'lucide-vue-next'
import { useTariffsStore } from '../stores/tariffs'
import { shortDate } from '../lib/tariffs'

const tariffs = useTariffsStore()
const show = computed(() => tariffs.ready && tariffs.hasTariff)
</script>

<template>
  <div v-if="show" class="card flex flex-col items-center gap-3 p-5 text-center sm:flex-row sm:text-left" data-test="tariffs-card">
    <div class="flex-1">
      <p class="font-extrabold">Мой тариф</p>
      <p v-if="tariffs.yearActive && tariffs.yearUntil" class="text-sm text-[var(--muted)]">
        «Год» действует до {{ shortDate(tariffs.yearUntil) }}.
      </p>
      <p v-else class="text-sm text-[var(--muted)]">Уровень открыт за рубли. Подробности — на странице тарифов.</p>
    </div>
    <RouterLink to="/tariffs" class="btn btn-primary flex shrink-0 items-center gap-1.5" data-test="open-tariffs">
      <Sprout :size="17" :stroke-width="2.25" />
      Смотреть
    </RouterLink>
  </div>
</template>
