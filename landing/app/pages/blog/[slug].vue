<template>
  <main v-if="page" class="prose page-content article">
    <nav class="breadcrumbs" aria-label="Хлебные крошки">
      <NuxtLink to="/">Главная</NuxtLink>
      <span aria-hidden="true">→</span>
      <NuxtLink to="/blog">Блог</NuxtLink>
      <span aria-hidden="true">→</span>
      <span class="breadcrumbs__current">{{ page.title }}</span>
    </nav>

    <h1>{{ page.title }}</h1>
    <p class="prose__updated">{{ formatDate(page.date) }}</p>

    <div class="article-body">
      <ContentRenderer :value="page" />
    </div>

    <div class="article-cta">
      <p>Начните учить сербский бесплатно — 42 урока с нуля, без рекламы и подписки.</p>
      <a class="btn btn--primary" href="/register" @click="reachGoal('register_blog')"
        >Изучить сербский</a
      >
    </div>
  </main>
</template>

<script setup lang="ts">
const route = useRoute()

const { data: page } = await useAsyncData(route.path, () =>
  queryCollection('blog').path(route.path).first(),
)

if (!page.value) {
  throw createError({ statusCode: 404, statusMessage: 'Статья не найдена' })
}

function formatDate(value: string | Date) {
  return new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(new Date(value))
}

const url = `https://ucimo.ru${route.path}`
const ogImage = page.value.og_image
  ? `https://ucimo.ru${page.value.og_image}`
  : 'https://ucimo.ru/og-image.png'
const isoDate = new Date(page.value.date).toISOString()

useSeoMeta({
  title: page.value.title,
  description: page.value.description,
  ogType: 'article',
  ogTitle: page.value.title,
  ogDescription: page.value.description,
  ogUrl: url,
  ogSiteName: 'ucimo',
  ogLocale: 'ru_RU',
  ogImage,
  articlePublishedTime: isoDate,
  twitterCard: 'summary_large_image',
  twitterTitle: page.value.title,
  twitterDescription: page.value.description,
  twitterImage: ogImage,
})

useHead({
  link: [{ rel: 'canonical', href: url }],
  script: [
    {
      key: 'ld-json-article',
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@graph': [
          {
            '@type': 'BlogPosting',
            headline: page.value.title,
            description: page.value.description,
            datePublished: isoDate,
            dateModified: isoDate,
            inLanguage: 'ru',
            image: ogImage,
            mainEntityOfPage: url,
            author: { '@type': 'Organization', name: 'ucimo', url: 'https://ucimo.ru/' },
            publisher: {
              '@type': 'Organization',
              name: 'ucimo',
              logo: { '@type': 'ImageObject', url: 'https://ucimo.ru/favicon.png' },
            },
          },
          {
            '@type': 'BreadcrumbList',
            itemListElement: [
              { '@type': 'ListItem', position: 1, name: 'Главная', item: 'https://ucimo.ru/' },
              { '@type': 'ListItem', position: 2, name: 'Блог', item: 'https://ucimo.ru/blog' },
              { '@type': 'ListItem', position: 3, name: page.value.title, item: url },
            ],
          },
        ],
      }),
    },
  ],
})
</script>

<style scoped>
.article {
  max-width: 720px;
}

.breadcrumbs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 13px;
  color: var(--color-muted);
  margin-bottom: 24px;
}

.breadcrumbs a {
  color: var(--color-muted);
  text-decoration: underline;
}

.breadcrumbs a:hover {
  color: var(--color-primary);
}

.breadcrumbs__current {
  color: var(--color-text);
}

.article-cta {
  margin-top: 48px;
  padding: 28px 24px;
  background: var(--color-primary-soft);
  border-radius: 14px;
  text-align: center;
}

.article-cta p {
  margin: 0 0 4px;
  font-weight: 600;
  color: var(--color-primary-dark);
}
</style>
