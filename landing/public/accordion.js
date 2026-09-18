(function () {
  if (typeof Element === 'undefined' || !Element.prototype.animate) return
  var reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches

  var items = document.querySelectorAll('.faq__item')
  if (!items.length) return

  items.forEach(function (details) {
    var summary = details.querySelector('summary')
    if (!summary || details.open) return

    // Captured once, while the item is still in its natural closed state —
    // more reliable than re-deriving it later from summary/content pieces.
    var closedHeight = details.offsetHeight + 'px'

    var animation = null
    var isClosing = false
    var isExpanding = false

    summary.addEventListener('click', function (e) {
      e.preventDefault()
      if (reduced) {
        details.open = !details.open
        return
      }
      details.style.overflow = 'hidden'
      if (isClosing || !details.open) {
        expandOpen()
      } else if (isExpanding || details.open) {
        shrinkClose()
      }
    })

    function shrinkClose() {
      isClosing = true
      var startHeight = details.offsetHeight + 'px'
      if (animation) animation.cancel()
      animation = details.animate(
        { height: [startHeight, closedHeight] },
        { duration: 220, easing: 'ease-out' },
      )
      animation.onfinish = function () {
        onFinish(false)
      }
      animation.oncancel = function () {
        isClosing = false
      }
    }

    function expandOpen() {
      details.style.height = closedHeight
      details.open = true
      window.requestAnimationFrame(function () {
        isExpanding = true
        var endHeight = details.scrollHeight + 'px'
        if (animation) animation.cancel()
        animation = details.animate(
          { height: [closedHeight, endHeight] },
          { duration: 260, easing: 'ease-out' },
        )
        animation.onfinish = function () {
          onFinish(true)
        }
        animation.oncancel = function () {
          isExpanding = false
        }
      })
    }

    function onFinish(isOpen) {
      details.open = isOpen
      animation = null
      isClosing = false
      isExpanding = false
      details.style.height = ''
      details.style.overflow = ''
    }
  })
})()
