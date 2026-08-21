// SPDX-License-Identifier: AGPL-3.0-only

import type { ReactNode } from 'react'

import { Brand } from '@/components/brand'
import type { VersionInfo } from '@/lib/api'

type PageFrameProps = {
  children: ReactNode
  version?: VersionInfo
  actions?: ReactNode
  narrow?: boolean
}

export function PageFrame({ children, version, actions, narrow }: PageFrameProps) {
  return (
    <div className="mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-10 px-5 py-6 sm:px-8 lg:px-10">
      <header className="flex items-center justify-between gap-4">
        <Brand />
        {actions}
      </header>
      <main className={narrow ? 'mx-auto flex w-full max-w-lg flex-1 items-center' : 'flex-1'}>
        {children}
      </main>
      <footer className="flex flex-wrap items-center justify-between gap-2 border-t py-4 text-xs text-muted-foreground">
        <span>
          AGPL-3.0-only ·{' '}
          {version ? <a className="underline underline-offset-4 hover:text-foreground" href={version.sourceUrl} target="_blank" rel="noreferrer">Source code</a> : 'Free and open source'}
        </span>
        {version ? <span>Version {version.version} · {version.commit}</span> : null}
      </footer>
    </div>
  )
}
