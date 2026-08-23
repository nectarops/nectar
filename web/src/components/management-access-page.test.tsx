// SPDX-License-Identifier: AGPL-3.0-only

import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'

import { ManagementAccessPage } from '@/components/management-access-page'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

test('installs Traefik after the owner submits management access', async () => {
  const interaction = userEvent.setup()
  let submitted: unknown
  vi.stubGlobal('fetch', vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
    if (!init?.method) {
      return jsonResponse({ domain: '', acmeEmail: '' })
    }
    submitted = JSON.parse(String(init.body))
    return jsonResponse({ domain: 'nectar.example.com', acmeEmail: 'ops@example.com' })
  }))

  render(<ManagementAccessPage />)

  await screen.findByRole('button', { name: 'Install Traefik and enable HTTPS' })
  await interaction.type(screen.getByLabelText('Management domain'), 'nectar.example.com')
  await interaction.type(screen.getByLabelText("Let's Encrypt email"), 'ops@example.com')
  await interaction.click(
    screen.getByRole('button', { name: 'Install Traefik and enable HTTPS' }),
  )

  await waitFor(() => {
    expect(screen.getByText('HTTPS access is configured')).toBeInTheDocument()
  })
  expect(submitted).toEqual({
    domain: 'nectar.example.com',
    acmeEmail: 'ops@example.com',
  })
  expect(screen.getByRole('link', { name: 'https://nectar.example.com' })).toHaveAttribute(
    'href',
    'https://nectar.example.com',
  )
})

function jsonResponse(value: unknown) {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}
