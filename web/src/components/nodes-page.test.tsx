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

test('hides duplicate Down history by default and reveals it through the node filter', async () => {
  const interaction = userEvent.setup()
  const readyNode = {
    ...swarmNode,
    id: 'readyabcdefghijklmnopqrs',
    hostname: 'worker-rejoined',
    address: '10.0.0.22',
    dockerVersion: '28.3.0',
    versionDrift: false,
  }
  const historicalNode = {
    ...readyNode,
    id: 'downabcdefghijklmnopqrst',
    status: 'down',
  }
  const offlineNode = {
    ...readyNode,
    id: 'offlineabcdefghijklmnopq',
    hostname: 'worker-offline',
    address: '10.0.0.23',
    status: 'down',
  }
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    if (String(input) === '/api/v1/nodes') {
      return jsonResponse({ nodes: [readyNode, historicalNode, offlineNode] })
    }
    throw new Error(`unexpected request: ${String(input)}`)
  }))

  render(<NodesPage canManage={false} />)

  expect(await screen.findByText(readyNode.id)).toBeInTheDocument()
  expect(screen.getByText(offlineNode.id)).toBeInTheDocument()
  expect(screen.queryByText(historicalNode.id)).not.toBeInTheDocument()
  expect(screen.getByText('Current 2')).toBeInTheDocument()
  expect(screen.getByText('All 3')).toBeInTheDocument()
  expect(screen.getByText(/1 historical Down record hidden/)).toBeInTheDocument()

  await interaction.click(screen.getByRole('radio', { name: 'Show all node records' }))

  expect(await screen.findByText(historicalNode.id)).toBeInTheDocument()
  expect(screen.getByText('historical')).toBeInTheDocument()
  expect(screen.getByText(/including 1 historical Down record/)).toBeInTheDocument()
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
