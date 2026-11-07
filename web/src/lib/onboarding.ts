// The swallow's tour of the app, shown over the real interface. One <OnboardingTour> lives in App.vue; the course map
// asks `checkDue` whether to start it by itself (the server says yes once, after the first finished lesson), and the
// quests screen starts it by hand. The server keeps the only memory: nothing here is persisted.
import { computed, ref } from 'vue'
import { api } from '../api'
import { useWalletStore } from '../stores/wallet'

export interface TourStep {
  id: string
  // data-tour keys of the nav buttons to light up; empty = nothing is lit
  targets: string[]
  // The screen she takes the learner to for this line, and the data-tour keys of the things on it she is talking about
  // (the next lesson on the course map, the first story). Both are optional.
  route?: string
  focus?: string[]
  // The last line: a full-screen celebration with the seeds instead of a lit button.
  finale?: boolean
  text: string
  // a gloss for a Serbian word in the text, small and grey under it
  tr?: string
  mascot: 'guide' | 'happy'
}

// How the tour was started. `auto` opens with the offer to look around; `quest` was asked for, so it goes straight to
// the first stop.
export type TourMode = 'auto' | 'quest'

const open = ref(false)
const index = ref(0)
const mode = ref<TourMode>('auto')
// What finishing pays now: 0 once the seeds are taken, or when there is no quest.
const reward = ref(0)
// The final line has been reached, so the tour is already recorded as seen to the end.
let finished = false

// The swallow's lines (approved wording). `amount` turns a number into «5 зёрнышек».
export function buildSteps(m: TourMode, pay: number, amount: (n: number) => string): TourStep[] {
  const steps: TourStep[] = []
  if (m === 'auto') {
    steps.push({
      id: 'hello',
      targets: [],
      text: pay
        ? `Ćao! Первый урок позади, молодец! Давай покажу, где тут что. Пройдёшь со мной до конца, подарю ${amount(pay)}.`
        : 'Ćao! Первый урок позади, молодец! Давай покажу, где тут что.',
      tr: 'Ćao — привет.',
      mascot: 'guide',
    })
  }
  steps.push(
    {
      id: 'course',
      targets: ['course'],
      route: '/course',
      focus: ['lesson'],
      text: 'Здесь находятся уроки. Это главный раздел для изучения языка.',
      mascot: 'guide',
    },
    {
      id: 'stories',
      targets: ['stories'],
      route: '/stories',
      focus: ['story'],
      text: 'Истории — короткие рассказы для практики языка. Слушаешь и привыкаешь к живой речи.',
      mascot: 'guide',
    },
    {
      id: 'words',
      // «Слова» and the dictionary next to it are lit as one button.
      targets: ['review', 'vocab'],
      route: '/review',
      focus: ['words-card'],
      text: 'В этих разделах ты будешь повторять и изучать новые слова.',
      mascot: 'guide',
    },
    {
      id: 'quests',
      targets: ['quests'],
      route: '/quests',
      focus: ['wallet'],
      text: 'В этом разделе можно получать зёрнышки за выполненные задания. На них открываются новые уровни.',
      mascot: 'guide',
    },
    {
      id: 'profile',
      targets: ['profile'],
      route: '/profile',
      focus: ['progress'],
      text: 'Здесь можно повторить уже пройденный материал и посмотреть свою статистику.',
      mascot: 'guide',
    },
    {
      id: 'seeds',
      targets: [],
      finale: true,
      text: pay ? `Вот и всё! Держи свои ${amount(pay)}. Srećno!` : 'Вот и всё! Srećno!',
      tr: 'Srećno — удачи.',
      mascot: 'happy',
    },
  )
  return steps
}

export function useOnboarding() {
  const wallet = useWalletStore()

  const steps = computed(() => buildSteps(mode.value, reward.value, (n) => wallet.amount(n)))
  const step = computed(() => steps.value[index.value])
  const isLast = computed(() => index.value === steps.value.length - 1)
  // The offer on the first line asks a question, so a tap on the empty screen must not answer it.
  const isOffer = computed(() => mode.value === 'auto' && index.value === 0)

  async function launch(m: TourMode, pay: number) {
    // The currency's name comes with the wallet; without it the reward would read as a bare number.
    if (!wallet.loaded) await wallet.refresh()
    mode.value = m
    reward.value = pay
    index.value = 0
    finished = false
    open.value = true
  }

  // Called by the course map: starts the tour when the server says it is due.
  async function checkDue() {
    if (open.value) return
    try {
      const s = await api.onboarding()
      if (s.due) await launch('auto', s.reward)
    } catch {
      // The tour is a courtesy: a failed check just means no tour this time.
    }
  }

  // Called by the quests screen; `pay` is what the quest still pays (0 when already taken).
  function startFromQuest(pay: number) {
    void launch('quest', pay)
  }

  // Arriving at the last line records the tour as seen and pays the reward, so the balance in the header ticks up
  // while the swallow says «держи».
  async function finish() {
    if (finished) return
    finished = true
    try {
      await api.answerOnboarding('done')
    } catch {
      return // the quest stays on «Задания», nothing is lost
    }
    wallet.refresh()
    wallet.loadQuests(true).catch(() => {})
  }

  function next() {
    if (!open.value || isLast.value) return
    index.value++
    if (isLast.value) void finish()
  }

  // «Отложить на потом»: stops the tour for good unless the learner starts it again from «Задания».
  // Resolves once the answer is recorded, so a screen that mounts next (the course map asks «due?» on mount) already
  // sees it.
  async function postpone() {
    if (!open.value) return
    open.value = false
    if (!finished) await api.answerOnboarding('postpone').catch(() => {})
  }

  // Closes the tour without recording an answer: leaving the screen (a lesson was opened) or the final «Спасибо!».
  function close() {
    open.value = false
  }

  return { open, index, mode, reward, step, steps, isLast, isOffer, checkDue, startFromQuest, next, postpone, close }
}
