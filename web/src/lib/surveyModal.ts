// State of the one-time feedback survey. A single <SurveyModal> lives in AppNav;
// the end of a lesson asks `offerAfterLesson` whether to open it. The server
// answers "yes" at most once per learner, so nothing here remembers anything.
import { computed, ref } from 'vue'
import { api } from '../api'
import { authErrorMessage } from './authErrors'
import { useRewardModal } from './rewardModal'
import { useWalletStore } from '../stores/wallet'
import type { SurveyAnswers } from '../types'

export interface Option {
  code: string
  label: string
}

export const RATINGS = ['😞', '🙁', '😐', '🙂', '🤩']

export const LIKES: Option[] = [
  { code: 'lessons', label: 'Уроки и объяснения' },
  { code: 'audio', label: 'Озвучка' },
  { code: 'repeat', label: 'Повторение слов' },
  { code: 'seeds', label: 'Зёрнышки и задания' },
  { code: 'design', label: 'Как всё выглядит' },
  { code: 'other', label: 'Другое' },
]

export const PAINS: Option[] = [
  { code: 'hard', label: 'Бывает сложно' },
  { code: 'practice', label: 'Мало практики' },
  { code: 'unclear', label: 'Не всегда понятно задание' },
  { code: 'bugs', label: 'Что-то глючит' },
  { code: 'price', label: 'Цена' },
  { code: 'time', label: 'Не хватает времени' },
  { code: 'nothing', label: 'Ничего не мешает' },
  { code: 'other', label: 'Другое' },
]

export const PURPOSES: Option[] = [
  { code: 'move', label: 'Живу в Сербии или переезжаю' },
  { code: 'partner', label: 'Рядом сербы: партнёр, друзья, семья' },
  { code: 'trips', label: 'Езжу в Сербию отдыхать' },
  { code: 'interest', label: 'Просто интересно' },
  { code: 'other', label: 'Другое' },
]

export const WISHES: Option[] = [
  { code: 'speaking', label: 'Говорить: диалоги и живая практика' },
  { code: 'listening', label: 'Понимать на слух: больше озвучки' },
  { code: 'grammar', label: 'Грамматика попроще и с примерами' },
  { code: 'topics', label: 'Больше слов и фраз на нужные темы' },
  { code: 'reminders', label: 'Чтобы что-то помогало не забывать заниматься' },
  { code: 'people', label: 'Общение с другими учениками и сербами' },
  { code: 'nothing', label: 'Всего хватает' },
  { code: 'other', label: 'Другое' },
]

export const STEPS = 5
const SHOW_DELAY_MS = 1600

const open = ref(false)
const reward = ref(0)
const step = ref(0) // 0 = greeting, 1..5 = questions
const busy = ref(false)
const error = ref<string | null>(null)

const rating = ref(0)
const likes = ref<string[]>([])
const pains = ref<string[]>([])
const painText = ref('')
const purpose = ref('')
const wish = ref('')

function reset() {
  step.value = 0
  error.value = null
  rating.value = 0
  likes.value = []
  pains.value = []
  painText.value = ''
  purpose.value = ''
  wish.value = ''
}

// "Ничего не мешает" excludes every other complaint.
function togglePain(code: string) {
  if (code === 'nothing') {
    pains.value = pains.value.includes('nothing') ? [] : ['nothing']
    return
  }
  const rest = pains.value.filter((c) => c !== 'nothing')
  pains.value = rest.includes(code) ? rest.filter((c) => c !== code) : [...rest, code]
}

function toggleLike(code: string) {
  likes.value = likes.value.includes(code) ? likes.value.filter((c) => c !== code) : [...likes.value, code]
}

const canNext = computed(() => {
  switch (step.value) {
    case 1: return rating.value > 0
    case 2: return likes.value.length > 0
    case 3: return pains.value.length > 0
    case 4: return purpose.value !== ''
    case 5: return wish.value !== ''
    default: return true
  }
})

async function offerAfterLesson() {
  try {
    const res = await api.surveyOffer()
    if (!res.offer) return
    reward.value = res.reward
    reset()
    setTimeout(() => (open.value = true), SHOW_DELAY_MS)
  } catch {
    /* a survey is never worth an error on the screen that closes a lesson */
  }
}

// Closing without answering is final: the server already marked the offer as made.
async function dismiss() {
  open.value = false
  try {
    await api.dismissSurvey()
  } catch {
    /* nothing to do: the offer was already counted when it was made */
  }
}

async function submit() {
  if (busy.value || wish.value === '') return
  const answers: SurveyAnswers = {
    rating: rating.value,
    likes: likes.value,
    pains: pains.value,
    pain_text: painText.value.trim(),
    purpose: purpose.value,
    wish: wish.value,
  }
  busy.value = true
  error.value = null
  try {
    const res = await api.answerSurvey(answers)
    open.value = false
    const wallet = useWalletStore()
    await wallet.refresh()
    useRewardModal().celebrate(res.reward, 'Спасибо, что ответили! 💛', false, 'Опрос пройден')
  } catch (e) {
    error.value = authErrorMessage(e)
  } finally {
    busy.value = false
  }
}

export function useSurveyModal() {
  return {
    open, reward, step, busy, error, canNext,
    rating, likes, pains, painText, purpose, wish,
    toggleLike, togglePain, offerAfterLesson, dismiss, submit,
  }
}
