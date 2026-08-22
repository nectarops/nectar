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

test('submits the optional management domain with the owner setup', async () => {
  const interaction = userEvent.setup()
  const onComplete = vi.fn()
  let submitted: unknown
  vi.stubGlobal('fetch', vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
    submitted = JSON.parse(String(init?.body))
    return jsonResponse({ id: 1, username: 'admin', role: 'owner' })
  }))

  render(<SetupPage version={version} onComplete={onComplete} />)

  await interaction.type(screen.getByLabelText('One-time setup token'), 'setup-token')
  await interaction.type(screen.getByLabelText('Password'), 'abcde')
  await interaction.type(screen.getByLabelText('Confirm password'), 'abcde')
  await interaction.type(screen.getByLabelText('Management domain (optional)'), 'nectar.example.com')
  await interaction.type(screen.getByLabelText("Let's Encrypt email"), 'ops@example.com')
  await interaction.click(
    screen.getByRole('button', { name: 'Create owner and configure HTTPS' }),
  )

  await waitFor(() => expect(onComplete).toHaveBeenCalledTimes(1))
  expect(submitted).toEqual({
    initToken: 'setup-token',
    username: 'admin',
    password: 'abcde',
    domain: 'nectar.example.com',
    acmeEmail: 'ops@example.com',
  })
})

test('requires the domain and ACME email together', async () => {
  const interaction = userEvent.setup()
  const fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)

  render(<SetupPage version={version} onComplete={vi.fn()} />)

  await interaction.type(screen.getByLabelText('One-time setup token'), 'setup-token')
  await interaction.type(screen.getByLabelText('Password'), 'abcde')
  await interaction.type(screen.getByLabelText('Confirm password'), 'abcde')
  await interaction.type(screen.getByLabelText('Management domain (optional)'), 'nectar.example.com')
  await interaction.click(
    screen.getByRole('button', { name: 'Create owner and configure HTTPS' }),
  )

  expect(
    screen.getByText("Management domain and Let's Encrypt email must be provided together."),
  ).toBeInTheDocument()
  expect(fetchMock).not.toHaveBeenCalled()
})

function jsonResponse(value: unknown) {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}
