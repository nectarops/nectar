// SPDX-License-Identifier: AGPL-3.0-only

import { useState, type FormEvent } from 'react'
import { CheckCircle2, LoaderCircle, Rocket, ShieldCheck } from 'lucide-react'

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
import { deployService, type DeploymentResult } from '@/lib/api'

export function DeployForm() {
  const [serviceName, setServiceName] = useState('')
  const [image, setImage] = useState('')
  const [version, setVersion] = useState('')
  const [domain, setDomain] = useState('')
  const [acmeEmail, setAcmeEmail] = useState('')
  const [port, setPort] = useState('8080')
  const [replicas, setReplicas] = useState('1')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<DeploymentResult | null>(null)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setError('')
    setResult(null)

    try {
      setResult(await deployService({
        serviceName,
        image,
        version,
        domain,
        acmeEmail,
        port: Number(port),
        replicas: Number(replicas),
      }))
    } catch (deploymentError) {
      setError(deploymentError instanceof Error ? deploymentError.message : 'Deployment failed.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="flex items-center gap-2"><Rocket aria-hidden="true" />Deploy a service</CardTitle>
          <span className="flex items-center gap-1 text-xs text-muted-foreground"><ShieldCheck aria-hidden="true" />Automatic HTTPS</span>
        </div>
        <CardDescription>
          Dock-Weaver creates or rolls the Swarm service and provisions Traefik with Let's Encrypt on the first deployment.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form className="flex flex-col gap-6" onSubmit={handleSubmit}>
          {error ? (
            <Alert variant="destructive">
              <AlertTitle>Deployment failed</AlertTitle>
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          ) : null}
          {result ? (
            <Alert>
              <CheckCircle2 aria-hidden="true" />
              <AlertTitle>{result.updated ? 'Service update accepted' : 'Service created'}</AlertTitle>
              <AlertDescription>
                {result.image} · service ID {result.serviceId.slice(0, 12)}
                {result.warnings.length ? ` · ${result.warnings.join(' · ')}` : ''}
              </AlertDescription>
            </Alert>
          ) : null}

          <FieldGroup className="grid gap-5 md:grid-cols-2">
            <Field>
              <FieldLabel htmlFor="service-name">Service name</FieldLabel>
              <Input id="service-name" value={serviceName} onChange={(event) => setServiceName(event.target.value)} placeholder="payments-api" minLength={3} maxLength={63} required />
            </Field>
            <Field>
              <FieldLabel htmlFor="domain">Domain</FieldLabel>
              <Input id="domain" value={domain} onChange={(event) => setDomain(event.target.value)} placeholder="api.example.com" inputMode="url" required />
            </Field>
            <Field>
              <FieldLabel htmlFor="image">Image repository</FieldLabel>
              <Input id="image" value={image} onChange={(event) => setImage(event.target.value)} placeholder="ghcr.io/acme/payments" required />
              <FieldDescription>Do not include the tag.</FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="image-version">Image version</FieldLabel>
              <Input id="image-version" value={version} onChange={(event) => setVersion(event.target.value)} placeholder="1.4.2" required />
              <FieldDescription>The implicit latest tag is rejected.</FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="container-port">Container port</FieldLabel>
              <Input id="container-port" type="number" min={1} max={65535} value={port} onChange={(event) => setPort(event.target.value)} required />
            </Field>
            <Field>
              <FieldLabel htmlFor="replicas">Replicas</FieldLabel>
              <Input id="replicas" type="number" min={1} max={1000} value={replicas} onChange={(event) => setReplicas(event.target.value)} required />
            </Field>
            <Field className="md:col-span-2">
              <FieldLabel htmlFor="acme-email">Let's Encrypt email</FieldLabel>
              <Input id="acme-email" type="email" value={acmeEmail} onChange={(event) => setAcmeEmail(event.target.value)} placeholder="ops@example.com" autoComplete="email" required />
              <FieldDescription>
                The domain must already resolve to this Swarm, and inbound ports 80 and 443 must be open. Reuse this email for later deployments.
              </FieldDescription>
            </Field>
          </FieldGroup>
          <Button type="submit" size="lg" disabled={submitting}>
            {submitting ? <LoaderCircle data-icon="inline-start" className="animate-spin" /> : <Rocket data-icon="inline-start" />}
            {submitting ? 'Deploying…' : 'Deploy image'}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
