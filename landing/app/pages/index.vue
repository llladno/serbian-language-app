<template>
  <main class="hero page-content">
    <h1 class="hero__title">Учи сербский с нуля</h1>
    <p class="hero__subtitle">
      Личный курс сербского: уроки с автопроверкой, тренажёр слов на
      интервальном повторении и дашборд прогресса.
    </p>
    <a class="btn btn--primary" href="/login">Войти</a>
  </main>
</template>

<script setup lang="ts">
const title = 'ucimo — учи сербский с нуля'
const description =
  'Личное приложение для изучения сербского языка: уроки с автопроверкой, тренажёр слов на интервальном повторении (SRS), дашборд прогресса.'
const url = 'https://ucimo.ru/'

useSeoMeta({
  title,
  description,
  ogType: 'website',
  ogTitle: title,
  ogDescription: description,
  ogUrl: url,
  ogSiteName: 'ucimo',
  ogLocale: 'ru_RU',
  twitterCard: 'summary',
  twitterTitle: title,
  twitterDescription: description,
})

useHead({
  link: [{ rel: 'canonical', href: url }],
  script: [
    {
      key: 'ld-json-index',
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebSite',
        name: 'ucimo',
        url,
        description,
        inLanguage: 'ru',
      }),
    },
  ],
})
</script>

<style scoped>
.hero {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  max-width: 640px;
  margin: 0 auto;
  padding: 96px 20px;
}

.hero__title {
  font-size: clamp(32px, 5vw, 48px);
  font-weight: 800;
  letter-spacing: -0.02em;
  margin: 0;
  color: var(--color-text);
}

.hero__subtitle {
  font-size: 18px;
  color: var(--color-muted);
  margin: 0;
  max-width: 480px;
}
</style>
