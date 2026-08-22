// SPDX-License-Identifier: AGPL-3.0-only

import { AlertCircle, LoaderCircle } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { PageFrame } from '@/components/page-frame'

export function LoadingPage({ error }: { error?: string }) {
  return (
    <PageFrame narrow>
      {error ? (
        <Alert variant="destructive">
          <AlertCircle aria-hidden="true" />
          <AlertTitle>Nectar could not start</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : (
        <div className="flex w-full flex-col items-center gap-4 text-center text-muted-foreground">
          <LoaderCircle className="animate-spin" aria-hidden="true" />
          <p>Connecting to Nectar…</p>
        </div>
      )}
    </PageFrame>
  )
}
