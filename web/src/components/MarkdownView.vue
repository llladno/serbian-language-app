<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'

const props = defineProps<{ source: string }>()

const md = new MarkdownIt({ html: false, linkify: false, typographer: true })

// Wrap tables in a horizontally scrollable container so wide grammar tables
// never blow out the layout on narrow screens.
md.renderer.rules.table_open = () => '<div class="md-tablewrap"><table>'
md.renderer.rules.table_close = () => '</table></div>'

const html = computed(() => md.render(props.source ?? ''))
</script>

<template>
  <div class="md" v-html="html" />
</template>
