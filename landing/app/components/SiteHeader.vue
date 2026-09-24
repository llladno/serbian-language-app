<template>
  <header class="header">
    <div class="header__inner">
      <NuxtLink to="/" class="wordmark">ucimo</NuxtLink>
      <nav class="header__nav" aria-label="Разделы страницы">
        <a href="/#how">Как это работает</a>
        <a href="/#roadmap">Программа курса</a>
        <a href="/#faq">Вопросы</a>
        <NuxtLink to="/blog">Блог</NuxtLink>
      </nav>
      <a class="btn btn--ghost" href="/login" data-goal="login_click">Войти</a>
    </div>
  </header>
</template>

<style scoped>
.header {
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface);
}

.header__inner {
  max-width: 1080px;
  margin: 0 auto;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.header__nav {
  display: flex;
  gap: 24px;
  flex: 1;
  justify-content: center;
}

.header__nav a {
  color: var(--color-text);
  text-decoration: none;
  font-size: 15px;
  font-weight: 500;
}

.header__nav a:hover {
  color: var(--color-primary);
}

@media (max-width: 760px) {
  .header__nav {
    display: none;
  }
}

.wordmark {
  font-weight: 800;
  font-size: 20px;
  letter-spacing: -0.02em;
  color: var(--color-primary);
  text-decoration: none;
}
</style>
