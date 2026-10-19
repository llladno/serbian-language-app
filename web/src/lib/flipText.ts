// A short fade-in for text that has just been converted to the other script. The
// words change at once (nothing is re-created, so an answer being typed or an
// exercise already answered stays as it is); this only softens the swap.
//
// Opacity only, on purpose: a transform or a filter on a lesson's wrapper would, even
// for a moment, make it the containing block of the "Проверить" bar that the exercises
// pin to the bottom of the screen (see LessonView), and the bar would jump.
export function flipText(els: Array<Element | null | undefined>): void {
  if (typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches) return
  for (const el of els) {
    el?.animate?.([{ opacity: 0.15 }, { opacity: 1 }], { duration: 320, easing: 'ease-out' })
  }
}
