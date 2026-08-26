// SPDX-License-Identifier: AGPL-3.0-only

import { Menu, Plus, RefreshCw } from 'lucide-react'

import { Breadcrumbs } from '@/components/breadcrumbs'
import type { DashboardView } from '@/components/sidebar'
import { Button } from '@/components/ui/button'
import type { ClusterSnapshot } from '@/lib/api'

type TopHeaderProps = {
  activeView: DashboardView
  onNavigate: (view: DashboardView) => void
  cluster: ClusterSnapshot | null
  onRefresh: () => void
  onToggleMobileMenu?: () => void
}

const viewLabels: Record<DashboardView, { group: string; label: string }> = {
  overview: { group: 'Root Team', label: 'Dashboard' },
  projects: { group: 'Workspace', label: 'Projects' },
  terminal: { group: 'Workspace', label: 'Terminal' },
  nodes: { group: 'Infrastructure', label: 'Nodes' },
  deploy: { group: 'Infrastructure', label: 'Deploy service' },
  access: { group: 'Infrastructure', label: 'HTTPS access' },
  storage: { group: 'Infrastructure', label: 'S3 Storage' },
  network: { group: 'Infrastructure', label: 'Network' },
  team: { group: 'Manage', label: 'Team' },
  notifications: { group: 'Manage', label: 'Notifications' },
  keys: { group: 'Manage', label: 'Keys & Tokens' },
  settings: { group: 'Manage', label: 'Settings' },
}

export function TopHeader({
  activeView,
  onNavigate,
  cluster,
  onRefresh,
  onToggleMobileMenu,
}: TopHeaderProps) {
  const current = viewLabels[activeView] || { group: 'Root Team', label: 'Dashboard' }
  const swarmActive = cluster?.swarmState === 'active'

  return (
    <header className="sticky top-0 z-20 flex h-14 w-full items-center justify-between border-b border-neutral-200/80 bg-white/90 px-4 sm:px-6 backdrop-blur-md">
      <div className="flex items-center gap-3">
        {onToggleMobileMenu && (
          <button
            type="button"
            onClick={onToggleMobileMenu}
            aria-label="Open mobile menu"
            className="rounded-lg p-1.5 text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900 lg:hidden"
          >
            <Menu className="size-5" />
          </button>
        )}

        {/* Breadcrumb Navigation (tasks/1.png reference) */}
        <Breadcrumbs
          items={[
            {
              label: current.group,
              onClick: () => onNavigate('overview'),
              hasDropdown: true,
            },
            {
              label: current.label,
              hasDropdown: true,
            },
          ]}
        />
      </div>

      {/* Right Header Status & Action Controls */}
      <div className="flex items-center gap-3">
        {/* Swarm Status Indicator */}
        <div className="hidden items-center gap-2 rounded-full border border-neutral-200/80 bg-neutral-50/70 px-3 py-1 text-xs text-neutral-600 sm:flex">
          <span
            className={`inline-block size-2 rounded-full ${
              swarmActive
                ? 'bg-emerald-500 shadow-xs shadow-emerald-500/50 animate-pulse'
                : 'bg-amber-500'
            }`}
          />
          <span className="font-medium">
            {cluster
              ? `${cluster.nodes} node(s) · ${swarmActive ? 'Active Swarm' : 'Inactive'}`
              : 'Checking Docker Engine...'}
          </span>
        </div>

        {/* Refresh Button */}
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onRefresh}
          className="h-8 gap-1.5 rounded-lg border-neutral-200 px-2.5 text-xs text-neutral-700 hover:bg-neutral-100"
        >
          <RefreshCw className="size-3.5" data-icon="inline-start" />
          <span className="hidden sm:inline">Refresh</span>
        </Button>

        {/* Deploy Service Button */}
        <Button
          type="button"
          size="sm"
          onClick={() => onNavigate('deploy')}
          className="h-8 gap-1.5 rounded-lg bg-neutral-900 px-3 text-xs font-medium text-white shadow-xs hover:bg-neutral-800"
        >
          <Plus className="size-3.5" />
          <span>Deploy</span>
        </Button>
      </div>
    </header>
  )
}
