// Small, self-dismissing notices in the corner. Module-level state, like the
// other shared modals, so anything can raise one and the single <ToastStack>
// in AppNav renders it.
import { ref } from 'vue'

const VISIBLE_MS = 6000
const MAX_VISIBLE = 3

export interface Toast {
  id: number
  title: string
  text: string
  /** Shown as "+N" with a feather next to it. */
  reward?: number
  /** Where tapping the toast goes. */
  to?: string
}

const items = ref<Toast[]>([])
let nextId = 1
const timers = new Map<number, ReturnType<typeof setTimeout>>()

function dismiss(id: number) {
  items.value = items.value.filter((t) => t.id !== id)
  const timer = timers.get(id)
  if (timer) clearTimeout(timer)
  timers.delete(id)
}

function push(toast: Omit<Toast, 'id'>): number {
  const id = nextId++
  items.value = [...items.value, { ...toast, id }].slice(-MAX_VISIBLE)
  timers.set(
    id,
    setTimeout(() => dismiss(id), VISIBLE_MS),
  )
  return id
}

// Tests mount and unmount the stack repeatedly; leftovers from one would show
// up in the next.
function clearToasts() {
  for (const timer of timers.values()) clearTimeout(timer)
  timers.clear()
  items.value = []
}

export function useToasts() {
  return { items, push, dismiss, clearToasts }
}
