// SPDX-License-Identifier: AGPL-3.0-only

import { useCallback, useEffect, useRef, useState } from 'react'
import {
  AlertTriangle,
  Check,
  Clipboard,
  LoaderCircle,
  Network,
  RefreshCw,
  Server,
  ServerCog,
  ShieldCheck,
  Trash2,
} from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Field,
  FieldDescription,
  FieldLegend,
  FieldSet,
  FieldTitle,
} from '@/components/ui/field'
import { Skeleton } from '@/components/ui/skeleton'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  createNodeEnrollment,
  getNodeEnrollments,
  getNodes,
  parseNodeEnrollmentEvent,
  revokeNodeEnrollment,
  type NodeEnrollment,
  type NodeEnrollmentCommand,
  type NodeRole,
  type SwarmNode,
} from '@/lib/api'

const terminalStatuses = new Set([
  'completed',
  'promotion_blocked',
  'revoked',
  'expired',
])

type NodeView = 'current' | 'all'

export function NodesPage({ canManage }: { canManage: boolean }) {
  const [nodes, setNodes] = useState<SwarmNode[]>([])
  const [enrollments, setEnrollments] = useState<NodeEnrollment[]>([])
  const [role, setRole] = useState<NodeRole>('worker')
  const [nodeView, setNodeView] = useState<NodeView>('current')
  const [generated, setGenerated] = useState<NodeEnrollmentCommand | null>(null)
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')
  const streamRef = useRef<EventSource | null>(null)
  const partitionedNodes = partitionSwarmNodes(nodes)
  const visibleNodes = nodeView === 'all' ? nodes : partitionedNodes.current

  const loadPage = useCallback(async (signal?: AbortSignal) => {
    try {
      const [loadedNodes, loadedEnrollments] = await fetchNodePage(canManage, signal)
      setNodes(loadedNodes)
      setEnrollments(loadedEnrollments)
      setError('')
    } catch (loadError) {
      if (!signal?.aborted) {
        setError(errorMessage(loadError, 'Unable to load Swarm nodes.'))
      }
    } finally {
      if (!signal?.aborted) setLoading(false)
    }
  }, [canManage])

  useEffect(() => {
    const controller = new AbortController()
    fetchNodePage(canManage, controller.signal)
      .then(([loadedNodes, loadedEnrollments]) => {
        setNodes(loadedNodes)
        setEnrollments(loadedEnrollments)
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(errorMessage(loadError, 'Unable to load Swarm nodes.'))
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => {
      controller.abort()
      streamRef.current?.close()
    }
  }, [canManage])

  async function handleCreate() {
    setCreating(true)
    setCopied(false)
    setError('')
    streamRef.current?.close()
    try {
      const result = await createNodeEnrollment(role)
      setGenerated(result)
      setEnrollments((current) => [result.enrollment, ...current])
      watchEnrollment(result.enrollment.id)
    } catch (createError) {
      setError(errorMessage(createError, 'Unable to create a node enrollment command.'))
    } finally {
      setCreating(false)
    }
  }

  function watchEnrollment(id: string) {
    const stream = new EventSource(`/api/v1/node-enrollments/${encodeURIComponent(id)}/events`)
    streamRef.current = stream
    stream.addEventListener('enrollment', (event) => {
      try {
        const update = parseNodeEnrollmentEvent(JSON.parse((event as MessageEvent<string>).data))
        setEnrollments((current) => current.map((enrollment) => (
          enrollment.id === update.enrollmentId
            ? {
                ...enrollment,
                status: update.status,
                message: update.message,
                updatedAt: update.createdAt,
              }
            : enrollment
        )))
        setGenerated((current) => current?.enrollment.id === update.enrollmentId
          ? {
              ...current,
              enrollment: {
                ...current.enrollment,
                status: update.status,
                message: update.message,
                updatedAt: update.createdAt,
              },
            }
          : current)
        if (terminalStatuses.has(update.status)) {
          stream.close()
          void loadPage()
        }
      } catch {
        stream.close()
        setError('Nectar received an invalid node enrollment event.')
      }
    })
    stream.onerror = () => {
      if (stream.readyState === EventSource.CLOSED) return
      setError('Live enrollment progress was interrupted. Refresh to see the current state.')
    }
  }

  async function handleCopy() {
    if (!generated) return
    try {
      await navigator.clipboard.writeText(generated.command)
      setCopied(true)
    } catch {
      setError('The browser could not copy the command. Select it manually instead.')
    }
  }

  async function handleRevoke(enrollment: NodeEnrollment) {
    setError('')
    try {
      const revoked = await revokeNodeEnrollment(enrollment.id)
      setEnrollments((current) => current.map((item) => item.id === revoked.id ? revoked : item))
      if (generated?.enrollment.id === revoked.id) {
        streamRef.current?.close()
        setGenerated(null)
      }
    } catch (revokeError) {
      setError(errorMessage(revokeError, 'Unable to revoke the enrollment command.'))
    }
  }

  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-2">
          <p className="text-sm font-medium text-primary">Cluster membership</p>
          <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">Nodes</h1>
          <p className="max-w-2xl text-muted-foreground">
            Inspect the Swarm and enroll a host without entering its SSH password in Nectar.
          </p>
        </div>
        <Button type="button" variant="outline" onClick={() => void loadPage()} disabled={loading}>
          <RefreshCw data-icon="inline-start" />
          Refresh
        </Button>
      </div>

      {error ? (
        <Alert variant="destructive">
          <AlertTriangle aria-hidden="true" />
          <AlertTitle>Node operation failed</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : null}

      {canManage ? (
        <EnrollmentCreator
          role={role}
          creating={creating}
          generated={generated}
          copied={copied}
          onRoleChange={setRole}
          onCreate={() => void handleCreate()}
          onCopy={() => void handleCopy()}
        />
      ) : null}

      <Alert>
        <Network aria-hidden="true" />
        <AlertTitle>Use a trusted private network or VPN</AlertTitle>
        <AlertDescription>
          The target host must reach this Nectar URL and the Manager on 2377/TCP. Swarm nodes also
          need 7946/TCP+UDP and 4789/UDP between one another. Do not expose these ports broadly to
          the public Internet.
        </AlertDescription>
      </Alert>

      <div className="flex flex-col gap-4">
        <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-end">
          <div>
            <h2 className="text-xl font-semibold">Swarm nodes</h2>
            <p className="text-sm text-muted-foreground">Live data from the Docker Engine API.</p>
          </div>
          {nodes.length > 0 ? (
            <ToggleGroup
              type="single"
              variant="outline"
              size="sm"
              value={nodeView}
              onValueChange={(value) => {
                if (value === 'current' || value === 'all') setNodeView(value)
              }}
              aria-label="Filter Swarm nodes"
            >
              <ToggleGroupItem value="current" aria-label="Show current node records">
                Current {partitionedNodes.current.length}
              </ToggleGroupItem>
              <ToggleGroupItem value="all" aria-label="Show all node records">
                All {nodes.length}
              </ToggleGroupItem>
            </ToggleGroup>
          ) : null}
        </div>
        {partitionedNodes.historicalIDs.size > 0 ? (
          <p className="text-sm text-muted-foreground" aria-live="polite">
            {nodeView === 'current'
              ? `${partitionedNodes.historicalIDs.size} historical Down ${nodeRecordLabel(partitionedNodes.historicalIDs.size)} hidden. Select All to inspect stale Node IDs.`
              : `Showing all records, including ${partitionedNodes.historicalIDs.size} historical Down ${nodeRecordLabel(partitionedNodes.historicalIDs.size)}.`}
          </p>
        ) : null}
        {loading ? <NodeSkeleton /> : null}
        {!loading && nodes.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center gap-2 py-10 text-center">
              <Server className="size-8 text-muted-foreground" aria-hidden="true" />
              <p className="font-medium">No Swarm nodes were returned</p>
              <p className="text-sm text-muted-foreground">Verify that Nectar is connected to an active Manager.</p>
            </CardContent>
          </Card>
        ) : null}
        {visibleNodes.length > 0 ? (
          <div className="grid gap-4 lg:grid-cols-2">
            {visibleNodes.map((node) => (
              <NodeCard
                key={node.id}
                node={node}
                historical={partitionedNodes.historicalIDs.has(node.id)}
              />
            ))}
          </div>
        ) : null}
      </div>

      {canManage && enrollments.length > 0 ? (
        <div className="flex flex-col gap-4">
          <div>
            <h2 className="text-xl font-semibold">Recent enrollment operations</h2>
            <p className="text-sm text-muted-foreground">Commands expire after 30 minutes and bind to the first machine that claims them.</p>
          </div>
          <div className="grid gap-3">
            {enrollments.map((enrollment) => (
              <EnrollmentCard
                key={enrollment.id}
                enrollment={enrollment}
                onRevoke={() => void handleRevoke(enrollment)}
              />
            ))}
          </div>
        </div>
      ) : null}
    </section>
  )
}

type EnrollmentCreatorProps = {
  role: NodeRole
  creating: boolean
  generated: NodeEnrollmentCommand | null
  copied: boolean
  onRoleChange: (role: NodeRole) => void
  onCreate: () => void
  onCopy: () => void
}

function EnrollmentCreator(props: EnrollmentCreatorProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ServerCog aria-hidden="true" />
          Add a node
        </CardTitle>
        <CardDescription>
          Generate a short-lived command, then run it as root on the host you want to add.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-6">
        <FieldSet>
          <FieldLegend>Final node role</FieldLegend>
          <div className="grid gap-3 sm:grid-cols-2" data-slot="radio-group">
            <RoleOption
              value="worker"
              checked={props.role === 'worker'}
              title="Worker"
              description="Runs application tasks without participating in Manager quorum."
              onChange={props.onRoleChange}
            />
            <RoleOption
              value="manager"
              checked={props.role === 'manager'}
              title="Manager"
              description="Joins as Worker first, then Nectar verifies and promotes it."
              onChange={props.onRoleChange}
            />
          </div>
        </FieldSet>

        {props.role === 'manager' ? (
          <Alert>
            <ShieldCheck aria-hidden="true" />
            <AlertTitle>Manager promotion is deliberately stricter</AlertTitle>
            <AlertDescription>
              The new host must run the exact cluster Docker target version and become Ready before
              Nectar promotes it. A mismatch leaves it safely joined as a Worker.
            </AlertDescription>
          </Alert>
        ) : null}

        <Button type="button" className="self-start" onClick={props.onCreate} disabled={props.creating}>
          {props.creating ? <LoaderCircle className="animate-spin" aria-hidden="true" /> : <ServerCog aria-hidden="true" />}
          {props.creating ? 'Generating…' : 'Generate enrollment command'}
        </Button>

        {props.generated ? (
          <div className="flex flex-col gap-3 rounded-lg border bg-muted/40 p-4">
            <div className="flex flex-col justify-between gap-2 sm:flex-row sm:items-center">
              <div>
                <p className="font-medium">Run once on the target host</p>
                <p className="text-xs text-muted-foreground">
                  Expires {formatTimestamp(props.generated.enrollment.expiresAt)}. Treat it as a secret.
                </p>
              </div>
              <Button type="button" size="sm" variant="outline" onClick={props.onCopy}>
                {props.copied ? <Check aria-hidden="true" /> : <Clipboard aria-hidden="true" />}
                {props.copied ? 'Copied' : 'Copy command'}
              </Button>
            </div>
            <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded-md bg-slate-950 p-4 text-xs text-slate-100">
              <code>{props.generated.command}</code>
            </pre>
            <div className="flex items-center gap-2 text-sm">
              <StatusBadge status={props.generated.enrollment.status} />
              <span className="text-muted-foreground">{props.generated.enrollment.message}</span>
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}

function RoleOption({
  value,
  checked,
  title,
  description,
  onChange,
}: {
  value: NodeRole
  checked: boolean
  title: string
  description: string
  onChange: (role: NodeRole) => void
}) {
  return (
    <label className={`cursor-pointer rounded-lg border p-4 transition-colors ${checked ? 'border-primary bg-primary/5' : 'hover:bg-muted/50'}`}>
      <Field orientation="horizontal">
        <input
          type="radio"
          name="node-role"
          value={value}
          checked={checked}
          onChange={() => onChange(value)}
          aria-label={title}
          className="mt-1 size-4 accent-primary"
        />
        <div className="flex flex-col gap-1">
          <FieldTitle>{title}</FieldTitle>
          <FieldDescription>{description}</FieldDescription>
        </div>
      </Field>
    </label>
  )
}

function NodeCard({ node, historical }: { node: SwarmNode; historical: boolean }) {
  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <CardTitle>{node.hostname}</CardTitle>
            <CardDescription>{node.address} · {node.operatingSystem} · {node.architecture}</CardDescription>
          </div>
          <div className="flex gap-2">
            {historical ? <Badge variant="secondary">historical</Badge> : null}
            <Badge variant={node.status === 'ready' ? 'default' : 'secondary'}>{node.status}</Badge>
            <Badge variant="outline">{node.role}</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="grid gap-4 sm:grid-cols-2">
        <NodeDetail label="Availability" value={node.availability} />
        <NodeDetail label="Manager status" value={node.managerStatus ?? 'not a manager'} />
        <NodeDetail label="Docker Engine" value={node.dockerVersion} />
        <NodeDetail label="Cluster target" value={node.desiredDockerVersion || 'not recorded'} />
        <NodeDetail className="sm:col-span-2" label="Node ID" value={node.id} mono />
      </CardContent>
      {node.versionDrift ? (
        <CardFooter>
          <Alert>
            <AlertTriangle aria-hidden="true" />
            <AlertTitle>Docker version drift</AlertTitle>
            <AlertDescription>
              Existing Docker was preserved. Review compatibility before assigning Manager duties.
            </AlertDescription>
          </Alert>
        </CardFooter>
      ) : null}
    </Card>
  )
}

function EnrollmentCard({ enrollment, onRevoke }: { enrollment: NodeEnrollment; onRevoke: () => void }) {
  const canRevoke = !terminalStatuses.has(enrollment.status)
  return (
    <Card>
      <CardContent className="flex flex-col justify-between gap-4 p-4 sm:flex-row sm:items-center">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge status={enrollment.status} />
            <Badge variant="outline">{enrollment.requestedRole}</Badge>
            <span className="font-medium">{enrollment.hostname || 'Waiting for a host'}</span>
          </div>
          <p className="mt-2 text-sm text-muted-foreground">{enrollment.message || 'No progress reported.'}</p>
          <p className="mt-1 truncate font-mono text-xs text-muted-foreground" title={enrollment.id}>{enrollment.id}</p>
        </div>
        {canRevoke ? (
          <Button type="button" size="sm" variant="outline" onClick={onRevoke}>
            <Trash2 aria-hidden="true" />
            Revoke
          </Button>
        ) : null}
      </CardContent>
    </Card>
  )
}

function StatusBadge({ status }: { status: string }) {
  const variant = status === 'completed'
    ? 'default'
    : status === 'failed' || status === 'promotion_blocked'
      ? 'destructive'
      : 'secondary'
  return <Badge variant={variant}>{status.replaceAll('_', ' ')}</Badge>
}

function NodeDetail({ label, value, mono, className }: { label: string; value: string; mono?: boolean; className?: string }) {
  return (
    <div className={`flex min-w-0 flex-col gap-1 ${className ?? ''}`}>
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
      <span className={mono ? 'truncate font-mono text-sm' : 'text-sm font-medium'} title={value}>{value}</span>
    </div>
  )
}

function NodeSkeleton() {
  return (
    <div className="grid gap-4 lg:grid-cols-2" aria-label="Loading Swarm nodes">
      {[0, 1].map((key) => (
        <Card key={key}>
          <CardHeader><Skeleton className="h-6 w-40" /><Skeleton className="h-4 w-64" /></CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            {[0, 1, 2, 3].map((detail) => <Skeleton key={detail} className="h-10 w-full" />)}
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function formatTimestamp(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value))
}

function errorMessage(error: unknown, fallback: string) {
  return error instanceof Error ? error.message : fallback
}

function fetchNodePage(canManage: boolean, signal?: AbortSignal) {
  return Promise.all([
    getNodes(signal),
    canManage ? getNodeEnrollments(signal) : Promise.resolve([]),
  ])
}

function partitionSwarmNodes(nodes: SwarmNode[]) {
  const currentIdentities = new Set<string>()
  const historicalIDs = new Set<string>()

  for (const node of nodes) {
    if (node.status.toLowerCase() !== 'down') {
      currentIdentities.add(nodeIdentity(node))
    }
  }

  const current = nodes.filter((node) => {
    const historical = node.status.toLowerCase() === 'down' &&
      currentIdentities.has(nodeIdentity(node))
    if (historical) historicalIDs.add(node.id)
    return !historical
  })

  return { current, historicalIDs }
}

function nodeIdentity(node: SwarmNode) {
  return `${node.hostname.trim().toLowerCase()}\u0000${node.address.trim().toLowerCase()}`
}

function nodeRecordLabel(count: number) {
  return count === 1 ? 'record' : 'records'
}
