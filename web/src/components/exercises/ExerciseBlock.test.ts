import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ExerciseBlock from './ExerciseBlock.vue'
import { speakClip } from '../../lib/sound'
import type { ExerciseBlock as Block } from '../../types'

vi.mock('../../lib/sound', () => ({ speakClip: vi.fn() }))

// Every exercise of a block stays mounted (v-show). A clip that started on mount
// played for every listening exercise of the step at once, and a tap on the
// speaker then sounded over another exercise's phrase (a real report, 2026-10-27).
const block = {
  instruction: '',
  exercises: [
    { id: 'a', type: 'listen', prompt: 'Запиши', audio: 'a.mp3' },
    { id: 'b', type: 'choice', prompt: 'Что ты услышал?', options: ['да', 'нет'], audio: 'b.mp3' },
    { id: 'c', type: 'listen', prompt: 'Запиши', audio: 'c.mp3' },
  ],
} as unknown as Block

describe('ExerciseBlock plays a clip only for the exercise on screen', () => {
  beforeEach(() => vi.mocked(speakClip).mockClear())

  it('starts only the active one, and the next when it comes up', async () => {
    const w = mount(ExerciseBlock, { props: { lesson: '01', block, activeId: 'a' } })
    expect(vi.mocked(speakClip).mock.calls).toEqual([['a.mp3']])

    await w.setProps({ activeId: 'b' })
    await w.setProps({ activeId: 'c' })
    expect(vi.mocked(speakClip).mock.calls).toEqual([['a.mp3'], ['b.mp3'], ['c.mp3']])
  })
})
