// SPDX-License-Identifier: AGPL-3.0-only

import { useState } from 'react'
import { Check, Clipboard, Eye, EyeOff, Key, Lock, Plus, ShieldCheck } from 'lucide-react'

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

export function KeysView() {
  const [showToken, setShowToken] = useState(false)
  const [copied, setCopied] = useState(false)

  const maskedToken = 'swmtkn-1-5abc************************************-4def'
  const demoToken = 'swmtkn-1-5abc9842894723948723984723984723984723-4def'

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(demoToken)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // ignore
    }
  }

  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">Manage</p>
          <h1 className="text-3xl font-semibold tracking-tight text-neutral-900">Keys & Tokens</h1>
          <p className="max-w-2xl text-sm text-neutral-500">
            SSH private keys, Docker Swarm join tokens, and API authorization secrets.
          </p>
        </div>
        <Button
          type="button"
          className="gap-2 rounded-xl bg-neutral-900 text-white hover:bg-neutral-800"
        >
          <Plus className="size-4" />
          <span>Add SSH Key</span>
        </Button>
      </div>

      <Alert>
        <ShieldCheck className="size-4 text-emerald-600" aria-hidden="true" />
        <AlertTitle>Encryption at Rest</AlertTitle>
        <AlertDescription>
          Private keys and credentials are encrypted using AES-GCM with instance keys configured outside the database. Tokens are never logged.
        </AlertDescription>
      </Alert>

      <div className="grid gap-6 sm:grid-cols-2">
        {/* SSH Keypairs */}
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <Key className="size-4 text-neutral-700" />
                Cluster Deployment SSH Key
              </CardTitle>
              <Badge variant="outline" className="border-emerald-200 bg-emerald-50 text-emerald-700 text-[11px]">
                Active
              </Badge>
            </div>
            <CardDescription>Default key used for remote worker node bootstrap.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Key Type</span>
              <span className="font-mono text-xs text-neutral-800">ED25519</span>
            </div>
            <div className="flex justify-between border-b border-neutral-100 pb-2">
              <span className="text-neutral-500">Fingerprint</span>
              <span className="font-mono text-xs text-neutral-800">SHA256:7f...e92a</span>
            </div>
            <div className="flex justify-between">
              <span className="text-neutral-500">Strict Host Checking</span>
              <span className="font-medium text-emerald-700">Enforced (known_hosts)</span>
            </div>
          </CardContent>
        </Card>

        {/* Swarm Join Token */}
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <Lock className="size-4 text-neutral-700" />
                Active Worker Join Token
              </CardTitle>
              <Badge variant="secondary" className="text-[11px]">
                On-demand
              </Badge>
            </div>
            <CardDescription>Retrieved dynamically from the Swarm Manager engine.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center justify-between gap-2 rounded-xl bg-neutral-50 p-2.5 border border-neutral-200/80">
              <span className="font-mono text-xs text-neutral-700 truncate">
                {showToken ? demoToken : maskedToken}
              </span>
              <div className="flex items-center gap-1 shrink-0">
                <button
                  type="button"
                  onClick={() => setShowToken(!showToken)}
                  aria-label={showToken ? 'Hide token' : 'Show token'}
                  className="rounded-lg p-1.5 text-neutral-500 hover:bg-neutral-200/70 transition-colors"
                >
                  {showToken ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                </button>
                <button
                  type="button"
                  onClick={handleCopy}
                  aria-label="Copy token"
                  className="rounded-lg p-1.5 text-neutral-500 hover:bg-neutral-200/70 transition-colors"
                >
                  {copied ? <Check className="size-3.5 text-emerald-600" /> : <Clipboard className="size-3.5" />}
                </button>
              </div>
            </div>
            <p className="text-xs text-neutral-500">
              Generated node enrollment tokens are time-bounded and expire automatically.
            </p>
          </CardContent>
        </Card>
      </div>
    </section>
  )
}
