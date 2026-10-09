// The line under the percentage on the end-of-lesson screen. Four bands, five
// phrases each, one picked at random so a learner who finishes lesson after
// lesson does not read the same sentence twice in a row.
//
// Wording is deliberately free of gendered forms («молодец», «прошёл»): the app
// does not know who is reading. And no band tells a low score off — the lowest
// one points at the repetitions instead.

export type Band = 'perfect' | 'high' | 'good' | 'low'

export const CHEERS: Record<Band, string[]> = {
  perfect: [
    'Ни одной ошибки — безупречно',
    'Идеально: с первого раза и без единой ошибки',
    'Сто из ста — вот это уровень',
    'Чистая работа от начала до конца',
    'Урок на отлично — так и продолжай',
  ],
  high: [
    'Почти без ошибок — так держать',
    'Чисто! Материал явно лёг',
    'Отличная работа, сербский звучит всё увереннее',
    'Уверенный результат',
    'Вот так слова и запоминаются',
  ],
  good: [
    'Хороший результат, а ошибки как раз и учат',
    'Многое получилось, остальное закрепят повторения',
    'Крепкая работа, с каждым уроком будет легче',
    'Большая часть далась с первого раза',
    'Есть над чем поработать, и это нормально',
  ],
  low: [
    'Урок был непростым, и он позади — это главное',
    'Ошибки — часть пути, повторения помогут закрепить',
    'Не всё запоминается сразу, загляни в повторения',
    'Трудный материал — вернись к нему через пару дней',
    'Дальше поможет повторение, оно для этого и нужно',
  ],
}

export function bandFor(percent: number): Band {
  if (percent >= 100) return 'perfect'
  if (percent >= 80) return 'high'
  if (percent >= 60) return 'good'
  return 'low'
}

// `random` is injectable so a test can ask for a particular phrase.
export function cheerFor(percent: number, random: () => number = Math.random): string {
  const list = CHEERS[bandFor(percent)]
  return list[Math.min(list.length - 1, Math.floor(random() * list.length))]
}
