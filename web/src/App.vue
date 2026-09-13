<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { useSessionStore } from './stores/session'
import AppNav from './components/AppNav.vue'

const session = useSessionStore()
onMounted(() => {
  session.fetchSession()
})
</script>

<template>
  <div v-if="session.loading" class="flex min-h-screen items-center justify-center text-[var(--muted)]">
    Загрузка…
  </div>
  <template v-else-if="session.user">
    <AppNav :name="session.user.name" />
    <main class="mx-auto max-w-3xl px-4 pt-5 pb-[calc(5rem+env(safe-area-inset-bottom))] sm:pb-16">
      <RouterView v-slot="{ Component, route }">
        <div
          :key="route.name === 'lesson' ? route.fullPath : (route.name as string)"
          class="view-in"
        >
          <component :is="Component" />
        </div>
      </RouterView>
    </main>
  </template>
  <template v-else>
    <RouterView />
  </template>
</template>
