// SPDX-License-Identifier: AGPL-3.0-only

import { useEffect, useState } from 'react'
import {
  Box,
  Boxes,
  CircleGauge,
  Cpu,
  Database,
  HardDrive,
  LoaderCircle,
  LogOut,
  MemoryStick,
  Network,
  RefreshCw,
  Server,
  ShieldAlert,
} from 'lucide-react'

import { PageFrame } from '@/components/page-frame'
import { DeployForm } from '@/components/deploy-form'
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
import { Skeleton } from '@/components/ui/skeleton'
import {
  getCluster,
  logout,
  type ClusterSnapshot,
  type User,
  type VersionInfo,
} from '@/lib/api'

type DashboardPageProps = {
  user: User
  version: VersionInfo
  onLoggedOut: () => void
}

export function DashboardPage({ user, version, onLoggedOut }: DashboardPageProps) {
  const [cluster, setCluster] = useState<ClusterSnapshot | null>(null)
  const [error, setError] = useState('')
  const [refreshKey, setRefreshKey] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    setError('')

    getCluster(controller.signal)
      .then(setCluster)
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(loadError instanceof Error ? loadError.message : 'Unable to inspect Docker Engine.')
        }
      })

    return () => controller.abort()
  }, [refreshKey])

  async function handleLogout() {
    try {
      await logout()
    } finally {
      onLoggedOut()
    }
  }

  const actions = (
    <div className="flex items-center gap-2">
      <Badge variant="outline" className="hidden sm:inline-flex">{user.username} · {user.role}</Badge>
      <Button type="button" variant="outline" size="sm" onClick={handleLogout}>
        <LogOut data-icon="inline-start" />
        Sign out
      </Button>
    </div>
  )

  return (
    <PageFrame version={version} actions={actions}>
      <div className="flex flex-col gap-8">
        <section className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
          <div className="flex flex-col gap-2">
            <p className="text-sm font-medium text-primary">Control plane overview</p>
            <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">Your Swarm at a glance</h1>
            <p className="max-w-2xl text-muted-foreground">
              Live state reported by the Docker Engine connected to this Dock-Weaver instance.
            </p>
          </div>
          <Button type="button" variant="outline" onClick={() => setRefreshKey((key) => key + 1)}>
            <RefreshCw data-icon="inline-start" />
            Refresh
          </Button>
        </section>

        {error ? (
          <Alert variant="destructive">
            <ShieldAlert aria-hidden="true" />
            <AlertTitle>Docker Engine is unavailable</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        {!cluster && !error ? <DashboardSkeleton /> : null}
        {cluster ? <ClusterOverview cluster={cluster} /> : null}
        {cluster?.available && cluster.swarmState === 'active' ? <DeployForm /> : null}
      </div>
    </PageFrame>
  )
}

function ClusterOverview({ cluster }: { cluster: ClusterSnapshot }) {
  if (!cluster.available) {
    return (
      <Alert>
        <LoaderCircle aria-hidden="true" />
        <AlertTitle>Docker inspection is disabled</AlertTitle>
        <AlertDescription>{cluster.error ?? 'No Docker Engine is connected.'}</AlertDescription>
      </Alert>
    )
  }

  const memory = formatBytes(cluster.memoryBytes)
  const swarmActive = cluster.swarmState === 'active'

  return (
    <div className="flex flex-col gap-6">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard icon={Network} label="Swarm state" value={cluster.swarmState || 'unknown'} detail={`${cluster.nodes} node(s) · ${cluster.managers} manager(s)`} />
        <MetricCard icon={Cpu} label="CPU capacity" value={`${cluster.cpus} cores`} detail={cluster.architecture || 'Architecture unavailable'} />
        <MetricCard icon={MemoryStick} label="Memory capacity" value={memory} detail="Reported by this Docker host" />
        <MetricCard icon={Box} label="Workload" value={`${cluster.containersRunning} running`} detail={`${cluster.images} local image(s)`} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between gap-3">
              <CardTitle className="flex items-center gap-2"><Server aria-hidden="true" />Host</CardTitle>
              <Badge variant="secondary">{cluster.hostname || 'unknown host'}</Badge>
            </div>
            <CardDescription>Engine and operating-system identity.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <Detail label="Operating system" value={cluster.operatingSystem} />
            <Detail label="Kernel" value={cluster.kernelVersion} />
            <Detail label="Docker Engine" value={cluster.dockerVersion} />
            <Detail label="Docker API" value={cluster.dockerApiVersion} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between gap-3">
              <CardTitle className="flex items-center gap-2"><Boxes aria-hidden="true" />Swarm node</CardTitle>
              <Badge variant={swarmActive ? 'default' : 'secondary'}>{swarmActive ? 'Active' : 'Not initialized'}</Badge>
            </div>
            <CardDescription>Membership reported by the local Engine.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <Detail label="Role" value={cluster.nodeRole || 'none'} />
            <Detail label="Status" value={cluster.nodeStatus || 'unknown'} />
            <Detail label="Availability" value={cluster.availability || 'unknown'} />
            <Detail label="Manager status" value={cluster.managerStatus || 'not a manager'} />
            {cluster.nodeId ? <Detail className="sm:col-span-2" label="Node ID" value={cluster.nodeId} mono /> : null}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

type MetricCardProps = {
  icon: typeof CircleGauge
  label: string
  value: string
  detail: string
}

function MetricCard({ icon: Icon, label, value, detail }: MetricCardProps) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-3 text-muted-foreground">
          <CardDescription>{label}</CardDescription>
          <Icon aria-hidden="true" />
        </div>
        <CardTitle className="text-2xl capitalize">{value}</CardTitle>
      </CardHeader>
      <CardContent className="text-xs text-muted-foreground">{detail}</CardContent>
    </Card>
  )
}

function Detail({ label, value, mono, className }: { label: string; value: string; mono?: boolean; className?: string }) {
  return (
    <div className={`flex min-w-0 flex-col gap-1 ${className ?? ''}`}>
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
      <span className={mono ? 'truncate font-mono text-sm' : 'text-sm font-medium'} title={value}>{value || 'unknown'}</span>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label="Loading cluster state">
      {[HardDrive, Database, Cpu, Boxes].map((Icon, index) => (
        <Card key={index}>
          <CardHeader>
            <div className="flex justify-between"><Skeleton className="h-4 w-24" /><Icon className="text-muted-foreground" aria-hidden="true" /></div>
            <Skeleton className="h-8 w-32" />
          </CardHeader>
          <CardContent><Skeleton className="h-3 w-36" /></CardContent>
        </Card>
      ))}
    </div>
  )
}

function formatBytes(bytes: number) {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** exponent).toFixed(exponent > 2 ? 1 : 0)} ${units[exponent]}`
}
