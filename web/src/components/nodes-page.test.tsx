// SPDX-License-Identifier: AGPL-3.0-only

import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'

import { NodesPage } from '@/components/nodes-page'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

test('shows live nodes and generates a Manager enrollment command', async () => {
  const interaction = userEvent.setup()
  const eventSources: FakeEventSource[] = []
  vi.stubGlobal('EventSource', class extends FakeEventSource {
    constructor(url: string) {
      super(url)
      eventSources.push(this)
    }
  })
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const path = String(input)
    if (path === '/api/v1/nodes') {
      return jsonResponse({ nodes: [swarmNode] })
    }
    if (path === '/api/v1/node-enrollments' && init?.method === 'POST') {
      return jsonResponse({
        enrollment,
        command: "curl -fsSL 'https://nectar.example/client.sh' | sudo env NECTAR_SERVER_URL='https://nectar.example' bash -s -- 'secret-token'",
      }, 201)
    }
    if (path === '/api/v1/node-enrollments') {
      return jsonResponse({ enrollments: [] })
    }
    throw new Error(`unexpected request: ${path}`)
  }))

  render(<NodesPage canManage />)

  expect(await screen.findByText('worker-1')).toBeInTheDocument()
  expect(screen.getByText('Docker version drift')).toBeInTheDocument()
  await interaction.click(screen.getByRole('radio', { name: 'Manager' }))
  expect(screen.getByText('Manager promotion is deliberately stricter')).toBeInTheDocument()

  await interaction.click(screen.getByRole('button', { name: 'Generate enrollment command' }))

  expect(await screen.findByText('Run once on the target host')).toBeInTheDocument()
  expect(screen.getByText(/client\.sh/)).toBeInTheDocument()
  await waitFor(() => expect(eventSources).toHaveLength(1))
  expect(eventSources[0]?.url).toContain(`${enrollment.id}/events`)
})

class FakeEventSource {
  static readonly CLOSED = 2
  readonly url: string
  readyState = 1
  onerror: ((event: Event) => void) | null = null

  constructor(url: string) {
    this.url = url
  }

  addEventListener() {}
  close() {
    this.readyState = FakeEventSource.CLOSED
  }
}

function jsonResponse(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

const swarmNode = {
  id: 'abcdefghijklmnopqrstuvwxy',
  hostname: 'worker-1',
  role: 'worker',
  status: 'ready',
  availability: 'active',
  address: '10.0.0.12',
  operatingSystem: 'linux',
  architecture: 'amd64',
  dockerVersion: '27.5.1',
  desiredDockerVersion: '28.3.0',
  versionDrift: true,
}

const enrollment = {
  id: 'ne_abcdefghijklmnopqrstuvwx',
  requestedRole: 'manager',
  status: 'pending',
  message: 'Enrollment command created',
  expiresAt: '2026-08-24T12:30:00Z',
  createdBy: 1,
  createdAt: '2026-08-24T12:00:00Z',
  updatedAt: '2026-08-24T12:00:00Z',
}
