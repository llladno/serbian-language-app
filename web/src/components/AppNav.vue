<script setup lang="ts">
const links = [
  { to: '/', label: 'Дашборд' },
  { to: '/course', label: 'Курс' },
  { to: '/review', label: 'Повторение' },
  { to: '/vocab', label: 'Словарь' },
  { to: '/false-friends', label: 'Ложные друзья' },
]
</script>

<template>
  <nav class="sticky top-0 z-10 flex flex-wrap gap-x-4 gap-y-1 border-b border-stone-300 bg-stone-100/90 px-4 py-2 backdrop-blur dark:border-stone-700 dark:bg-stone-900/90">
    <RouterLink
      v-for="l in links"
      :key="l.to"
      :to="l.to"
      class="text-sm text-stone-600 hover:text-stone-900 dark:text-stone-400 dark:hover:text-stone-100"
      active-class="!text-stone-900 font-semibold dark:!text-stone-100"
    >
      {{ l.label }}
    </RouterLink>
  </nav>
</template>
