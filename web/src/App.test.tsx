// SPDX-License-Identifier: AGPL-3.0-only

import { render, screen } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import App from '@/App'

afterEach(() => {
  vi.unstubAllGlobals()
})

test('shows setup when the instance is not configured', async () => {
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input)
    if (path.endsWith('/api/v1/setup/status')) {
      return jsonResponse({ completed: false })
    }
    if (path.endsWith('/api/v1/version')) {
      return jsonResponse({ version: '0.1.0', commit: 'test', sourceUrl: 'https://example.test/source' })
    }
    return jsonResponse({ error: { code: 'unauthenticated', message: 'authentication required' } }, 401)
  }))

  render(<App />)

  expect(await screen.findByRole('heading', { name: 'Secure your control plane' })).toBeInTheDocument()
  expect(screen.getByLabelText('One-time setup token')).toBeInTheDocument()
})

test('shows login when setup is complete and there is no session', async () => {
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input)
    if (path.endsWith('/api/v1/setup/status')) {
      return jsonResponse({ completed: true })
    }
    if (path.endsWith('/api/v1/version')) {
      return jsonResponse({ version: '0.1.0', commit: 'test', sourceUrl: 'https://example.test/source' })
    }
    return jsonResponse({ error: { code: 'unauthenticated', message: 'authentication required' } }, 401)
  }))

  render(<App />)

  expect(await screen.findByRole('heading', { name: 'Welcome back' })).toBeInTheDocument()
})

function jsonResponse(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}
