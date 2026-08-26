// SPDX-License-Identifier: AGPL-3.0-only

import { Database, HardDrive, Info, RefreshCw } from 'lucide-react'

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

type StorageViewProps = {
  cluster: ClusterSnapshot | null
  onRefresh: () => void
}

export function StorageView({ cluster, onRefresh }: StorageViewProps) {
  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Infrastructure</p>
          <h1 className="text-3xl font-semibold tracking-tight text-neutral-900">S3 Storage & Volumes</h1>
          <p className="max-w-2xl text-sm text-neutral-500">
            Persistent Swarm data volumes, SQLite WAL database, and S3 backup integration.
          </p>
        </div>
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
      </div>

      <Alert>
        <Info aria-hidden="true" />
        <AlertTitle>Volume Replication in Docker Swarm</AlertTitle>
        <AlertDescription>
          Standard Docker Swarm volumes are node-local. For distributed multi-node state, bind services to persistent volume plugins or external S3-compatible storage.
        </AlertDescription>
      </Alert>

      <div className="grid gap-6 sm:grid-cols-2">
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <Database className="size-4 text-neutral-600" />
                Nectar SQLite Database
              </CardTitle>
              <Badge variant="outline" className="border-emerald-200 bg-emerald-50 text-emerald-700">
                WAL Mode Active
              </Badge>
            </div>
            <CardDescription>Embedded state store for node enrollments and deployments.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Location</span>
              <span className="font-mono text-xs text-neutral-800">/var/lib/nectar/nectar.db</span>
            </div>
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Journal Mode</span>
              <span className="font-medium text-neutral-800">Write-Ahead Logging</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Migrations</span>
              <span className="font-medium text-neutral-800">Up-to-date</span>
            </div>
          </CardContent>
        </Card>

        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <HardDrive className="size-4 text-neutral-600" />
                Local Docker Volumes
              </CardTitle>
              <Badge variant="secondary">
                {cluster?.containersRunning ? `${cluster.containersRunning} running` : '1 active'}
              </Badge>
            </div>
            <CardDescription>Persistent storage volumes mounted by services on this host.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Volume Driver</span>
              <span className="font-medium text-neutral-800">local</span>
            </div>
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Scope</span>
              <span className="font-medium text-neutral-800">Local node</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Images Cached</span>
              <span className="font-medium text-neutral-800">{cluster?.images ?? 1} images</span>
            </div>
          </CardContent>
        </Card>
      </div>
    </section>
  )
}
