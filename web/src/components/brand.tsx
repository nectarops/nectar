// SPDX-License-Identifier: AGPL-3.0-only

import { Boxes } from 'lucide-react'

type BrandProps = {
  collapsed?: boolean
  version?: string
  subtitle?: string
}

export function Brand({ collapsed = false, version, subtitle }: BrandProps) {
  if (collapsed) {
    return (
      <div className="flex items-center justify-center" title="Nectar">
        <span className="flex size-9 items-center justify-center rounded-lg bg-neutral-900 text-white shadow-sm ring-1 ring-black/10">
          <Boxes className="size-4" aria-hidden="true" />
        </span>
      </div>
    )
  }

  return (
    <div className="flex items-center gap-2.5">
      <span className="flex size-8 items-center justify-center rounded-lg bg-neutral-900 text-white shadow-sm ring-1 ring-black/10">
        <Boxes className="size-4" aria-hidden="true" />
      </span>
      <div className="flex flex-col">
        <div className="flex items-baseline gap-1.5">
          <span className="text-base font-semibold tracking-tight text-neutral-900">Nectar</span>
          <span className="font-mono text-xs text-neutral-400">{version ? `v${version}` : 'v0.1.0'}</span>
        </div>
        {subtitle ? <span className="text-[11px] text-muted-foreground leading-none">{subtitle}</span> : null}
      </div>
    </div>
  )
}
