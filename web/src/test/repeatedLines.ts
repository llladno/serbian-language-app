import type { VueWrapper } from '@vue/test-utils'

// A line of text must not appear twice on a learning screen: a listening choice read
// as text once showed its sentence and then the same sentence again as "what was
// played" (a learner's report, 2026-10-28). A line is the visible text of one block
// (a paragraph, a label, a heading...), with the words inside it joined, so a
// dialogue line split into clickable words still counts as one line. Buttons are
// left out: an answer option is legitimately repeated in «Правильно: …» under a
// wrong answer. Hidden blocks (v-show) are left out too.

const BLOCK = 'p, li, h1, h2, h3, h4, label, div, section, header, form'

function hidden(el: Element | null): boolean {
  for (let e = el; e; e = e.parentElement) {
    if ((e as HTMLElement).style?.display === 'none') return true
  }
  return false
}

/** Lines that appear more than once on screen. */
export function repeatedLines(w: VueWrapper): string[] {
  const root = w.element as Element
  const lines = new Map<Element, string>()
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT)
  for (let n = walker.nextNode(); n; n = walker.nextNode()) {
    const parent = n.parentElement
    if (!parent || parent.closest('button') || hidden(parent)) continue
    const block = parent.closest(BLOCK) ?? root
    lines.set(block, (lines.get(block) ?? '') + (n.textContent ?? ''))
  }
  const seen = new Map<string, number>()
  for (const text of lines.values()) {
    const line = text.replace(/\s+/g, ' ').trim()
    if (line) seen.set(line, (seen.get(line) ?? 0) + 1)
  }
  return [...seen].filter(([, n]) => n > 1).map(([t]) => t)
}
