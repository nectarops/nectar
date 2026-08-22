// SPDX-License-Identifier: AGPL-3.0-only

import { useState, type FormEvent } from 'react'
import { KeyRound, LoaderCircle, ShieldCheck } from 'lucide-react'

import { PageFrame } from '@/components/page-frame'
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
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { completeSetup, type User, type VersionInfo } from '@/lib/api'

type SetupPageProps = {
  version: VersionInfo
  onComplete: (user: User) => void
}

export function SetupPage({ version, onComplete }: SetupPageProps) {
  const [initToken, setInitToken] = useState('')
  const [username, setUsername] = useState('admin')
  const [password, setPassword] = useState('')
  const [confirmation, setConfirmation] = useState('')
  const [domain, setDomain] = useState('')
  const [acmeEmail, setAcmeEmail] = useState('')
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')

    if (password !== confirmation) {
      setError('The password confirmation does not match.')
      return
    }
    if (Boolean(domain.trim()) !== Boolean(acmeEmail.trim())) {
      setError("Management domain and Let's Encrypt email must be provided together.")
      return
    }

    setSubmitting(true)
    try {
      const user = await completeSetup({
        initToken,
        username,
        password,
        domain: domain.trim(),
        acmeEmail: acmeEmail.trim(),
      })
      onComplete(user)
    } catch (submitError) {
      setError(getMessage(submitError))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <PageFrame version={version} narrow>
      <Card className="w-full border-primary/15 shadow-xl shadow-primary/5">
        <CardHeader className="gap-3">
          <span className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <ShieldCheck aria-hidden="true" />
          </span>
          <CardTitle className="text-2xl">Secure your control plane</CardTitle>
          <CardDescription>
            Create the owner account, then optionally publish this control plane through
            Traefik with automatic HTTPS.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="flex flex-col gap-6" onSubmit={handleSubmit}>
            {error ? (
              <Alert variant="destructive">
                <KeyRound aria-hidden="true" />
                <AlertTitle>Setup was not completed</AlertTitle>
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            ) : null}
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="init-token">One-time setup token</FieldLabel>
                <Input
                  id="init-token"
                  name="initToken"
                  type="password"
                  autoComplete="off"
                  value={initToken}
                  onChange={(event) => setInitToken(event.target.value)}
                  required
                />
                <FieldDescription>This token is invalidated after setup succeeds.</FieldDescription>
              </Field>
              <Field>
                <FieldLabel htmlFor="username">Owner username</FieldLabel>
                <Input
                  id="username"
                  name="username"
                  autoComplete="username"
                  minLength={3}
                  maxLength={32}
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  required
                />
                <FieldDescription>Use 3–32 letters, numbers, dots, dashes, or underscores.</FieldDescription>
              </Field>
              <Field data-invalid={Boolean(error) || undefined}>
                <FieldLabel htmlFor="password">Password</FieldLabel>
                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="new-password"
                  minLength={5}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  required
                />
                <FieldDescription>Use at least 5 characters.</FieldDescription>
              </Field>
              <Field data-invalid={Boolean(error) || undefined}>
                <FieldLabel htmlFor="confirmation">Confirm password</FieldLabel>
                <Input
                  id="confirmation"
                  name="confirmation"
                  type="password"
                  autoComplete="new-password"
                  minLength={5}
                  value={confirmation}
                  onChange={(event) => setConfirmation(event.target.value)}
                  required
                />
                {password && confirmation && password !== confirmation ? (
                  <FieldError>Passwords do not match.</FieldError>
                ) : null}
              </Field>
              <Field data-invalid={Boolean(error) || undefined}>
                <FieldLabel htmlFor="management-domain">Management domain (optional)</FieldLabel>
                <Input
                  id="management-domain"
                  name="domain"
                  inputMode="url"
                  placeholder="nectar.example.com"
                  value={domain}
                  onChange={(event) => setDomain(event.target.value)}
                />
                <FieldDescription>
                  Point this domain to the server first. Traefik requires inbound ports 80
                  and 443.
                </FieldDescription>
              </Field>
              <Field data-invalid={Boolean(error) || undefined}>
                <FieldLabel htmlFor="acme-email">Let&apos;s Encrypt email</FieldLabel>
                <Input
                  id="acme-email"
                  name="acmeEmail"
                  type="email"
                  autoComplete="email"
                  placeholder="ops@example.com"
                  value={acmeEmail}
                  onChange={(event) => setAcmeEmail(event.target.value)}
                />
                <FieldDescription>
                  Required with a management domain and used only for ACME certificate
                  registration.
                </FieldDescription>
                {domain && !acmeEmail ? (
                  <FieldError>Enter the Let&apos;s Encrypt email for this domain.</FieldError>
                ) : null}
              </Field>
            </FieldGroup>
            <Button type="submit" size="lg" disabled={submitting}>
              {submitting ? <LoaderCircle data-icon="inline-start" className="animate-spin" /> : null}
              {domain.trim() ? 'Create owner and configure HTTPS' : 'Create owner account'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </PageFrame>
  )
}

function getMessage(error: unknown) {
  return error instanceof Error ? error.message : 'An unexpected error occurred.'
}
