<template>
  <div class="page">
    <SiteHeader />
    <NuxtPage />
    <SiteFooter />

    <div id="cookie-banner" class="cookie-banner" hidden>
      <div class="cookie-banner__inner">
        <p class="cookie-banner__text">
          Мы используем cookie для аналитики (Яндекс.Метрика), чтобы
          понимать, как вы пользуетесь сайтом. Подробнее — в
          <NuxtLink to="/privacy">Политике конфиденциальности</NuxtLink>.
        </p>
        <div class="cookie-banner__actions">
          <button id="cookie-decline" type="button" class="btn btn--ghost">
            Отклонить
          </button>
          <button id="cookie-accept" type="button" class="btn btn--primary">
            Принять
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
useHead({
  script: [
    {
      key: 'cookie-consent',
      src: '/cookie-consent.js',
    },
    {
      key: 'metrika-goals',
      src: '/metrika-goals.js',
    },
  ],
})
</script>

<style scoped>
.page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.cookie-banner {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 100;
  background: var(--color-surface);
  border-top: 1px solid var(--color-border);
  box-shadow: 0 -4px 24px rgba(23, 35, 38, 0.1);
}

.cookie-banner[hidden] {
  display: none;
}

.cookie-banner__inner {
  max-width: 1080px;
  margin: 0 auto;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
}

.cookie-banner__text {
  flex: 1;
  min-width: 240px;
  margin: 0;
  font-size: 14px;
  color: var(--color-text);
}

.cookie-banner__text a {
  color: var(--color-primary);
}

.cookie-banner__actions {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
}

.cookie-banner__actions .btn {
  margin-top: 0;
  padding: 10px 20px;
}

@media (max-width: 560px) {
  .cookie-banner__inner {
    flex-direction: column;
    align-items: stretch;
  }

  .cookie-banner__actions {
    justify-content: stretch;
  }

  .cookie-banner__actions .btn {
    flex: 1;
    text-align: center;
  }
}
</style>
