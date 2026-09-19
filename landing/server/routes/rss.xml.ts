const SITE_URL = 'https://ucimo.ru'

function escapeXml(value: string) {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

export default defineEventHandler(async (event) => {
  const posts = await queryCollection(event, 'blog')
    .order('date', 'DESC')
    .select('title', 'description', 'path', 'date')
    .all()

  const items = posts
    .map((post) => {
      const link = `${SITE_URL}${post.path}`
      return `  <item>
    <title>${escapeXml(post.title)}</title>
    <link>${link}</link>
    <guid isPermaLink="true">${link}</guid>
    <description>${escapeXml(post.description)}</description>
    <pubDate>${new Date(post.date).toUTCString()}</pubDate>
  </item>`
    })
    .join('\n')

  setHeader(event, 'Content-Type', 'application/rss+xml; charset=utf-8')
  return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <title>ucimo — блог о сербском языке</title>
  <link>${SITE_URL}/blog</link>
  <description>Статьи об изучении сербского языка для русскоговорящих</description>
  <language>ru</language>
${items}
</channel>
</rss>
`
})
