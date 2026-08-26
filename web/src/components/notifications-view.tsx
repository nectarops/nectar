// SPDX-License-Identifier: AGPL-3.0-only

import { CheckCircle2 } from 'lucide-react'

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

export function NotificationsView() {
  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col gap-1.5">
        <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Manage</p>
        <h1 className="text-3xl font-semibold tracking-tight text-neutral-900">Notifications & Audit</h1>
        <p className="max-w-2xl text-sm text-neutral-500">
          Cluster health events, node joins, and deployment lifecycle notifications.
        </p>
      </div>

      <Card className="rounded-2xl border border-neutral-200/80 bg-white">
        <CardHeader>
          <CardTitle className="text-base font-semibold">Event Stream</CardTitle>
          <CardDescription>Recent actions recorded by the Nectar control plane.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="flex items-start gap-3 rounded-xl bg-neutral-50 p-3.5 border border-neutral-100">
              <CheckCircle2 className="size-5 text-emerald-500 shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-neutral-900">Control Plane Initialized</p>
                <p className="text-xs text-neutral-500 mt-0.5">
                  Nectar successfully bound to local Docker Swarm manager daemon.
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </section>
  )
}
