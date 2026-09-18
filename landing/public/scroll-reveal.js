(function () {
  if (!('IntersectionObserver' in window)) return
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return

  function init() {
    var els = document.querySelectorAll('.reveal')
    if (!els.length) return

    els.forEach(function (el) {
      el.classList.add('is-hidden')
    })

    var io = new IntersectionObserver(
      function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) {
            entry.target.classList.remove('is-hidden')
            io.unobserve(entry.target)
          }
        })
      },
      { threshold: 0.15, rootMargin: '0px 0px -40px 0px' },
    )

    els.forEach(function (el) {
      io.observe(el)
    })
  }

  // Runs after the whole page (incl. framework hydration in dev — this
  // is static HTML in production, no hydration there) has settled, so
  // this never mutates `class` on an element a framework is still
  // reconciling.
  if (document.readyState === 'complete') {
    init()
  } else {
    window.addEventListener('load', init)
  }
})()
