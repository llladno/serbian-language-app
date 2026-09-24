(function () {
  var METRIKA_ID = 112668663

  // The landing ships zero Nuxt/Vue client JS in production
  // (nuxt.config.ts `features.noScripts`), so goal tracking on CTAs can't
  // use Vue's @click — it never attaches. Plain listeners on [data-goal]
  // elements instead, same pattern as cookie-consent.js.
  function bind() {
    var els = document.querySelectorAll('[data-goal]')
    for (var i = 0; i < els.length; i++) {
      ;(function (el) {
        el.addEventListener('click', function () {
          var goal = el.getAttribute('data-goal')
          if (window.ym) window.ym(METRIKA_ID, 'reachGoal', goal)
        })
      })(els[i])
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', bind)
  } else {
    bind()
  }
})()
