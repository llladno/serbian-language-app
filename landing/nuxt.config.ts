// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  css: ['~/assets/css/main.css'],

  // Purely static content, zero interactivity (just <a href> links) — skip
  // shipping Nuxt's own JS/hydration bundle in the generated output. Also
  // sidesteps a real conflict: Nuxt's inline hydration bootstrap script
  // would otherwise be blocked by the app's CSP, which has no 'unsafe-inline'
  // for script-src (see server/internal/api/middleware.go).
  features: {
    noScripts: 'production',
  },

  // Explicit route list rather than the link-crawler, so it doesn't wander
  // into the /login link and try to prerender the Go app's own SPA route.
  nitro: {
    prerender: {
      crawlLinks: false,
      routes: ['/', '/privacy', '/terms'],
    },
  },

  app: {
    head: {
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        {
          name: 'google-site-verification',
          content: 'wN2GAS2b71nQI4eM6AFwpC_SoGT2DlGCNE36OajjjOE',
        },
        { name: 'p:domain_verify', content: 'f21147467cc1a41332fabce1e6d671a2' },
        { name: 'theme-color', content: '#f7f9f7', media: '(prefers-color-scheme: light)' },
        { name: 'theme-color', content: '#0f1b1a', media: '(prefers-color-scheme: dark)' },
      ],
      link: [
        { rel: 'icon', type: 'image/png', href: '/favicon.png' },
        { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' },
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Manrope:wght@400;500;600;700;800&family=Lora:ital,wght@0,500;0,600;1,500&display=swap',
        },
      ],
      // Yandex.Metrika — served as a same-origin file (public/metrika-init.js)
      // rather than inlined, so the app's CSP doesn't need 'unsafe-inline' for
      // script-src; only https://mc.yandex.ru needs allowlisting (see
      // server/internal/api/middleware.go). Hand-rolled rather than a
      // third-party Nuxt module (those are mostly unmaintained / Nuxt 2-era).
      script: [{ key: 'yandex-metrika', src: '/metrika-init.js' }],
      noscript: [
        {
          key: 'yandex-metrika-noscript',
          innerHTML:
            '<div><img src="https://mc.yandex.ru/watch/112668663" style="position:absolute; left:-9999px;" alt="" /></div>',
        },
      ],
    },
  },
})
