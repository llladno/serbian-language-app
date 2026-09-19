<template>
  <main class="prose page-content blog-index">
    <h1>Блог ucimo</h1>
    <p class="prose__updated">
      Статьи о сербском языке для тех, кто учит его самостоятельно — с
      нуля, для переезда или просто для себя.
    </p>

    <ul class="blog-list">
      <li v-for="post in posts" :key="post.path" class="blog-card">
        <NuxtLink :to="post.path">
          <h2>{{ post.title }}</h2>
          <p>{{ post.description }}</p>
          <time :datetime="isoDate(post.date)">{{ formatDate(post.date) }}</time>
        </NuxtLink>
      </li>
    </ul>
  </main>
</template>

<script setup lang="ts">
const { data: posts } = await useAsyncData('blog-list', () =>
  queryCollection('blog')
    .order('date', 'DESC')
    .select('title', 'description', 'path', 'date')
    .all(),
)

function isoDate(value: string | Date) {
  return new Date(value).toISOString().slice(0, 10)
}

function formatDate(value: string | Date) {
  return new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(new Date(value))
}

const title = 'Блог — ucimo'
const description =
  'Статьи об изучении сербского языка: с чего начать, ложные друзья переводчика, кириллица и латиница, язык для переезда в Сербию.'
const url = 'https://ucimo.ru/blog'

useSeoMeta({
  title,
  description,
  ogType: 'website',
  ogTitle: title,
  ogDescription: description,
  ogUrl: url,
  ogSiteName: 'ucimo',
  ogLocale: 'ru_RU',
  ogImage: 'https://ucimo.ru/og-image.png',
  twitterCard: 'summary_large_image',
  twitterTitle: title,
  twitterDescription: description,
})

useHead({
  link: [{ rel: 'canonical', href: url }],
})
</script>

<style scoped>
.blog-index {
  max-width: 760px;
}

.blog-list {
  list-style: none;
  margin: 32px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.blog-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  transition: border-color 0.15s ease;
}

.blog-card:hover {
  border-color: var(--color-primary);
}

.blog-card a {
  display: block;
  padding: 20px 24px;
  text-decoration: none;
  color: inherit;
}

.blog-card h2 {
  margin: 0 0 8px;
  font-size: 19px;
  font-weight: 700;
  color: var(--color-primary-dark);
}

.blog-card p {
  margin: 0 0 10px;
  color: var(--color-muted);
  font-size: 15px;
  line-height: 1.6;
}

.blog-card time {
  color: var(--color-muted);
  font-size: 13px;
}
</style>
