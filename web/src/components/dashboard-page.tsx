// SPDX-License-Identifier: AGPL-3.0-only

import { useEffect, useState } from 'react'
import {
  AlertTriangle,
  ArrowRight,
  Box,
  Boxes,
  CircleGauge,
  Cpu,
  Database,
  HardDrive,
  LoaderCircle,
  MemoryStick,
  Network,
  Plus,
  RefreshCw,
  Rocket,
  Server,
  Settings2,
  ShieldAlert,
} from 'lucide-react'

import { CommandPalette } from '@/components/command-palette'
import { DeployForm } from '@/components/deploy-form'
import { KeysView } from '@/components/keys-view'
import { ManagementAccessPage } from '@/components/management-access-page'
import { NetworkView } from '@/components/network-view'
import { NodesPage } from '@/components/nodes-page'
import { NotificationsView } from '@/components/notifications-view'
import { ProjectsView } from '@/components/projects-view'
import { SettingsView } from '@/components/settings-view'
import { Sidebar, type DashboardView } from '@/components/sidebar'
import { StorageView } from '@/components/storage-view'
import { TeamView } from '@/components/team-view'
import { TerminalView } from '@/components/terminal-view'
import { TopHeader } from '@/components/top-header'
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
import { Skeleton } from '@/components/ui/skeleton'
import {
  getCluster,
  logout,
  type ClusterSnapshot,
  type User,
  type VersionInfo,
} from '@/lib/api'

type DashboardPageProps = {
  user: User
  version: VersionInfo
  onLoggedOut: () => void
}

export function DashboardPage({ user, version, onLoggedOut }: DashboardPageProps) {
  const [activeView, setActiveView] = useState<DashboardView>('overview')
  const [isCollapsed, setIsCollapsed] = useState(false)
  const [isSearchOpen, setIsSearchOpen] = useState(false)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [cluster, setCluster] = useState<ClusterSnapshot | null>(null)
  const [error, setError] = useState('')
  const [refreshKey, setRefreshKey] = useState(0)

  useEffect(() => {
    const controller = new AbortController()

    getCluster(controller.signal)
      .then((snapshot) => {
        setCluster(snapshot)
        setError('')
      })
      .catch((loadError: unknown) => {
        if (!controller.signal.aborted) {
          setError(
            loadError instanceof Error
              ? loadError.message
              : 'Unable to inspect Docker Engine.',
          )
        }
      })

    return () => controller.abort()
  }, [refreshKey])

  // Global Keyboard Shortcut for Search (⌘K / Ctrl+K)
  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault()
        setIsSearchOpen((prev) => !prev)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  async function handleLogout() {
    try {
      await logout()
    } finally {
      onLoggedOut()
    }
  }

  function handleNavigate(view: DashboardView) {
    setActiveView(view)
    setMobileMenuOpen(false)
  }

  return (
    <div className="min-h-screen bg-white text-neutral-900 selection:bg-neutral-900 selection:text-white">
      {/* Left Sidebar Navigation (Desktop) */}
      <div className="hidden lg:block">
        <Sidebar
          activeView={activeView}
          onNavigate={handleNavigate}
          isCollapsed={isCollapsed}
          onToggleCollapse={() => setIsCollapsed(!isCollapsed)}
          onOpenSearch={() => setIsSearchOpen(true)}
          user={user}
          version={version}
          onLogout={handleLogout}
        />
      </div>

      {/* Mobile Drawer Overlay */}
      {mobileMenuOpen && (
        <div
          className="fixed inset-0 z-40 bg-neutral-900/40 backdrop-blur-xs lg:hidden"
          onClick={() => setMobileMenuOpen(false)}
        >
          <div
            className="w-64 h-full bg-white shadow-2xl"
            onClick={(e) => e.stopPropagation()}
          >
            <Sidebar
              activeView={activeView}
              onNavigate={handleNavigate}
              isCollapsed={false}
              onToggleCollapse={() => setMobileMenuOpen(false)}
              onOpenSearch={() => {
                setMobileMenuOpen(false)
                setIsSearchOpen(true)
              }}
              user={user}
              version={version}
              onLogout={handleLogout}
            />
          </div>
        </div>
      )}

      {/* Command Palette Search Modal (⌘K) */}
      <CommandPalette
        isOpen={isSearchOpen}
        onClose={() => setIsSearchOpen(false)}
        onNavigate={handleNavigate}
        onRefresh={() => setRefreshKey((k) => k + 1)}
        onLogout={handleLogout}
      />

      {/* Right Content Area */}
      <div
        className={`flex min-h-screen flex-col transition-all duration-200 ${
          isCollapsed ? 'lg:pl-16' : 'lg:pl-60'
        }`}
      >
        {/* Top Sticky Header */}
        <TopHeader
          activeView={activeView}
          onNavigate={handleNavigate}
          cluster={cluster}
          onRefresh={() => setRefreshKey((k) => k + 1)}
          onToggleMobileMenu={() => setMobileMenuOpen(!mobileMenuOpen)}
        />

        {/* Dynamic Page Views */}
        <main className="flex-1 p-5 sm:p-8 lg:p-10 max-w-7xl w-full mx-auto">
          {activeView === 'overview' && (
            <OverviewPage
              cluster={cluster}
              error={error}
              onRefresh={() => setRefreshKey((k) => k + 1)}
              onNavigate={handleNavigate}
            />
          )}

          {activeView === 'projects' && (
            <ProjectsView
              cluster={cluster}
              onNavigateToDeploy={() => handleNavigate('deploy')}
            />
          )}

          {activeView === 'terminal' && (
            <TerminalView
              cluster={cluster}
              onRefresh={() => setRefreshKey((k) => k + 1)}
            />
          )}

          {activeView === 'nodes' && (
            <NodesPage canManage={user.role === 'owner'} />
          )}

          {activeView === 'deploy' && (
            <DeploymentPage
              cluster={cluster}
              error={error}
              onRefresh={() => setRefreshKey((k) => k + 1)}
            />
          )}

          {activeView === 'access' && <ManagementAccessPage />}

          {activeView === 'storage' && (
            <StorageView
              cluster={cluster}
              onRefresh={() => setRefreshKey((k) => k + 1)}
            />
          )}

          {activeView === 'network' && (
            <NetworkView
              cluster={cluster}
              onRefresh={() => setRefreshKey((k) => k + 1)}
            />
          )}

          {activeView === 'team' && <TeamView user={user} />}

          {activeView === 'notifications' && <NotificationsView />}

          {activeView === 'keys' && <KeysView />}

          {activeView === 'settings' && (
            <SettingsView cluster={cluster} version={version} />
          )}
        </main>
      </div>
    </div>
  )
}

function OverviewPage({
  cluster,
  error,
  onRefresh,
  onNavigate,
}: {
  cluster: ClusterSnapshot | null
  error: string
  onRefresh: () => void
  onNavigate: (view: DashboardView) => void
}) {
  const swarmActive = cluster?.swarmState === 'active'

  return (
    <section className="flex flex-col gap-10">
      {/* Header matching Nectar Control Plane */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">
            Control plane overview
          </p>
          <h1 className="text-3xl font-semibold tracking-tight text-neutral-900 sm:text-4xl">
            Your Swarm at a glance
          </h1>
          <p className="max-w-2xl text-sm text-neutral-500">
            Live state reported by the Docker Engine connected to this Nectar instance.
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={onRefresh}
          className="gap-2 rounded-xl border-neutral-200 text-neutral-700 hover:bg-neutral-50"
        >
          <RefreshCw className="size-4" data-icon="inline-start" />
          Refresh
        </Button>
      </div>

      {error ? <ClusterError message={error} /> : null}
      {!cluster && !error ? <DashboardSkeleton /> : null}

      {cluster ? (
        <div className="flex flex-col gap-10">
          {/* Projects Section (tasks/1.png reference) */}
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-neutral-900">Projects</h2>
                <p className="text-xs text-neutral-500">Your deployment workspaces</p>
              </div>
              <button
                type="button"
                onClick={() => onNavigate('projects')}
                className="inline-flex items-center gap-1 text-xs font-medium text-neutral-600 hover:text-neutral-900 transition-colors"
              >
                <span>View all</span>
                <ArrowRight className="size-3.5" />
              </button>
            </div>

            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {/* Project Card: nectar-control */}
              <div className="group flex flex-col justify-between rounded-2xl border border-neutral-200/80 bg-white p-5 transition-all hover:border-neutral-300 hover:shadow-xs">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-start gap-3">
                    <span className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-neutral-200/80 bg-neutral-50 text-neutral-800">
                      <Boxes className="size-5" />
                    </span>
                    <div>
                      <h3 className="text-sm font-semibold text-neutral-900">My first project</h3>
                      <p className="text-xs text-neutral-400 mt-0.5">No description</p>
                    </div>
                  </div>
                </div>

                <div className="mt-6 flex items-center justify-between border-t border-neutral-100 pt-3 text-xs text-neutral-500">
                  <span>1 env · 1 resource</span>
                  <div className="flex items-center gap-1">
                    <button
                      type="button"
                      onClick={() => onNavigate('deploy')}
                      title="Deploy update"
                      aria-label="Deploy update"
                      className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
                    >
                      <Plus className="size-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => onNavigate('settings')}
                      title="Settings"
                      className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
                    >
                      <Settings2 className="size-3.5" />
                    </button>
                  </div>
                </div>
              </div>

              {/* Dynamic Project Card: application-workload if containers exist */}
              {cluster.containersRunning > 1 ? (
                <div className="group flex flex-col justify-between rounded-2xl border border-neutral-200/80 bg-white p-5 transition-all hover:border-neutral-300 hover:shadow-xs">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex items-start gap-3">
                      <span className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-neutral-200/80 bg-neutral-50 text-neutral-800">
                        <Rocket className="size-5" />
                      </span>
                      <div>
                        <h3 className="text-sm font-semibold text-neutral-900">
                          application-workload
                        </h3>
                        <p className="text-xs text-neutral-400 mt-0.5">
                          {cluster.containersRunning} running containers
                        </p>
                      </div>
                    </div>
                  </div>

                  <div className="mt-6 flex items-center justify-between border-t border-neutral-100 pt-3 text-xs text-neutral-500">
                    <span>Production · Swarm replica</span>
                    <div className="flex items-center gap-1">
                      <button
                        type="button"
                        onClick={() => onNavigate('deploy')}
                        title="Deploy update"
                        aria-label="Deploy update"
                        className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
                      >
                        <Plus className="size-3.5" />
                      </button>
                    </div>
                  </div>
                </div>
              ) : null}
            </div>
          </div>

          {/* Servers Section (tasks/1.png reference) */}
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-lg font-semibold text-neutral-900">Servers</h2>
                <p className="text-xs text-neutral-500">
                  Infrastructure available for deployments
                </p>
              </div>
              <button
                type="button"
                onClick={() => onNavigate('nodes')}
                className="inline-flex items-center gap-1 text-xs font-medium text-neutral-600 hover:text-neutral-900 transition-colors"
              >
                <span>View all</span>
                <ArrowRight className="size-3.5" />
              </button>
            </div>

            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {/* Server Card: localhost */}
              <div
                onClick={() => onNavigate('nodes')}
                className="group cursor-pointer flex flex-col justify-between rounded-2xl border border-neutral-200/80 bg-white p-5 transition-all hover:border-neutral-300 hover:shadow-xs"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-start gap-3">
                    <span className="flex size-10 shrink-0 items-center justify-center rounded-xl border border-neutral-200/80 bg-neutral-50 text-neutral-800">
                      <Server className="size-5" />
                    </span>
                    <div>
                      <h3 className="text-sm font-semibold text-neutral-900">
                        {cluster.hostname || 'localhost'}
                      </h3>
                      <p className="text-xs text-neutral-400 mt-0.5 line-clamp-1">
                        This is the server where Nectar is running on. Don't del...
                      </p>
                    </div>
                  </div>
                  {!swarmActive && (
                    <AlertTriangle
                      className="size-4 text-amber-500 shrink-0"
                      aria-label="Swarm inactive"
                    />
                  )}
                </div>

                <div className="mt-6 flex items-center justify-between border-t border-neutral-100 pt-3 text-xs text-neutral-500">
                  <span>
                    {cluster.cpus} cores · {formatBytes(cluster.memoryBytes)}
                  </span>
                  <span className="font-mono text-[11px] text-neutral-400">
                    Docker {cluster.dockerVersion}
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* Cluster Status & Metrics */}
          <ClusterOverview cluster={cluster} />
        </div>
      ) : null}
    </section>
  )
}

function DeploymentPage({
  cluster,
  error,
  onRefresh,
}: {
  cluster: ClusterSnapshot | null
  error: string
  onRefresh: () => void
}) {
  const canDeploy = cluster?.available && cluster.swarmState === 'active'

  return (
    <section className="flex flex-col gap-8">
      <div className="flex flex-col gap-1.5">
        <p className="text-xs font-semibold uppercase tracking-wider text-neutral-500">
          Application delivery
        </p>
        <h1 className="text-3xl font-semibold tracking-tight text-neutral-900 sm:text-4xl">
          Deploy services
        </h1>
        <p className="max-w-2xl text-sm text-neutral-500">
          Create a versioned Swarm service or roll an existing service forward.
        </p>
      </div>

      {error ? <ClusterError message={error} /> : null}
      {!cluster && !error ? <DashboardSkeleton /> : null}
      {canDeploy ? <DeployForm /> : null}
      {cluster && !canDeploy ? (
        <Alert>
          <ShieldAlert aria-hidden="true" />
          <AlertTitle>Swarm must be active before deploying</AlertTitle>
          <AlertDescription className="flex flex-col items-start gap-4">
            <p>
              {cluster.available
                ? 'The connected Docker Engine is not an active Swarm manager. Initialize or join a Swarm, then refresh the cluster status.'
                : cluster.error ??
                  'Connect Nectar to a Docker Engine before deploying services.'}
            </p>
            <Button type="button" variant="outline" size="sm" onClick={onRefresh}>
              <RefreshCw data-icon="inline-start" />
              Refresh cluster status
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}
    </section>
  )
}

function ClusterError({ message }: { message: string }) {
  return (
    <Alert variant="destructive">
      <ShieldAlert aria-hidden="true" />
      <AlertTitle>Docker Engine is unavailable</AlertTitle>
      <AlertDescription>{message}</AlertDescription>
    </Alert>
  )
}

function ClusterOverview({ cluster }: { cluster: ClusterSnapshot }) {
  if (!cluster.available) {
    return (
      <Alert>
        <LoaderCircle aria-hidden="true" />
        <AlertTitle>Docker inspection is disabled</AlertTitle>
        <AlertDescription>
          {cluster.error ?? 'No Docker Engine is connected.'}
        </AlertDescription>
      </Alert>
    )
  }

  const memory = formatBytes(cluster.memoryBytes)
  const swarmActive = cluster.swarmState === 'active'
  const dockerVersionMismatch = Boolean(
    cluster.desiredDockerVersion &&
      cluster.dockerVersion &&
      cluster.desiredDockerVersion !== cluster.dockerVersion,
  )

  return (
    <div className="flex flex-col gap-6">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          icon={Network}
          label="Swarm state"
          value={cluster.swarmState || 'unknown'}
          detail={`${cluster.nodes} node(s) · ${cluster.managers} manager(s)`}
        />
        <MetricCard
          icon={Cpu}
          label="CPU capacity"
          value={`${cluster.cpus} cores`}
          detail={cluster.architecture || 'Architecture unavailable'}
        />
        <MetricCard
          icon={MemoryStick}
          label="Memory capacity"
          value={memory}
          detail="Reported by this Docker host"
        />
        <MetricCard
          icon={Box}
          label="Workload"
          value={`${cluster.containersRunning} running`}
          detail={`${cluster.images} local image(s)`}
        />
      </div>

      {dockerVersionMismatch ? (
        <Alert>
          <ShieldAlert aria-hidden="true" />
          <AlertTitle>Docker version policy mismatch</AlertTitle>
          <AlertDescription>
            This Manager runs Docker {cluster.dockerVersion}, but the cluster target is{' '}
            {cluster.desiredDockerVersion}. Nodes without Docker should install the
            target version; nodes with Docker can join without replacing their existing
            Engine and remain visible as version drift.
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-2">
        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex items-center justify-between gap-3">
              <CardTitle className="flex items-center gap-2">
                <Server aria-hidden="true" />
                Host
              </CardTitle>
              <Badge variant="secondary">{cluster.hostname || 'unknown host'}</Badge>
            </div>
            <CardDescription>Engine and operating-system identity.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <Detail label="Operating system" value={cluster.operatingSystem} />
            <Detail label="Kernel" value={cluster.kernelVersion} />
            <Detail label="Docker Engine" value={cluster.dockerVersion} />
            <Detail
              label="Cluster Docker target"
              value={cluster.desiredDockerVersion}
            />
            <Detail label="Docker API" value={cluster.dockerApiVersion} />
          </CardContent>
        </Card>

        <Card className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex items-center justify-between gap-3">
              <CardTitle className="flex items-center gap-2">
                <Boxes aria-hidden="true" />
                Swarm node
              </CardTitle>
              <Badge variant={swarmActive ? 'default' : 'secondary'}>
                {swarmActive ? 'Active' : 'Not initialized'}
              </Badge>
            </div>
            <CardDescription>Membership reported by the local Engine.</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4 sm:grid-cols-2">
            <Detail label="Role" value={cluster.nodeRole || 'none'} />
            <Detail label="Status" value={cluster.nodeStatus || 'unknown'} />
            <Detail label="Availability" value={cluster.availability || 'unknown'} />
            <Detail
              label="Manager status"
              value={cluster.managerStatus || 'not a manager'}
            />
            {cluster.nodeId ? (
              <Detail
                className="sm:col-span-2"
                label="Node ID"
                value={cluster.nodeId}
                mono
              />
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

type MetricCardProps = {
  icon: typeof CircleGauge
  label: string
  value: string
  detail: string
}

function MetricCard({ icon: Icon, label, value, detail }: MetricCardProps) {
  return (
    <Card className="rounded-2xl border border-neutral-200/80 bg-white">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-3 text-muted-foreground">
          <CardDescription>{label}</CardDescription>
          <Icon aria-hidden="true" className="size-4 text-neutral-500" />
        </div>
        <CardTitle className="text-2xl capitalize text-neutral-900">{value}</CardTitle>
      </CardHeader>
      <CardContent className="text-xs text-muted-foreground">{detail}</CardContent>
    </Card>
  )
}

function Detail({
  label,
  value,
  mono,
  className,
}: {
  label: string
  value: string
  mono?: boolean
  className?: string
}) {
  return (
    <div className={`flex min-w-0 flex-col gap-1 ${className ?? ''}`}>
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </span>
      <span
        className={mono ? 'truncate font-mono text-sm' : 'text-sm font-medium'}
        title={value}
      >
        {value || 'unknown'}
      </span>
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div
      className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4"
      aria-label="Loading cluster state"
    >
      {[HardDrive, Database, Cpu, Boxes].map((Icon, index) => (
        <Card key={index} className="rounded-2xl border border-neutral-200/80 bg-white">
          <CardHeader>
            <div className="flex justify-between">
              <Skeleton className="h-4 w-24" />
              <Icon className="text-muted-foreground" aria-hidden="true" />
            </div>
            <Skeleton className="h-8 w-32" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-3 w-36" />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function formatBytes(bytes: number) {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB']
  const exponent = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  )
  return `${(bytes / 1024 ** exponent).toFixed(exponent > 2 ? 1 : 0)} ${units[exponent]}`
}
