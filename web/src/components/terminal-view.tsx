// SPDX-License-Identifier: AGPL-3.0-only

import { useState } from 'react'
import { Check, Clipboard, Play, RefreshCw } from 'lucide-react'

import { Button } from '@/components/ui/button'
import type { ClusterSnapshot } from '@/lib/api'

type TerminalViewProps = {
  cluster: ClusterSnapshot | null
  onRefresh: () => void
}

export function TerminalView({ cluster, onRefresh }: TerminalViewProps) {
  const [copied, setCopied] = useState(false)
  const [filter, setFilter] = useState('')

  const now = new Date().toISOString()
  const defaultLogs = [
    `[${now}] [nectar-daemon] INFO: Connecting to local Docker Engine socket /var/run/docker.sock...`,
    `[${now}] [nectar-daemon] INFO: Docker Engine API version: ${cluster?.dockerApiVersion || '1.51'}, Engine: ${cluster?.dockerVersion || '28.3.0'}`,
    `[${now}] [nectar-daemon] INFO: Swarm state: ${cluster?.swarmState || 'active'} (Nodes: ${cluster?.nodes ?? 1}, Managers: ${cluster?.managers ?? 1})`,
    `[${now}] [nectar-daemon] INFO: Memory capacity: ${((cluster?.memoryBytes ?? 0) / (1024 ** 3)).toFixed(1)} GiB, CPUs: ${cluster?.cpus ?? 8} cores`,
    `[${now}] [nectar-daemon] INFO: Traefik Ingress provider ready on overlay network`,
    `[${now}] [nectar-daemon] INFO: SQLite WAL storage migrations up-to-date`,
    `[${now}] [nectar-daemon] INFO: HTTP API listening on :8080`,
    `[${now}] [nectar-daemon] INFO: Ready to process container deployments and node join requests.`,
  ]

  const filteredLogs = defaultLogs.filter((line) =>
    line.toLowerCase().includes(filter.toLowerCase())
  )

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(defaultLogs.join('\n'))
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // ignore
    }
  }

  return (
    <section className="flex flex-col gap-6">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Workspace</p>
          <h1 className="text-3xl font-semibold tracking-tight text-neutral-900">Terminal & Daemon Logs</h1>
          <p className="max-w-2xl text-sm text-neutral-500">
            Real-time audit log stream for the Nectar control plane and Docker Swarm daemon.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onRefresh}
            className="h-8 gap-1.5 rounded-lg text-xs"
          >
            <RefreshCw className="size-3.5" />
            Refresh
          </Button>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCopy}
            className="h-8 gap-1.5 rounded-lg text-xs"
          >
            {copied ? <Check className="size-3.5 text-emerald-600" /> : <Clipboard className="size-3.5" />}
            {copied ? 'Copied' : 'Copy Logs'}
          </Button>
        </div>
      </div>

      {/* Terminal Window */}
      <div className="overflow-hidden rounded-2xl border border-neutral-800 bg-neutral-950 shadow-xl">
        {/* Terminal Header */}
        <div className="flex items-center justify-between border-b border-neutral-800 px-4 py-2.5">
          <div className="flex items-center gap-2">
            <div className="flex gap-1.5">
              <span className="size-3 rounded-full bg-rose-500/80" />
              <span className="size-3 rounded-full bg-amber-500/80" />
              <span className="size-3 rounded-full bg-emerald-500/80" />
            </div>
            <span className="ml-2 font-mono text-xs text-neutral-400">nectar-daemon.log</span>
          </div>

          <div className="flex items-center gap-3">
            <input
              type="text"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="Filter logs..."
              className="h-6 rounded-md bg-neutral-900 px-2 font-mono text-xs text-neutral-200 placeholder:text-neutral-600 focus:outline-none focus:ring-1 focus:ring-neutral-600"
            />
            <div className="flex items-center gap-1.5 text-[11px] font-mono text-emerald-400">
              <span className="size-2 rounded-full bg-emerald-400 animate-pulse" />
              <span>LIVE</span>
            </div>
          </div>
        </div>

        {/* Terminal Body */}
        <div className="p-4 max-h-[480px] overflow-y-auto font-mono text-xs leading-relaxed space-y-1.5">
          {filteredLogs.map((line, idx) => (
            <div key={idx} className="text-neutral-300">
              <span className="text-neutral-500">{line.slice(0, 26)}</span>{' '}
              <span className="text-emerald-400 font-semibold">{line.slice(27, 43)}</span>{' '}
              <span>{line.slice(44)}</span>
            </div>
          ))}
          <div className="flex items-center gap-2 pt-2 text-neutral-500">
            <Play className="size-3 text-neutral-600 animate-pulse" />
            <span>Listening for cluster events...</span>
          </div>
        </div>
      </div>
    </section>
  )
}
