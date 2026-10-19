<script setup lang="ts">
import { X } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import { useSurveyModal, RATINGS, LIKES, PAINS, PURPOSES, WISHES, STEPS } from '../lib/surveyModal'
import { useWalletStore } from '../stores/wallet'
import supportMascot from '../assets/support-mascot.webp'

const s = useSurveyModal()
const wallet = useWalletStore()
</script>

<template>
  <ModalShell :open="s.open.value" panel-class="p-5" @close="s.dismiss">
    <button class="icon-btn absolute right-3 top-3" title="Закрыть" aria-label="Закрыть" data-test="survey-close" @click="s.dismiss">
      <X :size="19" :stroke-width="2.25" />
    </button>

    <div v-if="s.step.value === 0" class="space-y-3 text-center" data-test="survey-intro">
      <img :src="supportMascot" alt="" class="mx-auto h-24 w-24 object-contain" />
      <p class="text-lg font-extrabold">Ćao! 👋 (Привет!)</p>
      <p class="text-sm text-[var(--muted)]">
        Мы делаем ucimo вместе с вами, и нам правда важно знать, как у вас идёт. Пять коротких вопросов, минута
        времени — и мы положим вам в копилку
        <b class="text-[var(--accent)]">+{{ wallet.amount(s.reward.value) }}</b> 🌻
      </p>
      <p class="text-xs text-[var(--muted)]">Награда — за ответы на все вопросы. Если закроете окно, опрос больше не появится.</p>
      <button class="btn btn-primary w-full" data-test="survey-start" @click="s.step.value = 1">Давайте</button>
    </div>

    <div v-else class="space-y-4">
      <div class="pr-8">
        <p class="text-xs font-semibold text-[var(--muted)]">Вопрос {{ s.step.value }} из {{ STEPS }}</p>
        <div class="mt-1 h-1.5 overflow-hidden rounded-full bg-[var(--line,#0001)]">
          <div class="h-full rounded-full bg-[var(--accent)] transition-all" :style="{ width: (s.step.value / STEPS) * 100 + '%' }" />
        </div>
      </div>

      <div v-if="s.step.value === 1" data-test="survey-q1">
        <p class="text-lg font-extrabold">Как тебе ucimo в целом?</p>
        <p class="mt-1 text-sm text-[var(--muted)]">Честно, нам можно любую оценку 🙂</p>
        <div class="mt-3 flex justify-between gap-1">
          <button
            v-for="(emoji, i) in RATINGS"
            :key="i"
            class="flex-1 rounded-xl border py-2 text-2xl transition"
            :class="s.rating.value === i + 1 ? 'border-[var(--accent)] bg-[var(--accent)]/10' : 'border-transparent opacity-70'"
            :data-test="`rating-${i + 1}`"
            @click="s.rating.value = i + 1"
          >{{ emoji }}</button>
        </div>
      </div>

      <div v-else-if="s.step.value === 2" data-test="survey-q2">
        <p class="text-lg font-extrabold">Что тебе нравится больше всего?</p>
        <p class="mt-1 text-sm text-[var(--muted)]">Можно выбрать несколько.</p>
        <div class="mt-3 flex flex-wrap gap-2">
          <button
            v-for="o in LIKES" :key="o.code" class="chip" :class="{ 'chip-on': s.likes.value.includes(o.code) }"
            :data-test="`like-${o.code}`" @click="s.toggleLike(o.code)"
          >{{ o.label }}</button>
        </div>
      </div>

      <div v-else-if="s.step.value === 3" data-test="survey-q3">
        <p class="text-lg font-extrabold">Что мешает или немного бесит?</p>
        <p class="mt-1 text-sm text-[var(--muted)]">Это самое ценное для нас — говори как есть.</p>
        <div class="mt-3 flex flex-wrap gap-2">
          <button
            v-for="o in PAINS" :key="o.code" class="chip" :class="{ 'chip-on': s.pains.value.includes(o.code) }"
            :data-test="`pain-${o.code}`" @click="s.togglePain(o.code)"
          >{{ o.label }}</button>
        </div>
        <textarea
          v-model="s.painText.value" class="field mt-3 w-full" rows="3" maxlength="1000"
          placeholder="Расскажи своими словами (необязательно)" data-test="pain-text"
        />
      </div>

      <div v-else-if="s.step.value === 4" data-test="survey-q4">
        <p class="text-lg font-extrabold">Зачем ты учишь сербский?</p>
        <div class="mt-3 flex flex-col gap-2">
          <button
            v-for="o in PURPOSES" :key="o.code" class="chip text-left" :class="{ 'chip-on': s.purpose.value === o.code }"
            :data-test="`purpose-${o.code}`" @click="s.purpose.value = o.code"
          >{{ o.label }}</button>
        </div>
      </div>

      <div v-else data-test="survey-q5">
        <p class="text-lg font-extrabold">Чего тебе больше всего не хватает?</p>
        <p class="mt-1 text-sm text-[var(--muted)]">Выбери одно — то, что важнее всего. Так мы поймём, что делать дальше.</p>
        <div class="mt-3 flex flex-col gap-2">
          <button
            v-for="o in WISHES" :key="o.code" class="chip text-left" :class="{ 'chip-on': s.wish.value === o.code }"
            :data-test="`wish-${o.code}`" @click="s.wish.value = o.code"
          >{{ o.label }}</button>
        </div>
      </div>

      <p v-if="s.error.value" class="text-sm text-[var(--bad)]">{{ s.error.value }}</p>

      <div class="flex gap-2">
        <button class="btn btn-ghost" data-test="survey-back" @click="s.step.value--">Назад</button>
        <button
          v-if="s.step.value < STEPS" class="btn btn-primary flex-1" data-test="survey-next"
          :disabled="!s.canNext.value" @click="s.step.value++"
        >Дальше</button>
        <button
          v-else class="btn btn-primary flex-1" data-test="survey-submit"
          :disabled="!s.canNext.value || s.busy.value" @click="s.submit"
        >{{ s.busy.value ? 'Отправляем…' : `Отправить и забрать +${wallet.amount(s.reward.value)}` }}</button>
      </div>
    </div>
  </ModalShell>
</template>

<style scoped>
.chip {
  display: inline-flex;
  align-items: center;
  border: 1.5px solid var(--border, #0002);
  border-radius: 999px;
  padding: 0.45rem 0.85rem;
  font-size: 0.875rem;
  font-weight: 600;
  transition: all 0.15s ease;
}
.chip-on {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 12%, transparent);
  color: var(--accent);
}
</style>
