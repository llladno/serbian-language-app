<template>
  <footer class="footer">
    <div class="footer__inner">
      <span>ucimo</span>
      <nav class="footer__links">
        <NuxtLink to="/privacy">Политика конфиденциальности</NuxtLink>
        <NuxtLink to="/terms">Условия использования</NuxtLink>
      </nav>
    </div>
  </footer>
</template>

<style scoped>
.footer {
  padding: 24px 20px;
  color: var(--color-muted);
  font-size: 14px;
}

.footer__inner {
  max-width: 960px;
  margin: 0 auto;
  display: flex;
  flex-wrap: wrap;
  gap: 8px 24px;
  align-items: center;
  justify-content: space-between;
}

.footer__links {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
}

.footer__links a {
  color: var(--color-muted);
  text-decoration: underline;
}

.footer__links a:hover {
  color: var(--color-text);
}
</style>
