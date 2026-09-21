// Maps the backend's error vocabulary to Russian UI text. auth.go/me.go mix a
// few already-Russian sentences (the login 401) with literal English codes
// meant for the client to switch on ("wrong_password", "email_unverified").
// Anything unrecognized falls back to a generic message — never leak an
// English backend string into this Russian-only UI.
import { ApiError } from '../api'

const MESSAGES: Record<string, string> = {
  'invalid email': 'Некорректный email',
  'password must be 8 to 128 characters': 'Пароль должен быть от 8 до 128 символов',
  'password must be 8-128 characters': 'Пароль должен быть от 8 до 128 символов',
  'name must be 1 to 40 characters': 'Имя должно быть от 1 до 40 символов',
  'message must not be empty': 'Напишите сообщение перед отправкой',
  'bad request body': 'Некорректный запрос',
  'too many attempts': 'Слишком много попыток, попробуйте позже',
  wrong_password: 'Неверный пароль',
  email_required: 'Укажите email',
  invalid_token: 'Ссылка недействительна или устарела',
  reauth_required: 'Нужен свежий вход — войдите ещё раз и повторите',
  telegram_disabled: 'Вход через Telegram пока не настроен',
  bad_telegram_auth: 'Не удалось подтвердить вход через Telegram',
  telegram_taken: 'Этот Telegram уже привязан к другому аккаунту',
  only_login_method: 'Это единственный способ входа — сначала задайте пароль',
  not_linked: 'Telegram не привязан',
  'no session': 'Сессия истекла — войдите снова',
  'session required': 'Сессия истекла — войдите снова',
  'internal error': 'Что-то пошло не так, попробуйте ещё раз',
}

const FALLBACK = 'Что-то пошло не так, попробуйте ещё раз'

export function authErrorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    // login's bad-credentials body is already a finished Russian sentence
    // (badCredentials in auth.go) — pass anything Cyrillic through untouched.
    if (/[а-яё]/i.test(e.message)) return e.message
    return MESSAGES[e.message] ?? FALLBACK
  }
  return FALLBACK
}

// email_unverified is routed to /verify rather than shown inline, so callers
// check for it before falling back to authErrorMessage.
export function isEmailUnverified(e: unknown): boolean {
  return e instanceof ApiError && e.status === 403 && e.message === 'email_unverified'
}
