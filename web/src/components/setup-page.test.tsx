// SPDX-License-Identifier: AGPL-3.0-only

import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'

import { SetupPage } from '@/components/setup-page'

const version = {
  version: '0.1.0',
  commit: 'test',
  sourceUrl: 'https://example.test/source',
}

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

test('creates only the owner account during initial setup', async () => {
  const interaction = userEvent.setup()
  const onComplete = vi.fn()
  let submitted: unknown
  vi.stubGlobal('fetch', vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
    submitted = JSON.parse(String(init?.body))
    return jsonResponse({ id: 1, username: 'admin', role: 'owner' })
  }))

  render(<SetupPage version={version} onComplete={onComplete} />)

  expect(screen.queryByLabelText('Management domain (optional)')).not.toBeInTheDocument()

  await interaction.type(screen.getByLabelText('One-time setup token'), 'setup-token')
  await interaction.type(screen.getByLabelText('Password'), 'abcde')
  await interaction.type(screen.getByLabelText('Confirm password'), 'abcde')
  await interaction.click(screen.getByRole('button', { name: 'Create owner account' }))

  await waitFor(() => expect(onComplete).toHaveBeenCalledTimes(1))
  expect(submitted).toEqual({
    initToken: 'setup-token',
    username: 'admin',
    password: 'abcde',
  })
})


function jsonResponse(value: unknown) {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}
