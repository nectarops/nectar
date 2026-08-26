// SPDX-License-Identifier: AGPL-3.0-only

import {
  Bell,
  Boxes,
  Globe,
  HardDrive,
  Key,
  LayoutDashboard,
  Network,
  PanelLeftClose,
  PanelLeftOpen,
  Rocket,
  Search,
  Server,
  Settings,
  Terminal,
  Users,
} from 'lucide-react'

import { Brand } from '@/components/brand'
import { UserMenu } from '@/components/user-menu'
import type { User, VersionInfo } from '@/lib/api'

export type DashboardView =
  | 'overview'
  | 'nodes'
  | 'deploy'
  | 'access'
  | 'projects'
  | 'terminal'
  | 'storage'
  | 'network'
  | 'team'
  | 'notifications'
  | 'keys'
  | 'settings'

type SidebarNavGroup = {
  title: string
  items: Array<{
    id: DashboardView
    label: string
    icon: typeof LayoutDashboard
    badge?: string
  }>
}

type SidebarProps = {
  activeView: DashboardView
  onNavigate: (view: DashboardView) => void
  isCollapsed: boolean
  onToggleCollapse: () => void
  onOpenSearch: () => void
  user: User
  version?: VersionInfo
  onLogout: () => void
}

const navGroups: SidebarNavGroup[] = [
  {
    title: 'Workspace',
    items: [
      { id: 'overview', label: 'Overview', icon: LayoutDashboard },
      { id: 'projects', label: 'Projects', icon: Boxes },
      { id: 'terminal', label: 'Terminal', icon: Terminal },
    ],
  },
  {
    title: 'Infrastructure',
    items: [
      { id: 'nodes', label: 'Nodes', icon: Server },
      { id: 'deploy', label: 'Deploy service', icon: Rocket },
      { id: 'access', label: 'HTTPS access', icon: Globe },
      { id: 'storage', label: 'S3 Storage', icon: HardDrive },
      { id: 'network', label: 'Network', icon: Network },
    ],
  },
  {
    title: 'Manage',
    items: [
      { id: 'team', label: 'Team', icon: Users },
      { id: 'notifications', label: 'Notifications', icon: Bell },
      { id: 'keys', label: 'Keys & Tokens', icon: Key },
      { id: 'settings', label: 'Settings', icon: Settings },
    ],
  },
]

export function Sidebar({
  activeView,
  onNavigate,
  isCollapsed,
  onToggleCollapse,
  onOpenSearch,
  user,
  version,
  onLogout,
}: SidebarProps) {
  return (
    <aside
      className={`fixed inset-y-0 left-0 z-30 flex flex-col border-r border-neutral-200/80 bg-white transition-all duration-200 select-none ${
        isCollapsed ? 'w-16' : 'w-60'
      }`}
    >
      {/* Brand Header */}
      <div
        className={`flex h-14 items-center border-b border-neutral-100 ${
          isCollapsed ? 'justify-center px-2' : 'px-4'
        }`}
      >
        <Brand collapsed={isCollapsed} version={version?.version} />
      </div>

      {/* Quick Search Trigger (tasks/1.png, 3.png & 4.png) */}
      <div className={`pt-3 pb-2 ${isCollapsed ? 'px-2 flex justify-center' : 'px-3'}`}>
        {isCollapsed ? (
          <button
            type="button"
            onClick={onOpenSearch}
            aria-label="Search"
            title="Search (⌘K)"
            className="group relative flex size-10 items-center justify-center rounded-xl bg-neutral-100/80 text-neutral-600 hover:bg-neutral-200/70 hover:text-neutral-900 transition-colors mx-auto"
          >
            <Search className="size-4" aria-hidden="true" />
            <div
              role="tooltip"
              className="pointer-events-none absolute left-full ml-3 hidden items-center rounded-lg bg-neutral-900 px-2.5 py-1 text-xs font-medium text-white shadow-lg whitespace-nowrap group-hover:flex z-50 animate-in fade-in zoom-in-95 duration-150"
            >
              Search (⌘K)
            </div>
          </button>
        ) : (
          <button
            type="button"
            onClick={onOpenSearch}
            aria-label="Search"
            className="flex w-full items-center justify-between gap-2 rounded-xl border border-neutral-200/80 bg-neutral-50/50 px-3 py-1.5 text-xs text-neutral-500 hover:bg-neutral-100/70 hover:text-neutral-800 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Search className="size-3.5 text-neutral-400" aria-hidden="true" />
              <span>Search</span>
            </div>
            <kbd className="rounded bg-neutral-200/70 px-1.5 py-0.5 font-mono text-[10px] text-neutral-600">
              ⌘K
            </kbd>
          </button>
        )}
      </div>

      {/* Navigation Groups */}
      <div
        className={`flex-1 px-2 py-2 space-y-4 ${
          isCollapsed ? 'overflow-visible' : 'overflow-y-auto'
        }`}
      >
        {navGroups.map((group) => (
          <div key={group.title} className="space-y-1">
            {!isCollapsed && (
              <p className="px-2 pb-1 text-[11px] font-medium tracking-wide text-neutral-400 uppercase">
                {group.title}
              </p>
            )}
            <div className={`space-y-0.5 ${isCollapsed ? 'flex flex-col items-center gap-1' : ''}`}>
              {group.items.map(({ id, label, icon: Icon, badge }) => {
                const isActive = activeView === id
                return (
                  <button
                    key={id}
                    type="button"
                    onClick={() => onNavigate(id)}
                    aria-current={isActive ? 'page' : undefined}
                    title={isCollapsed ? label : undefined}
                    className={`group relative flex items-center transition-colors ${
                      isCollapsed
                        ? 'size-10 justify-center rounded-xl mx-auto'
                        : 'w-full justify-between rounded-xl px-3 py-1.5 text-sm'
                    } ${
                      isActive
                        ? 'bg-neutral-100 font-semibold text-neutral-900 shadow-xs'
                        : 'text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900'
                    }`}
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <Icon
                        className={`size-4 shrink-0 transition-colors ${
                          isActive
                            ? 'text-neutral-900'
                            : 'text-neutral-500 group-hover:text-neutral-800'
                        }`}
                        aria-hidden="true"
                      />
                      {!isCollapsed && <span className="truncate">{label}</span>}
                    </div>
                    {!isCollapsed && badge && (
                      <span className="rounded-full bg-neutral-200 px-1.5 py-0.2 text-[10px] font-medium text-neutral-700">
                        {badge}
                      </span>
                    )}

                    {/* Floating Tooltip in Collapsed Mode (tasks/4.png) */}
                    {isCollapsed && (
                      <div
                        role="tooltip"
                        className="pointer-events-none absolute left-full ml-3 hidden items-center rounded-lg bg-neutral-900 px-2.5 py-1 text-xs font-medium text-white shadow-lg whitespace-nowrap group-hover:flex z-50 animate-in fade-in zoom-in-95 duration-150"
                      >
                        {label}
                      </div>
                    )}
                  </button>
                )
              })}
            </div>
          </div>
        ))}
      </div>

      {/* Bottom User Area & Collapse Toggle (tasks/1.png, 2.png, 3.png, 4.png) */}
      <div className="border-t border-neutral-150 p-2">
        {isCollapsed ? (
          <div className="flex flex-col items-center gap-2">
            {/* Collapse / Expand Toggle Button */}
            <button
              type="button"
              onClick={onToggleCollapse}
              aria-label="Expand sidebar"
              title="Expand sidebar"
              className="group relative flex size-10 items-center justify-center rounded-xl text-neutral-400 hover:bg-neutral-100 hover:text-neutral-800 transition-colors mx-auto"
            >
              <PanelLeftOpen className="size-4" aria-hidden="true" />
              <div
                role="tooltip"
                className="pointer-events-none absolute left-full ml-3 hidden items-center rounded-lg bg-neutral-900 px-2.5 py-1 text-xs font-medium text-white shadow-lg whitespace-nowrap group-hover:flex z-50 animate-in fade-in zoom-in-95 duration-150"
              >
                Expand sidebar
              </div>
            </button>

            {/* Circular User Avatar Menu (tasks/4.png) */}
            <div className="w-full flex justify-center">
              <UserMenu
                user={user}
                version={version}
                collapsed={true}
                onLogout={onLogout}
              />
            </div>
          </div>
        ) : (
          <div className="flex items-center justify-between gap-1">
            <div className="flex-1 min-w-0">
              <UserMenu
                user={user}
                version={version}
                collapsed={false}
                onLogout={onLogout}
              />
            </div>

            <button
              type="button"
              onClick={onToggleCollapse}
              aria-label="Collapse sidebar"
              title="Collapse sidebar"
              className="flex size-8 shrink-0 items-center justify-center rounded-lg text-neutral-400 hover:bg-neutral-100 hover:text-neutral-800 transition-colors"
            >
              <PanelLeftClose className="size-4" aria-hidden="true" />
            </button>
          </div>
        )}
      </div>
    </aside>
  )
}
