<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

const props = withDefaults(
  defineProps<{
    modelValue: string
    options: string[]
    placeholder?: string
    disabled?: boolean
    state?: 'ok' | 'bad' | null
  }>(),
  { placeholder: '— выбери —', disabled: false, state: null },
)
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)

function toggle() {
  if (!props.disabled) open.value = !open.value
}
function pick(o: string) {
  emit('update:modelValue', o)
  open.value = false
}
function onDocClick(e: MouseEvent) {
  if (root.value && !root.value.contains(e.target as Node)) open.value = false
}
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') open.value = false
}
onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onKey)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onKey)
})
</script>

<template>
  <div ref="root" class="relative">
    <button
      type="button"
      :disabled="disabled"
      class="flex w-full items-center justify-between gap-2 rounded-lg border px-3 py-2 text-left text-sm transition"
      :class="[
        state === 'ok'
          ? 'border-[var(--good)] ring-1 ring-[var(--good)]'
          : state === 'bad'
            ? 'border-[var(--bad)] ring-1 ring-[var(--bad)]'
            : open
              ? 'border-[var(--accent)] ring-1 ring-[var(--accent)]'
              : 'border-[var(--border)] hover:border-[var(--accent)]',
        disabled ? 'opacity-60' : 'bg-[var(--card)]',
      ]"
      @click="toggle"
    >
      <span :class="modelValue ? 'serbian text-[var(--fg)]' : 'text-[var(--muted)]'">
        {{ modelValue || placeholder }}
      </span>
      <span class="text-xs text-[var(--muted)] transition" :class="{ 'rotate-180': open }">▾</span>
    </button>

    <div
      v-if="open"
      class="absolute z-30 mt-1 w-full overflow-hidden rounded-lg border border-[var(--border)] bg-[var(--card)] py-1 shadow-[var(--shadow)]"
    >
      <button
        v-for="o in options"
        :key="o"
        type="button"
        class="serbian block w-full px-3 py-1.5 text-left text-sm transition"
        :class="
          o === modelValue
            ? 'bg-[var(--accent-soft)] text-[var(--accent)]'
            : 'text-[var(--fg)] hover:bg-[var(--bg-soft)]'
        "
        @click="pick(o)"
      >
        {{ o }}
      </button>
    </div>
  </div>
</template>
