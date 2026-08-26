// SPDX-License-Identifier: AGPL-3.0-only

import { Boxes, CheckCircle2, Plus, Rocket, Settings2 } from 'lucide-react'

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

type ProjectsViewProps = {
  cluster: ClusterSnapshot | null
  onNavigateToDeploy: () => void
}

export function ProjectsView({ cluster, onNavigateToDeploy }: ProjectsViewProps) {
  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Workspace</p>
          <h1 className="text-3xl font-semibold tracking-tight text-neutral-900">Projects & Services</h1>
          <p className="max-w-2xl text-sm text-neutral-500">
            Manage your Docker Swarm application stacks, services, and live environments.
          </p>
        </div>
        <Button
          type="button"
          onClick={onNavigateToDeploy}
          className="gap-2 rounded-xl bg-neutral-900 text-white hover:bg-neutral-800"
        >
          <Plus className="size-4" />
          <span>New Service</span>
        </Button>
      </div>

      {/* Projects Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {/* Core Nectar Stack Card */}
        <Card className="group relative overflow-hidden rounded-2xl border border-neutral-200/80 bg-white transition-all hover:border-neutral-300 hover:shadow-md">
          <CardHeader className="p-5 pb-3">
            <div className="flex items-start justify-between gap-3">
              <div className="flex items-center gap-3">
                <span className="flex size-10 items-center justify-center rounded-xl border border-neutral-200 bg-neutral-50 text-neutral-800 group-hover:bg-neutral-100 transition-colors">
                  <Boxes className="size-5" />
                </span>
                <div>
                  <CardTitle className="text-base font-semibold text-neutral-900">
                    nectar-control
                  </CardTitle>
                  <CardDescription className="text-xs text-neutral-500">
                    Nectar Control Plane & Dashboard
                  </CardDescription>
                </div>
              </div>
              <Badge variant="outline" className="border-emerald-200 bg-emerald-50 text-emerald-700 gap-1 text-[11px]">
                <CheckCircle2 className="size-3" />
                Healthy
              </Badge>
            </div>
          </CardHeader>
          <CardContent className="p-5 pt-2">
            <div className="mt-2 flex items-center justify-between border-t border-neutral-100 pt-3 text-xs text-neutral-500">
              <span>1 env · 1 replica</span>
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={onNavigateToDeploy}
                  title="Deploy update"
                  className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
                >
                  <Plus className="size-3.5" />
                </button>
                <button
                  type="button"
                  title="Settings"
                  className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
                >
                  <Settings2 className="size-3.5" />
                </button>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Dynamic Running Services Card */}
        {cluster?.containersRunning && cluster.containersRunning > 1 ? (
          <Card className="group relative overflow-hidden rounded-2xl border border-neutral-200/80 bg-white transition-all hover:border-neutral-300 hover:shadow-md">
            <CardHeader className="p-5 pb-3">
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-3">
                  <span className="flex size-10 items-center justify-center rounded-xl border border-neutral-200 bg-neutral-50 text-neutral-800 group-hover:bg-neutral-100 transition-colors">
                    <Rocket className="size-5" />
                  </span>
                  <div>
                    <CardTitle className="text-base font-semibold text-neutral-900">
                      application-workload
                    </CardTitle>
                    <CardDescription className="text-xs text-neutral-500">
                      {cluster.containersRunning} running containers
                    </CardDescription>
                  </div>
                </div>
                <Badge variant="outline" className="border-emerald-200 bg-emerald-50 text-emerald-700 gap-1 text-[11px]">
                  <CheckCircle2 className="size-3" />
                  Active
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="p-5 pt-2">
              <div className="mt-2 flex items-center justify-between border-t border-neutral-100 pt-3 text-xs text-neutral-500">
                <span>Production · {cluster.containersRunning} tasks</span>
                <div className="flex items-center gap-1">
                  <button
                    type="button"
                    onClick={onNavigateToDeploy}
                    title="Deploy service"
                    className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
                  >
                    <Plus className="size-3.5" />
                  </button>
                </div>
              </div>
            </CardContent>
          </Card>
        ) : null}

        {/* Deploy New Workspace Empty State Card */}
        <button
          type="button"
          onClick={onNavigateToDeploy}
          className="flex min-h-[140px] flex-col items-center justify-center gap-2 rounded-2xl border-2 border-dashed border-neutral-200 bg-neutral-50/50 p-6 text-center transition-all hover:border-neutral-400 hover:bg-neutral-100/50"
        >
          <div className="flex size-9 items-center justify-center rounded-full bg-white shadow-xs border border-neutral-200 text-neutral-600">
            <Plus className="size-4" />
          </div>
          <span className="text-sm font-semibold text-neutral-800">Deploy New Service</span>
          <span className="text-xs text-neutral-500">Roll out image from Docker Hub or GHCR</span>
        </button>
      </div>
    </section>
  )
}
