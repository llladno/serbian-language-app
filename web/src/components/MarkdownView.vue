<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'
import { loadPhraseIndex, phraseClip } from '../lib/phraseAudio'
import { playClip } from '../lib/playClip'

const props = defineProps<{ source: string }>()

const md = new MarkdownIt({ html: false, linkify: false, typographer: true })

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
  const clip = phraseClip(tokens[idx].content)
  if (!clip) return base
  return `${base}<button type="button" class="md-speak" data-clip="${md.utils.escapeHtml(clip)}" aria-label="озвучить" title="озвучить">${SPEAKER_SVG}</button>`
}

void loadPhraseIndex()
const html = computed(() => md.render(props.source ?? ''))

function onClick(e: MouseEvent) {
  const btn = (e.target as HTMLElement).closest<HTMLElement>('[data-clip]')
  if (btn?.dataset.clip) playClip(btn.dataset.clip)
}
</script>

<template>
  <div class="md" @click="onClick" v-html="html" />
</template>
