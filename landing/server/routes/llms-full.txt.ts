import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const SITE_URL = 'https://ucimo.ru'

// Plain-text dump of the whole blog for LLM crawlers (GEO): one request returns every
// article's title, URL, date and full text. Read straight from content/blog/ at build time
// (the page is prerendered), so a new post shows up here without editing anything.
export default defineEventHandler((event) => {
  const dir = join(process.cwd(), 'content', 'blog')
  const posts = readdirSync(dir)
    .filter((file) => file.endsWith('.md'))
    .map((file) => {
      const raw = readFileSync(join(dir, file), 'utf-8')
      const match = raw.match(/^---\n([\s\S]*?)\n---\n?([\s\S]*)$/)
      const front = match?.[1] ?? ''
      const body = (match?.[2] ?? raw).trim()
      const field = (name: string) =>
        front.match(new RegExp(`^${name}:\\s*"?(.*?)"?$`, 'm'))?.[1] ?? ''
      return {
        slug: file.replace(/\.md$/, ''),
        title: field('title'),
        description: field('description'),
        date: field('date'),
        body,
      }
    })
    .sort((a, b) => b.date.localeCompare(a.date))

  const sections = posts
    .map(
      (post) => `# ${post.title}

URL: ${SITE_URL}/blog/${post.slug}
Дата: ${post.date}
Кратко: ${post.description}

${post.body}`,
    )
    .join('\n\n---\n\n')

  setHeader(event, 'Content-Type', 'text/plain; charset=utf-8')
  return `# ucimo — блог о сербском языке (полный текст)

> Статьи для русскоговорящих, которые учат сербский язык: алфавит, фразы, грамматика, переезд в Сербию. Сайт: ${SITE_URL}/. Краткая версия со ссылками: ${SITE_URL}/llms.txt

---

${sections}
`
})
