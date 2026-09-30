(function () {
  // The track itself is a plain CSS scroll-snap strip (see .reviews__track
  // in index.vue) — swipe/drag/trackpad scrolling already works with zero
  // JS. This only adds the optional arrow buttons and dot indicators on
  // top, same progressive-enhancement spirit as scroll-reveal.js/accordion.js.
  var sliders = document.querySelectorAll('.reviews__slider')
  if (!sliders.length) return

  sliders.forEach(function (root) {
    var track = root.querySelector('.reviews__track')
    var cards = track ? track.querySelectorAll('.review-card') : []
    var dotsWrap = root.querySelector('.reviews__dots')
    var prevBtn = root.querySelector('.reviews__nav--prev')
    var nextBtn = root.querySelector('.reviews__nav--next')
    if (!track || cards.length < 2) return

    var dots = []
    cards.forEach(function (card, i) {
      var dot = document.createElement('button')
      dot.type = 'button'
      dot.className = 'reviews__dot'
      dot.setAttribute('aria-label', 'Отзыв ' + (i + 1) + ' из ' + cards.length)
      dot.addEventListener('click', function () {
        track.scrollTo({ left: card.offsetLeft - track.offsetLeft, behavior: 'smooth' })
      })
      dotsWrap.appendChild(dot)
      dots.push(dot)
    })

    function closestIndex() {
      var target = track.scrollLeft + track.offsetLeft
      var closest = 0
      var closestDist = Infinity
      cards.forEach(function (card, i) {
        var d = Math.abs(card.offsetLeft - target)
        if (d < closestDist) {
          closestDist = d
          closest = i
        }
      })
      return closest
    }

    function sync() {
      var i = closestIndex()
      dots.forEach(function (dot, j) {
        dot.classList.toggle('is-active', j === i)
      })
      if (prevBtn) prevBtn.disabled = i === 0
      if (nextBtn) nextBtn.disabled = i === cards.length - 1
    }

    function goTo(i) {
      i = Math.max(0, Math.min(cards.length - 1, i))
      track.scrollTo({ left: cards[i].offsetLeft - track.offsetLeft, behavior: 'smooth' })
    }

    if (prevBtn) prevBtn.addEventListener('click', function () { goTo(closestIndex() - 1) })
    if (nextBtn) nextBtn.addEventListener('click', function () { goTo(closestIndex() + 1) })

    var raf = null
    track.addEventListener(
      'scroll',
      function () {
        if (raf) cancelAnimationFrame(raf)
        raf = requestAnimationFrame(sync)
      },
      { passive: true },
    )
    window.addEventListener('resize', sync)
    sync()
  })
})()
