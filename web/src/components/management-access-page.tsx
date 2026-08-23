// SPDX-License-Identifier: AGPL-3.0-only

import { useEffect, useState, type FormEvent } from 'react'
import { ExternalLink, Globe, LoaderCircle, ShieldCheck } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import {
  configureManagementAccess,
  getManagementAccess,
  type ManagementAccess,
} from '@/lib/api'

export function ManagementAccessPage() {
  const [access, setAccess] = useState<ManagementAccess | null>(null)
  const [domain, setDomain] = useState('')
  const [acmeEmail, setAcmeEmail] = useState('')
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    const controller = new AbortController()
    getManagementAccess(controller.signal)
      .then((current) => {
        setAccess(current)
        setDomain(current.domain)
        setAcmeEmail(current.acmeEmail)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) setError(getMessage(loadError))
      })
    return () => controller.abort()
  }, [])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setSaved(false)
    if (!domain.trim() || !acmeEmail.trim()) {
      setError("Management domain and Let's Encrypt email are both required.")
      return
    }

    setSubmitting(true)
    try {
      const configured = await configureManagementAccess({
        domain: domain.trim(),
        acmeEmail: acmeEmail.trim(),
      })
      setAccess(configured)
      setDomain(configured.domain)
      setAcmeEmail(configured.acmeEmail)
      setSaved(true)
    } catch (submitError) {
      setError(getMessage(submitError))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col gap-2">
        <p className="text-sm font-medium text-primary">Management ingress</p>
        <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">Configure HTTPS access</h1>
        <p className="max-w-2xl text-muted-foreground">
          Keep IP:8080 as the recovery path, then install one Traefik task on this Nectar Manager for domain access and automatic HTTPS.
        </p>
      </div>

      {error ? (
        <Alert variant="destructive">
          <Globe aria-hidden="true" />
          <AlertTitle>HTTPS access was not configured</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}
      {saved && access?.domain ? (
        <Alert>
          <ShieldCheck aria-hidden="true" />
          <AlertTitle>HTTPS access is configured</AlertTitle>
          <AlertDescription>
            Open <a className="font-medium underline" href={`https://${access.domain}`} target="_blank" rel="noreferrer">https://{access.domain}</a> after DNS and Let&apos;s Encrypt validation complete.
          </AlertDescription>
        </Alert>
      ) : null}

      <Card>
        <CardHeader>
          <CardTitle>Management domain</CardTitle>
          <CardDescription>
            Point the domain to this Nectar Manager first. Only this node will bind host ports 80 and 443.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="flex flex-col gap-6" onSubmit={handleSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="management-domain">Management domain</FieldLabel>
                <Input id="management-domain" inputMode="url" placeholder="nectar.example.com" value={domain} onChange={(event) => setDomain(event.target.value)} required />
                <FieldDescription>Use a DNS name whose A/AAAA record reaches this Manager.</FieldDescription>
              </Field>
              <Field>
                <FieldLabel htmlFor="acme-email">Let&apos;s Encrypt email</FieldLabel>
                <Input id="acme-email" type="email" autoComplete="email" placeholder="ops@example.com" value={acmeEmail} onChange={(event) => setAcmeEmail(event.target.value)} required />
                <FieldDescription>Used for ACME registration and certificate-expiry notices.</FieldDescription>
              </Field>
            </FieldGroup>
            <Button type="submit" className="self-start" disabled={submitting || access === null}>
              {submitting ? <LoaderCircle data-icon="inline-start" className="animate-spin" /> : <ExternalLink data-icon="inline-start" />}
              {access?.domain ? 'Update HTTPS access' : 'Install Traefik and enable HTTPS'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </section>
  )
}

function getMessage(error: unknown) {
  return error instanceof Error ? error.message : 'An unexpected error occurred.'
}
