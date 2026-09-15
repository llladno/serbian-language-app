<template>
  <header class="header">
    <div class="header__inner">
      <NuxtLink to="/" class="wordmark">ucimo</NuxtLink>
      <a class="btn btn--ghost" href="/login">Войти</a>
    </div>
  </header>
</template>

<style scoped>
.header {
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}

.header__inner {
  max-width: 960px;
  margin: 0 auto;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.wordmark {
  font-weight: 800;
  font-size: 20px;
  letter-spacing: -0.02em;
  color: var(--color-primary);
  text-decoration: none;
}
</style>
