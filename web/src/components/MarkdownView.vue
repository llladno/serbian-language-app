<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'

const props = defineProps<{ source: string }>()

const md = new MarkdownIt({ html: false, linkify: false, typographer: true })

const html = computed(() => md.render(props.source ?? ''))
</script>

<template>
  <div class="md" v-html="html" />
</template>
