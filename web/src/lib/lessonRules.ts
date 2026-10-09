// A wrong answer in a lesson is not a dead end and not a free pass: the
// exercise goes to the end of its step and has to be answered again (see
// LessonView). While that rule is on, the "Переделать" button is hidden after
// a mistake — otherwise it is one tap to peek at the right answer and retype
// it on the spot, which is exactly what sending the exercise to the end
// prevents. The answer components read the flag from here instead of taking a
// prop, because they are nested three levels deep and the dialogue uses them
// too.
import { inject, provide, ref, type InjectionKey, type Ref } from 'vue'

const KEY: InjectionKey<Ref<boolean>> = Symbol('wrongGoesToEnd')

export function provideWrongGoesToEnd(on: Ref<boolean>) {
  provide(KEY, on)
}

export function useWrongGoesToEnd(): Ref<boolean> {
  return inject(KEY, ref(false))
}
