// The learner's script, as a composable. A component paints Serbian text with
// show(text) and nothing else: values, ids and audio keys stay Latin.
//
// A lesson about the letters themselves (`script: latin` in its manifest,
// Lesson.keep_latin) calls provideKeepLatin() so everything inside it stays
// Latin whatever the account says: in Cyrillic its table "č = ч" would read
// "ч = ч".
import { computed, inject, provide, ref, type InjectionKey, type Ref } from 'vue'
import { useSessionStore } from '../stores/session'
import { toScript, type Script } from './script'

const KEEP_LATIN: InjectionKey<Ref<boolean>> = Symbol('keepLatin')

export function provideKeepLatin(on: Ref<boolean>) {
  provide(KEEP_LATIN, on)
}

export function useScript() {
  const session = useSessionStore()
  const keepLatin = inject(KEEP_LATIN, ref(false))
  const script = computed<Script>(() => (keepLatin.value ? 'latin' : (session.user?.script ?? 'latin')))
  const show = (text: string | null | undefined) => toScript(text, script.value)
  return { script, show }
}
