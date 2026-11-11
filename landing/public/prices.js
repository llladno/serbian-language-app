// The prices on the landing are written into the page when it is built, and the production build ships no framework script
// (nuxt.config.ts, `noScripts`), so this small script asks the app for the live prices and puts them into the page. The app
// changes its prices on its own at the moment of an announced change, so a visitor after it sees the new ones without a rebuild.
// Elements: data-price="levelRegular|levelFirst|year" (the text is replaced), data-price-first (the "за первый" line, hidden when
// the first purchase costs the same). The FAQ text and the markup for search engines are not touched: they follow the next build.
(function () {
  fetch('/api/prices', { credentials: 'omit' })
    .then(function (r) {
      return r.ok ? r.json() : null
    })
    .then(function (p) {
      if (!p || !p.level || !p.year || !(p.level.regular > 0) || !(p.level.first > 0) || !(p.year > 0)) return
      var fmt = function (n) {
        return n.toLocaleString('ru-RU').replace(/\s/g, ' ') + ' ₽'
      }
      var map = { levelRegular: fmt(p.level.regular), levelFirst: fmt(p.level.first), year: fmt(p.year) }
      document.querySelectorAll('[data-price]').forEach(function (el) {
        var v = map[el.getAttribute('data-price')]
        if (v) el.textContent = v
      })
      document.querySelectorAll('[data-price-first]').forEach(function (el) {
        el.hidden = !(p.level.first < p.level.regular)
        el.textContent = fmt(p.level.first) + ' за первый'
      })
    })
    .catch(function () {})
})()
