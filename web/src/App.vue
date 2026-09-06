<script setup lang="ts">
import { getAccount } from './account'
import AppNav from './components/AppNav.vue'
import LoginGate from './components/LoginGate.vue'

const account = getAccount()
</script>

<template>
  <LoginGate v-if="!account" />
  <template v-else>
    <AppNav :account="account" />
    <main class="mx-auto max-w-3xl px-4 pb-24 pt-5">
      <RouterView v-slot="{ Component }">
        <Transition name="fade" mode="out-in">
          <component :is="Component" />
        </Transition>
      </RouterView>
    </main>
  </template>
</template>
