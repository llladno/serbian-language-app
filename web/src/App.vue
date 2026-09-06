<script setup lang="ts">
import AppNav from './components/AppNav.vue'
</script>

<template>
  <AppNav />
  <main class="mx-auto max-w-3xl px-4 pb-24 pt-5">
    <RouterView v-slot="{ Component }">
      <Transition name="fade" mode="out-in">
        <component :is="Component" />
      </Transition>
    </RouterView>
  </main>
</template>
