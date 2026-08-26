// SPDX-License-Identifier: AGPL-3.0-only

import { Globe, Layers, Network as NetworkIcon, Plus, RefreshCw, ShieldCheck } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import type { ClusterSnapshot } from '@/lib/api'

type NetworkViewProps = {
  cluster: ClusterSnapshot | null
  onRefresh: () => void
}

export function NetworkView({ cluster, onRefresh }: NetworkViewProps) {
  const swarmActive = cluster?.swarmState === 'active'

  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Infrastructure</p>
          <h1 className="text-3xl font-semibold tracking-tight text-neutral-900">Networks & Routing</h1>
          <p className="max-w-2xl text-sm text-neutral-500">
            Docker Swarm overlay networks, Traefik ingress routing, and service mesh connectivity.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onRefresh}
            className="gap-1.5 rounded-lg text-xs"
          >
            <RefreshCw className="size-3.5" />
            Refresh
          </Button>
          <Button
            type="button"
            className="gap-2 rounded-xl bg-neutral-900 text-white hover:bg-neutral-800"
          >
            <Plus className="size-4" />
            <span>Create Network</span>
          </Button>
        </div>
      </div>

      <Alert>
        <ShieldCheck className="size-4 text-emerald-600" aria-hidden="true" />
        <AlertTitle>Encrypted Swarm Overlay Networks</AlertTitle>
        <AlertDescription>
          Multi-host container communication uses Docker VXLAN overlay encryption with IPSec data plane tunnels.
        </AlertDescription>
      </Alert>

      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {/* Nectar Public Ingress Network */}
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <Globe className="size-4 text-neutral-700" />
                nectar-public
              </CardTitle>
              <Badge variant="outline" className="border-emerald-200 bg-emerald-50 text-emerald-700 text-[11px]">
                Attachable
              </Badge>
            </div>
            <CardDescription>Traefik HTTPS ingress network for published services.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-xs">
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Driver</span>
              <span className="font-mono font-medium text-neutral-800">overlay</span>
            </div>
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Scope</span>
              <span className="font-medium text-neutral-800">Swarm Cluster</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Subnet</span>
              <span className="font-mono text-neutral-700">10.0.1.0/24</span>
            </div>
          </CardContent>
        </Card>

        {/* Ingress Network */}
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <Layers className="size-4 text-neutral-700" />
                ingress
              </CardTitle>
              <Badge variant="secondary" className="text-[11px]">
                System
              </Badge>
            </div>
            <CardDescription>Default Docker Swarm routing mesh network.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-xs">
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Driver</span>
              <span className="font-mono font-medium text-neutral-800">overlay</span>
            </div>
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Routing Mesh</span>
              <span className="font-medium text-neutral-800">Active (VIP mode)</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Status</span>
              <span className="font-medium text-emerald-700">
                {swarmActive ? 'Connected' : 'Pending Swarm'}
              </span>
            </div>
          </CardContent>
        </Card>

        {/* Bridge Network */}
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <NetworkIcon className="size-4 text-neutral-700" />
                docker_gwbridge
              </CardTitle>
              <Badge variant="secondary" className="text-[11px]">
                Host Gateway
              </Badge>
            </div>
            <CardDescription>Gateway bridge for container outbound connectivity.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2 text-xs">
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Driver</span>
              <span className="font-mono font-medium text-neutral-800">bridge</span>
            </div>
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Scope</span>
              <span className="font-medium text-neutral-800">Local node</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">IPv4 Forwarding</span>
              <span className="font-medium text-emerald-700">Enabled</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </section>
  )
}
