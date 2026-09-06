<script setup lang="ts">
import AppNav from './components/AppNav.vue'
</script>

<template>
  <AppNav />
  <main class="mx-auto max-w-3xl p-4">
    <RouterView />
  </main>
</template>
