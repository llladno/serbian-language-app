(function () {
  // First-touch UTM attribution for visitors who land on the public pages (the landing, the blog), which is
  // where UTM links point. The app (web/src/attribution.ts, captureAttribution) does the same for its own
  // pages and has to stay in step: same storage keys, same beacon. The landing ships no Nuxt/Vue client JS
  // (nuxt.config.ts `features.noScripts`), so without this file the visit was never counted and the UTM
  // parameters were dropped when the visitor pressed «Начать» (the links go to /register without them).
  var STORAGE_KEY = 'ucimo_attribution'
  var VISIT_LOGGED_KEY = 'ucimo_visit_logged'
  var FIELDS = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_content']

  var params
  try {
    params = new URLSearchParams(window.location.search)
  } catch (e) {
    return
  }
  var attr = {}
  var found = false
  for (var i = 0; i < FIELDS.length; i++) {
    var value = params.get(FIELDS[i])
    if (value) {
      attr[FIELDS[i]] = value
      found = true
    }
  }
  if (!found) return

  // One visit beacon per browser session, best effort.
  try {
    if (!sessionStorage.getItem(VISIT_LOGGED_KEY)) {
      sessionStorage.setItem(VISIT_LOGGED_KEY, '1')
      fetch('/api/track-visit', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(attr),
        keepalive: true,
      }).catch(function () {})
    }
  } catch (e) {
    /* sessionStorage unavailable (private mode etc.): skip the dedup */
  }

  // First touch only: the app reads this key when the visitor registers.
  try {
    if (!localStorage.getItem(STORAGE_KEY)) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(attr))
    }
  } catch (e) {
    /* localStorage unavailable: the attribution just will not reach registration */
  }
})()
