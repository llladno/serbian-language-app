<script setup lang="ts">
// The invite link and the ways to pass it on. The link is the site's own
// /register?ref=CODE: it works from every messenger and on every signup path
// (email or Telegram), and does not depend on the bot being reachable.
import { computed, ref, watch } from 'vue'
import { Check, Copy, Send, Share2, X } from 'lucide-vue-next'
import ModalShell from './ModalShell.vue'
import SeedIcon from './SeedIcon.vue'
import inviteScene from '../assets/invite-scene.webp'
import { api } from '../api'
import { authErrorMessage } from '../lib/authErrors'
import { useInviteModal } from '../lib/inviteModal'
import { openTelegramLink } from '../telegram'
import type { Referral } from '../types'

const props = defineProps<{ reward: number }>()
const { open, closeModal } = useInviteModal()

const referral = ref<Referral | null>(null)
const error = ref('')
const loading = ref(false)
const copied = ref(false)
const input = ref<HTMLInputElement | null>(null)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const SHARE_TEXT = 'Учу сербский в UCIMO — уроки, повторение слов и озвучка. Заходи по моей ссылке:'
const canShare = typeof navigator !== 'undefined' && typeof navigator.share === 'function'

async function load() {
  loading.value = true
  error.value = ''
  try {
    referral.value = await api.referral()
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    loading.value = false
  }
}

watch(
  open,
  (isOpen) => {
    copied.value = false
    if (isOpen) load()
  },
  { immediate: true },
)

const telegramShareUrl = computed(() =>
  referral.value
    ? `https://t.me/share/url?url=${encodeURIComponent(referral.value.url)}&text=${encodeURIComponent(SHARE_TEXT)}`
    : '',
)

function flashCopied() {
  copied.value = true
  clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => (copied.value = false), 2500)
}

// The clipboard API needs a secure context and a permission; when it says no,
// selecting the field and asking the browser to copy still works almost
// everywhere, and if even that fails the link is left selected for a manual copy.
async function copy() {
  if (!referral.value) return
  try {
    await navigator.clipboard.writeText(referral.value.url)
    flashCopied()
    return
  } catch {
    /* fall through to the selection fallback */
  }
  const el = input.value
  if (!el) return
  el.focus()
  el.select()
  el.setSelectionRange(0, el.value.length)
  try {
    if (document.execCommand('copy')) flashCopied()
  } catch {
    /* the link stays selected */
  }
}

async function share() {
  if (!referral.value) return
  try {
    await navigator.share({ text: SHARE_TEXT, url: referral.value.url })
  } catch {
    /* dismissed by the learner, or not allowed — nothing to report */
  }
}

// Inside Telegram the share link opens Telegram's own chat picker; elsewhere
// the anchor opens it in a new tab.
function shareInTelegram(e: Event) {
  if (telegramShareUrl.value && openTelegramLink(telegramShareUrl.value)) e.preventDefault()
}
</script>

<template>
  <ModalShell :open="open" size="sm" panel-class="p-5 text-center" @close="closeModal">
    <button
      class="absolute right-3 top-3 z-10 flex h-9 w-9 items-center justify-center rounded-full bg-black/35 text-white transition hover:bg-black/50"
      title="Закрыть"
      aria-label="Закрыть"
      @click="closeModal"
    >
      <X :size="18" :stroke-width="2.5" />
    </button>

    <!-- The picture runs to the panel's edges, so it pulls out of the padding. -->
    <img
      :src="inviteScene"
      alt=""
      class="-mx-5 -mt-5 mb-3 block max-w-none rounded-t-[20px] object-cover"
      style="width: calc(100% + 2.5rem); aspect-ratio: 3 / 2"
    />

    <div class="space-y-3" data-test="invite-modal">
      <p class="text-lg font-extrabold">Позови друга</p>
      <p class="text-sm text-[var(--muted)]">
        Друг зарегистрируется по твоей ссылке — тебе
        <span class="inline-flex items-center gap-1 align-middle font-extrabold text-[var(--fg)]">
          {{ props.reward }}<SeedIcon :size="15" />
        </span>
      </p>

      <p v-if="error" class="text-sm text-[var(--bad)]" data-test="invite-error">{{ error }}</p>
      <div v-else-if="loading || !referral" class="skel mx-auto h-11 w-full" />

      <template v-else>
        <input
          ref="input"
          :value="referral.url"
          readonly
          class="field w-full text-center text-sm"
          aria-label="Ссылка-приглашение"
          data-test="invite-link"
          @focus="($event.target as HTMLInputElement).select()"
        />
        <button class="btn btn-primary w-full" data-test="invite-copy" @click="copy">
          <Check v-if="copied" :size="16" :stroke-width="2.5" />
          <Copy v-else :size="16" :stroke-width="2.25" />
          {{ copied ? 'Ссылка скопирована' : 'Скопировать ссылку' }}
        </button>
        <div class="flex flex-col gap-2 sm:flex-row">
          <a
            :href="telegramShareUrl"
            target="_blank"
            rel="noopener"
            class="btn btn-ghost flex-1"
            data-test="invite-telegram"
            @click="shareInTelegram"
          >
            <Send :size="16" :stroke-width="2.25" />Telegram
          </a>
          <button v-if="canShare" type="button" class="btn btn-ghost flex-1" data-test="invite-share" @click="share">
            <Share2 :size="16" :stroke-width="2.25" />Поделиться
          </button>
        </div>
        <p class="text-xs text-[var(--muted)]" data-test="invite-friends">
          Друг должен подтвердить почту или войти через Telegram — тогда задание засчитается.
          <template v-if="referral.friends > 0">
            Уже присоединились: {{ referral.friends }}.
          </template>
        </p>
      </template>
    </div>
  </ModalShell>
</template>
