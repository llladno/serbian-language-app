// Account = just a name, stored locally and mirrored in the backend DB.
const KEY = 'srpski.account'

export function getAccount(): string {
  try {
    return localStorage.getItem(KEY) ?? ''
  } catch {
    return ''
  }
}

export function setAccount(name: string) {
  try {
    localStorage.setItem(KEY, name)
  } catch {
    /* ignore */
  }
  // Reload so every store refetches state for the new account.
  location.assign('/')
}

export function clearAccount() {
  try {
    localStorage.removeItem(KEY)
  } catch {
    /* ignore */
  }
  location.assign('/')
}
