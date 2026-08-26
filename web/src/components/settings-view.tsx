// SPDX-License-Identifier: AGPL-3.0-only

import { Code2, Cpu, Database, ExternalLink, Shield } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import type { ClusterSnapshot, VersionInfo } from '@/lib/api'

type SettingsViewProps = {
  cluster: ClusterSnapshot | null
  version?: VersionInfo
}

export function SettingsView({ cluster, version }: SettingsViewProps) {
  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col gap-1.5">
        <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Manage</p>
        <h1 className="text-3xl font-semibold tracking-tight text-neutral-900">Cluster Settings</h1>
        <p className="max-w-2xl text-sm text-neutral-500">
          Docker Engine version policies, SQLite database maintenance, and Nectar platform configuration.
        </p>
      </div>

      <Alert>
        <Shield className="size-4 text-emerald-600" aria-hidden="true" />
        <AlertTitle>Manager Quorum Protection</AlertTitle>
        <AlertDescription>
          Nectar enforces Raft quorum safety rules. Node demotions or removals that would leave fewer than 3 active managers require explicit confirmation.
        </AlertDescription>
      </Alert>

      <div className="grid gap-6 sm:grid-cols-2">
        {/* Docker Engine Version Policy */}
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <Cpu className="size-4 text-neutral-700" />
                Docker Engine Target
              </CardTitle>
              <Badge variant="outline" className="border-neutral-300 font-mono text-xs">
                {cluster?.desiredDockerVersion || '28.3.0'}
              </Badge>
            </div>
            <CardDescription>Baseline Docker Engine version installed on enrolled nodes.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Target Engine</span>
              <span className="font-mono font-medium text-neutral-800">
                {cluster?.desiredDockerVersion || '28.3.0'}
              </span>
            </div>
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Local Manager Engine</span>
              <span className="font-mono text-neutral-800">{cluster?.dockerVersion || '28.3.0'}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Auto-Upgrade Drift</span>
              <span className="font-medium text-neutral-700">Manual review required</span>
            </div>
          </CardContent>
        </Card>

        {/* Database & Persistence */}
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <Database className="size-4 text-neutral-700" />
                Persistence & Storage
              </CardTitle>
              <Badge variant="outline" className="border-emerald-200 bg-emerald-50 text-emerald-700 text-[11px]">
                SQLite WAL
              </Badge>
            </div>
            <CardDescription>Local transaction journal and database settings.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Database File</span>
              <span className="font-mono text-xs text-neutral-800">/var/lib/nectar/nectar.db</span>
            </div>
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Auto-Checkpoint</span>
              <span className="font-medium text-neutral-800">1000 pages (Passive)</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Schema Migrations</span>
              <span className="font-medium text-emerald-700">Applied (v3)</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Platform & License Info */}
      <Card className="rounded-2xl border border-neutral-200/80 bg-white">
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-base font-semibold">Nectar Platform Information</CardTitle>
            <Badge variant="secondary" className="font-mono text-xs">
              AGPL-3.0-only
            </Badge>
          </div>
          <CardDescription>
            Free and open-source self-hosted Docker Swarm control plane.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-3 text-sm">
            <div className="rounded-xl bg-neutral-50 p-3 border border-neutral-100">
              <span className="text-xs text-neutral-500 block">Current Version</span>
              <span className="font-semibold text-neutral-900 mt-1 block">
                {version ? `v${version.version}` : 'v0.1.0'}
              </span>
            </div>
            <div className="rounded-xl bg-neutral-50 p-3 border border-neutral-100">
              <span className="text-xs text-neutral-500 block">Commit Hash</span>
              <span className="font-mono text-xs text-neutral-900 mt-1 block truncate">
                {version?.commit || 'dev-build'}
              </span>
            </div>
            <div className="rounded-xl bg-neutral-50 p-3 border border-neutral-100">
              <span className="text-xs text-neutral-500 block">Architecture</span>
              <span className="font-medium text-neutral-900 mt-1 block">
                {cluster?.architecture || 'x86_64 / arm64'}
              </span>
            </div>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3 border-t border-neutral-100 pt-4 text-xs text-neutral-500">
            <span>
              Licensed under GNU Affero General Public License v3.0 (AGPL-3.0).
            </span>
            {version?.sourceUrl && (
              <a
                href={version.sourceUrl}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 font-medium text-neutral-900 underline hover:text-neutral-700"
              >
                <Code2 className="size-3.5" />
                <span>View Source Code</span>
                <ExternalLink className="size-3" />
              </a>
            )}
          </div>
        </CardContent>
      </Card>
    </section>
  )
}
