// SPDX-License-Identifier: AGPL-3.0-only

import { ShieldCheck } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import type { User } from '@/lib/api'

type TeamViewProps = {
  user: User
}

export function TeamView({ user }: TeamViewProps) {
  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col gap-1.5">
        <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Manage</p>
        <h1 className="text-3xl font-semibold tracking-tight text-neutral-900">Team & Access Control</h1>
        <p className="max-w-2xl text-sm text-neutral-500">
          Manage operators, control plane privileges, and authentication credentials.
        </p>
      </div>

      <Card className="rounded-2xl border border-neutral-200/80 bg-white">
        <CardHeader>
          <CardTitle className="text-base font-semibold">Active Cluster Users</CardTitle>
          <CardDescription>Accounts authorized to access and modify Swarm resources.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="divide-y divide-neutral-100">
            <div className="flex items-center justify-between py-3">
              <div className="flex items-center gap-3">
                <div className="flex size-9 items-center justify-center rounded-full bg-neutral-900 text-sm font-semibold text-white">
                  {user.username.charAt(0).toUpperCase()}
                </div>
                <div>
                  <p className="text-sm font-medium text-neutral-900">{user.username}</p>
                  <p className="text-xs text-neutral-500">{user.username}@cluster.local</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="border-neutral-300 capitalize text-xs">
                  <ShieldCheck className="size-3 text-emerald-600 mr-1" />
                  {user.role}
                </Badge>
                <Badge variant="secondary" className="text-xs">
                  Current Session
                </Badge>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </section>
  )
}
