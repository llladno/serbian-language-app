<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'
import { loadPhraseIndex, phraseClip } from '../lib/phraseAudio'
import { playClip } from '../lib/playClip'
import { toCyrillic } from '../lib/script'
import { useScript } from '../lib/useScript'

const props = defineProps<{ source: string }>()

const md = new MarkdownIt({ html: false, linkify: false, typographer: true })
const { script } = useScript()

// Cyrillic readers: Serbian in the text, in `code spans` and in code blocks is shown
// converted (links and URLs are left alone). The course sets its parallel
// Serbian / Russian dialogues in code blocks, so those are Serbian text too. A code
// span keeps its Latin original in token.meta, because the audio clip is found by
// that, not by what is on screen.
md.core.ruler.push('serbian_script', (state) => {
  if (state.env?.script !== 'cyrillic') return
  for (const block of state.tokens) {
    if (block.type === 'fence' || block.type === 'code_block') block.content = toCyrillic(block.content)
    if (block.type !== 'inline' || !block.children) continue
    for (const t of block.children) {
      if (t.type === 'text') {
        t.content = toCyrillic(t.content)
      } else if (t.type === 'code_inline') {
        t.meta = { latin: t.content }
        t.content = toCyrillic(t.content)
      }
    }
  }
})

// Wrap tables in a horizontally scrollable container so wide grammar tables
// never blow out the layout on narrow screens.
md.renderer.rules.table_open = () => '<div class="md-tablewrap"><table>'
md.renderer.rules.table_close = () => '</table></div>'

// A Serbian example in `backticks` gets a speaker button right after it when
// scripts/tts.py has voiced it (lib/phraseAudio). Reads the reactive clip
// index, so the html re-renders once the index arrives.
const SPEAKER_SVG =
  '<svg viewBox="0 0 24 24" width="13" height="13" fill="none" stroke="currentColor" stroke-width="2.25" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M11 5 6 9H2v6h4l5 4V5z"/><path d="M15.54 8.46a5 5 0 0 1 0 7.07"/><path d="M19.07 4.93a10 10 0 0 1 0 14.14"/></svg>'
const defaultCode = md.renderer.rules.code_inline!
md.renderer.rules.code_inline = (tokens, idx, options, env, self) => {
  const base = defaultCode(tokens, idx, options, env, self)
  const clip = phraseClip((tokens[idx].meta as { latin?: string } | null)?.latin ?? tokens[idx].content)
  if (!clip) return base
  return `${base}<button type="button" class="md-speak" data-clip="${md.utils.escapeHtml(clip)}" aria-label="озвучить" title="озвучить">${SPEAKER_SVG}</button>`
}

void loadPhraseIndex()
const html = computed(() => md.render(props.source ?? '', { script: script.value }))

function onClick(e: MouseEvent) {
  const btn = (e.target as HTMLElement).closest<HTMLElement>('[data-clip]')
  if (btn?.dataset.clip) playClip(btn.dataset.clip)
}
</script>

<template>
  <div class="md" @click="onClick" v-html="html" />
</template>
