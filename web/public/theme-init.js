// apply saved theme before first paint to avoid a flash
try {
  var t = localStorage.getItem('srpski.theme')
  if (t === 'light' || t === 'dark') document.documentElement.setAttribute('data-theme', t)
} catch (e) {}
