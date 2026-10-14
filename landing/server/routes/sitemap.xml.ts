const SITE_URL = 'https://ucimo.ru'

// Home and /blog change whenever a post is published, so their lastmod follows the newest post.
const staticUrls = (lastmod: string) => [
  { loc: '/', changefreq: 'weekly', priority: '1.0', lastmod },
  { loc: '/blog', changefreq: 'weekly', priority: '0.8', lastmod },
  { loc: '/privacy', changefreq: 'yearly', priority: '0.3' },
  { loc: '/terms', changefreq: 'yearly', priority: '0.3' },
]

export default defineEventHandler(async (event) => {
  const posts = await queryCollection(event, 'blog')
    .order('date', 'DESC')
    .select('path', 'date')
    .all()

  const newest = posts.length
    ? new Date(posts[0].date).toISOString().slice(0, 10)
    : new Date().toISOString().slice(0, 10)

  const urls = [
    ...staticUrls(newest),
    ...posts.map((post) => ({
      loc: post.path,
      changefreq: 'monthly',
      priority: '0.6',
      lastmod: new Date(post.date).toISOString().slice(0, 10),
    })),
  ]

  const body = urls
    .map(
      (url) => `  <url>
    <loc>${SITE_URL}${url.loc}</loc>
${'lastmod' in url ? `    <lastmod>${url.lastmod}</lastmod>\n` : ''}    <changefreq>${url.changefreq}</changefreq>
    <priority>${url.priority}</priority>
  </url>`,
    )
    .join('\n')

  setHeader(event, 'Content-Type', 'application/xml; charset=utf-8')
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${body}\n</urlset>\n`
})
