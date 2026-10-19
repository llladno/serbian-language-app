// Every component test gets an active (empty) Pinia, so a component that reads
// the signed-in learner (useScript) works without each test wiring one up. A
// test that needs its own store still calls setActivePinia(createPinia()) and
// simply replaces this one.
import { beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

beforeEach(() => setActivePinia(createPinia()))
