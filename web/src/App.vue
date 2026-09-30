<script setup lang="ts">
import { onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { useSessionStore } from './stores/session'
import AppNav from './components/AppNav.vue'

const session = useSessionStore()
const route = useRoute()
onMounted(() => {
  session.fetchSession()
})
</script>

<template>
  <!--
    Only reachable on a genuine cold start (no cached guess from a previous
    session — see stores/session.ts readCachedUser()): we don't yet know
    whether to show the app chrome or the login page, so there's nothing
    honest to render but the bare background. A returning visit skips this
    entirely — session.loading is already false from the cached guess.
  -->
  <div v-if="session.loading" class="min-h-screen"></div>
  <template v-else-if="session.user">
    <AppNav :name="session.user.name" :hide-tab-bar="!!route.meta?.hideTabBar" />
    <main
      class="mx-auto px-4 pt-5 transition-[max-width]"
      :class="[
        route.meta?.wide ? 'max-w-5xl' : 'max-w-3xl',
        route.meta?.hideTabBar
          ? 'pb-[calc(6rem+env(safe-area-inset-bottom))]'
          : 'pb-[calc(5rem+env(safe-area-inset-bottom))] sm:pb-16',
      ]"
    >
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
