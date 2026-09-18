(function () {
  var STORAGE_KEY = 'ucimo-cookie-consent'

  function getChoice() {
    try {
      return localStorage.getItem(STORAGE_KEY)
    } catch (e) {
      return null
    }
  }

  function setChoice(value) {
    try {
      localStorage.setItem(STORAGE_KEY, value)
    } catch (e) {
      /* private mode / storage blocked — banner just reappears next visit */
    }
  }

  function loadMetrika() {
    if (document.querySelector('script[data-metrika]')) return
    var s = document.createElement('script')
    s.src = '/metrika-init.js'
    s.setAttribute('data-metrika', '')
    document.body.appendChild(s)
  }

  function init() {
    var banner = document.getElementById('cookie-banner')
    var acceptBtn = document.getElementById('cookie-accept')
    var declineBtn = document.getElementById('cookie-decline')
    if (!banner || !acceptBtn || !declineBtn) return

    function show() {
      banner.hidden = false
    }

    function hide() {
      banner.hidden = true
    }

    acceptBtn.addEventListener('click', function () {
      setChoice('accepted')
      hide()
      loadMetrika()
    })

    declineBtn.addEventListener('click', function () {
      setChoice('declined')
      hide()
    })

    // A footer link (id="cookie-settings") lets a visitor change their mind
    // later — required for consent to be genuinely revocable, not just
    // askable once.
    var settingsLink = document.getElementById('cookie-settings')
    if (settingsLink) {
      settingsLink.addEventListener('click', function (e) {
        e.preventDefault()
        show()
      })
    }

    var choice = getChoice()
    if (choice === 'accepted') {
      loadMetrika()
    } else if (choice !== 'declined') {
      show()
    }
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init)
  } else {
    init()
  }
})()
