// SPDX-License-Identifier: AGPL-3.0-only

import { Boxes } from 'lucide-react'

export function Brand() {
  return (
    <div className="flex items-center gap-3">
      <span className="flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm">
        <Boxes aria-hidden="true" />
      </span>
      <div className="flex flex-col gap-0.5">
        <span className="text-lg font-semibold tracking-tight">Nectar</span>
        <span className="text-xs text-muted-foreground">Docker Swarm, without the ceremony</span>
      </div>
    </div>
  )
}
