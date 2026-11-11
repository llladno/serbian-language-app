// The prices of the paid tariffs, for the landing and the pages that quote them.
//
// The page is static, so what it says at first is what was true when it was built (BUILT: the prices of that day). The production
// build ships no framework script (`noScripts`), so the live prices reach the cards through public/prices.js (GET /api/prices); the
// request below runs only where the framework does run (the local dev server). The app changes its prices on its
// own at the moment of an announced change (migration 041 in the app), so the landing follows without a rebuild: a
// visitor after the change sees the new prices a moment after the page opens. When the app cannot be asked (the local
// dev server of the landing, a network error) the built prices stay.
//
// After a price change, set BUILT to the new prices and build the landing again, so that the first thing a visitor and a
// search engine see is already right.
export interface Prices {
  /** «Уровень» for a first purchase, and for every later one. */
  levelFirst: number
  levelRegular: number
  /** «Год». */
  year: number
}

const BUILT: Prices = { levelFirst: 199, levelRegular: 299, year: 1000 }

interface PricesReply {
  level?: { first?: number; regular?: number }
  year?: number
}

/** "1 999 ₽", with no-break spaces so a price never wraps in the middle. */
export function rub(n: number): string {
  return `${n.toLocaleString('ru-RU').replace(/\s/g, ' ')} ₽`
}

export function usePrices() {
  const prices = useState<Prices>('prices', () => ({ ...BUILT }))
  onMounted(async () => {
    try {
      const r = await $fetch<PricesReply>('/api/prices')
      const next: Prices = {
        levelFirst: r.level?.first ?? prices.value.levelFirst,
        levelRegular: r.level?.regular ?? prices.value.levelRegular,
        year: r.year ?? prices.value.year,
      }
      if (next.levelFirst > 0 && next.levelRegular > 0 && next.year > 0) prices.value = next
    } catch {
      /* the built prices stay */
    }
  })
  return prices
}
