<script setup lang="ts">
// A `.field`-styled dropdown trigger + app-themed option list, replacing the
// native <select> whose open popup can't be restyled (shows raw OS chrome
// regardless of the page's theme).
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { ChevronDown } from 'lucide-vue-next'

const props = defineProps<{ modelValue: string; options: { value: string; label: string }[] }>()
const emit = defineEmits<{ 'update:modelValue': [v: string] }>()

const open = ref(false)
const current = computed(() => props.options.find((o) => o.value === props.modelValue) ?? props.options[0])

function pick(v: string) {
  emit('update:modelValue', v)
  open.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') open.value = false
}
watch(open, (v) => {
  if (v) window.addEventListener('keydown', onKeydown)
  else window.removeEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="relative">
    <button
      type="button"
      class="field group flex w-full items-center justify-between gap-1.5 whitespace-nowrap transition hover:bg-[var(--accent-soft)] hover:text-[var(--accent)]"
      @click="open = !open"
    >
      <span>{{ current?.label }}</span>
      <ChevronDown :size="14" :stroke-width="2.5" class="text-[var(--muted)] transition group-hover:text-[var(--accent)]" />
    </button>

    <template v-if="open">
      <div class="fixed inset-0 z-10" @click="open = false" />
      <div
        class="card option-list absolute left-0 top-[calc(100%+0.375rem)] z-20 max-h-64 min-w-full overflow-auto p-1 pop"
      >
        <button
          v-for="o in options"
          :key="o.value"
          type="button"
          class="block w-full whitespace-nowrap rounded-lg px-3 py-2 text-left text-sm transition"
          :class="
            o.value === modelValue
              ? 'bg-[var(--accent)] text-white'
              : 'text-[var(--fg)] hover:bg-[var(--accent-soft)] hover:text-[var(--accent)]'
          "
          @click="pick(o.value)"
        >
          {{ o.label }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.option-list {
  scrollbar-width: thin;
  scrollbar-color: var(--ring-track) transparent;
}
.option-list::-webkit-scrollbar {
  width: 8px;
}
.option-list::-webkit-scrollbar-track {
  background: transparent;
}
.option-list::-webkit-scrollbar-thumb {
  background-color: var(--ring-track);
  border-radius: 8px;
}
.option-list::-webkit-scrollbar-thumb:hover {
  background-color: var(--muted);
}
</style>
