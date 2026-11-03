/**
 * A fair shuffle (Fisher-Yates): every order is equally likely. Not `sort(() => Math.random() - 0.5)`: that comparator is
 * inconsistent, and the engine's sort then keeps the first item in front far too often (for three options the first
 * one stayed first 44% of the time instead of 33%), which is exactly what a quiz must not do with its right answer.
 */
export function shuffled<T>(items: readonly T[], random: () => number = Math.random): T[] {
  const out = [...items]
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(random() * (i + 1))
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}
