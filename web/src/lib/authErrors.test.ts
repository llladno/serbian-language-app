import { describe, it, expect } from 'vitest'
import { authErrorMessage, isEmailUnverified } from './authErrors'
import { ApiError } from '../api'

describe('authErrorMessage', () => {
  it('maps a known English code to Russian', () => {
    expect(authErrorMessage(new ApiError(403, 'wrong_password'))).toBe('Неверный пароль')
  })
  it('passes an already-Russian message through untouched', () => {
    expect(authErrorMessage(new ApiError(401, 'неверная почта или пароль'))).toBe(
      'неверная почта или пароль',
    )
  })
  it('falls back to a generic message for an unrecognized code', () => {
    expect(authErrorMessage(new ApiError(500, 'some new backend string'))).toBe(
      'Что-то пошло не так, попробуйте ещё раз',
    )
  })
  it('falls back for a non-ApiError', () => {
    expect(authErrorMessage(new Error('network down'))).toBe('Что-то пошло не так, попробуйте ещё раз')
  })
})

describe('isEmailUnverified', () => {
  it('is true only for the exact 403 email_unverified shape', () => {
    expect(isEmailUnverified(new ApiError(403, 'email_unverified'))).toBe(true)
    expect(isEmailUnverified(new ApiError(401, 'email_unverified'))).toBe(false)
    expect(isEmailUnverified(new ApiError(403, 'wrong_password'))).toBe(false)
  })
})
