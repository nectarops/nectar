// SPDX-License-Identifier: AGPL-3.0-only

import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, test, vi } from 'vitest'

import { DashboardPage } from '@/components/dashboard-page'
import type { ClusterSnapshot } from '@/lib/api'

const user = { id: 1, username: 'admin', role: 'owner' }
const version = {
  version: '0.1.0',
  commit: 'test',
  sourceUrl: 'https://example.test/source',
}

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

test('moves the deployment form behind the sidebar navigation', async () => {
  const interaction = userEvent.setup()
  stubCluster({
    ...clusterSnapshot,
    swarmState: 'active',
    managers: 1,
    nodes: 1,
  })

  render(
    <DashboardPage
      user={user}
      version={version}
      onLoggedOut={vi.fn()}
    />,
  )

  expect(
    await screen.findByRole('heading', { name: 'Your Swarm at a glance' }),
  ).toBeInTheDocument()
  expect(screen.queryByRole('heading', { name: 'Deploy a service' })).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Overview' })).toHaveAttribute(
    'aria-current',
    'page',
  )

  await interaction.click(screen.getByRole('button', { name: 'Deploy service' }))

  expect(screen.getByRole('heading', { name: 'Deploy services' })).toBeInTheDocument()
  expect(screen.getByRole('heading', { name: 'Deploy a service' })).toBeInTheDocument()
  expect(
    screen.queryByRole('heading', { name: 'Your Swarm at a glance' }),
  ).not.toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Deploy service' })).toHaveAttribute(
    'aria-current',
    'page',
  )

  await interaction.click(screen.getByRole('button', { name: 'HTTPS access' }))

  expect(screen.getByRole('heading', { name: 'Configure HTTPS access' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'HTTPS access' })).toHaveAttribute(
    'aria-current',
    'page',
  )
})

test('explains why deployment is unavailable when Swarm is inactive', async () => {
  const interaction = userEvent.setup()
  stubCluster(clusterSnapshot)

  render(
    <DashboardPage
      user={user}
      version={version}
      onLoggedOut={vi.fn()}
    />,
  )

  await screen.findByRole('heading', { name: 'Your Swarm at a glance' })
  await interaction.click(screen.getByRole('button', { name: 'Deploy service' }))

  expect(screen.getByText('Swarm must be active before deploying')).toBeInTheDocument()
  expect(
    screen.getByRole('button', { name: 'Refresh cluster status' }),
  ).toBeInTheDocument()
  expect(screen.queryByRole('heading', { name: 'Deploy a service' })).not.toBeInTheDocument()
})

test('warns when the Manager Docker version differs from the cluster target', async () => {
  stubCluster({
    ...clusterSnapshot,
    dockerVersion: '29.0.2',
  })

  render(
    <DashboardPage
      user={user}
      version={version}
      onLoggedOut={vi.fn()}
    />,
  )

  expect(await screen.findByText('Docker version policy mismatch')).toBeInTheDocument()
  expect(screen.getByText(/Manager runs Docker 29\.0\.2/)).toBeInTheDocument()
  expect(screen.getByText('28.3.0')).toBeInTheDocument()
})

function stubCluster(cluster: ClusterSnapshot) {
  vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
    if (String(input).endsWith('/api/v1/management-access')) {
      return jsonResponse({ domain: '', acmeEmail: '' })
    }
    return jsonResponse(cluster)
  }))
}

function jsonResponse(value: unknown) {
  return new Response(JSON.stringify(value), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

const clusterSnapshot: ClusterSnapshot = {
  available: true,
  hostname: 'docker-desktop',
  operatingSystem: 'Docker Desktop',
  architecture: 'aarch64',
  kernelVersion: '6.10.14-linuxkit',
  dockerVersion: '28.3.0',
  desiredDockerVersion: '28.3.0',
  dockerApiVersion: '1.51',
  swarmState: 'inactive',
  nodeId: '',
  managerStatus: '',
  nodeRole: 'standalone',
  nodeStatus: 'unknown',
  availability: 'unknown',
  managers: 0,
  nodes: 0,
  cpus: 8,
  memoryBytes: 8 * 1024 ** 3,
  containersRunning: 1,
  images: 1,
}
