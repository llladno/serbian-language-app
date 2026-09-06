<script setup lang="ts">
// A row of Serbian-specific Latin letters. Tapping one inserts it at the
// caret of the text field that currently has focus (input or textarea),
// so you don't need a Serbian keyboard layout.
//
// pointerdown.prevent keeps the field focused (and the mobile keyboard up).

const CHARS = ['č', 'ć', 'đ', 'š', 'ž']

function insert(ch: string) {
  const el = document.activeElement as HTMLInputElement | HTMLTextAreaElement | null
  if (!el || (el.tagName !== 'INPUT' && el.tagName !== 'TEXTAREA') || el.disabled) return
  const start = el.selectionStart ?? el.value.length
  const end = el.selectionEnd ?? el.value.length
  el.value = el.value.slice(0, start) + ch + el.value.slice(end)
  const caret = start + ch.length
  el.setSelectionRange(caret, caret)
  el.dispatchEvent(new Event('input', { bubbles: true }))
}
</script>

<template>
  <div class="flex flex-wrap gap-1" aria-label="сербские буквы">
    <button
      v-for="ch in CHARS"
      :key="ch"
      type="button"
      tabindex="-1"
      class="serbian min-w-[2rem] rounded-md border border-[var(--border)] px-2 py-1 text-sm leading-none text-[var(--muted)] transition hover:border-[var(--accent)] hover:text-[var(--accent)] active:scale-90"
      @pointerdown.prevent="insert(ch)"
    >
      {{ ch }}
    </button>
  </div>
</template>
