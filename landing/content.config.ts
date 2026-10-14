import { defineCollection, defineContentConfig } from '@nuxt/content'
import { z } from 'zod'

export default defineContentConfig({
  collections: {
    blog: defineCollection({
      type: 'page',
      source: 'blog/*.md',
      schema: z.object({
        date: z.date(),
        keywords: z.array(z.string()).default([]),
        // Visible FAQ section is written in the post body; the same pairs go here
        // so the page can emit matching FAQPage JSON-LD (see app/pages/blog/[slug].vue).
        faq: z.array(z.object({ q: z.string(), a: z.string() })).default([]),
        og_image: z.string().optional(),
      }),
    }),
  },
})
