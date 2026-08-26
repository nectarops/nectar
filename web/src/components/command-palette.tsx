// SPDX-License-Identifier: AGPL-3.0-only

import { useCallback, useEffect, useRef, useState } from 'react'
import {
  Boxes,
  Globe,
  HardDrive,
  LayoutDashboard,
  LogOut,
  Plus,
  RefreshCw,
  Rocket,
  Search,
  Server,
  Settings,
  Terminal,
  Users,
  X,
} from 'lucide-react'

type DashboardView =
  | 'overview'
  | 'nodes'
  | 'deploy'
  | 'access'
  | 'projects'
  | 'terminal'
  | 'storage'
  | 'team'
  | 'settings'

type CommandItem = {
  id: string
  title: string
  category: string
  icon: typeof LayoutDashboard
  shortcut?: string
  action: () => void
}

type CommandPaletteProps = {
  isOpen: boolean
  onClose: () => void
  onNavigate: (view: DashboardView) => void
  onRefresh: () => void
  onLogout: () => void
}

export function CommandPalette({
  isOpen,
  onClose,
  onNavigate,
  onRefresh,
  onLogout,
}: CommandPaletteProps) {
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const items: CommandItem[] = [
    {
      id: 'nav-overview',
      title: 'Overview Dashboard',
      category: 'Navigation',
      icon: LayoutDashboard,
      shortcut: 'G O',
      action: () => {
        onNavigate('overview')
        onClose()
      },
    },
    {
      id: 'nav-projects',
      title: 'Projects & Services',
      category: 'Navigation',
      icon: Boxes,
      shortcut: 'G P',
      action: () => {
        onNavigate('projects')
        onClose()
      },
    },
    {
      id: 'nav-nodes',
      title: 'Swarm Nodes & Servers',
      category: 'Navigation',
      icon: Server,
      shortcut: 'G N',
      action: () => {
        onNavigate('nodes')
        onClose()
      },
    },
    {
      id: 'nav-deploy',
      title: 'Deploy a Service',
      category: 'Navigation',
      icon: Rocket,
      shortcut: 'G D',
      action: () => {
        onNavigate('deploy')
        onClose()
      },
    },
    {
      id: 'nav-access',
      title: 'HTTPS & Ingress Access',
      category: 'Navigation',
      icon: Globe,
      shortcut: 'G H',
      action: () => {
        onNavigate('access')
        onClose()
      },
    },
    {
      id: 'nav-terminal',
      title: 'Terminal & Swarm Logs',
      category: 'Navigation',
      icon: Terminal,
      shortcut: 'G T',
      action: () => {
        onNavigate('terminal')
        onClose()
      },
    },
    {
      id: 'nav-storage',
      title: 'S3 & Volumes Storage',
      category: 'Navigation',
      icon: HardDrive,
      action: () => {
        onNavigate('storage')
        onClose()
      },
    },
    {
      id: 'nav-team',
      title: 'Team & Permissions',
      category: 'Navigation',
      icon: Users,
      action: () => {
        onNavigate('team')
        onClose()
      },
    },
    {
      id: 'nav-settings',
      title: 'Cluster Settings',
      category: 'Navigation',
      icon: Settings,
      shortcut: 'G S',
      action: () => {
        onNavigate('settings')
        onClose()
      },
    },
    {
      id: 'act-refresh',
      title: 'Refresh Cluster State',
      category: 'Actions',
      icon: RefreshCw,
      action: () => {
        onRefresh()
        onClose()
      },
    },
    {
      id: 'act-new-service',
      title: 'Create New Swarm Service',
      category: 'Actions',
      icon: Plus,
      action: () => {
        onNavigate('deploy')
        onClose()
      },
    },
    {
      id: 'act-logout',
      title: 'Sign Out of Nectar',
      category: 'Account',
      icon: LogOut,
      action: () => {
        onClose()
        onLogout()
      },
    },
  ]

  const filteredItems = items.filter((item) =>
    item.title.toLowerCase().includes(query.toLowerCase()) ||
    item.category.toLowerCase().includes(query.toLowerCase())
  )

  const activeIndex = Math.min(selectedIndex, Math.max(0, filteredItems.length - 1))

  function handleQueryChange(value: string) {
    setQuery(value)
    setSelectedIndex(0)
  }

  const handleClose = useCallback(() => {
    setQuery('')
    setSelectedIndex(0)
    onClose()
  }, [onClose])

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent) {
      if (!isOpen) return

      if (event.key === 'Escape') {
        handleClose()
      } else if (event.key === 'ArrowDown') {
        event.preventDefault()
        setSelectedIndex((prev) => (prev + 1) % Math.max(1, filteredItems.length))
      } else if (event.key === 'ArrowUp') {
        event.preventDefault()
        setSelectedIndex((prev) => (prev - 1 + filteredItems.length) % Math.max(1, filteredItems.length))
      } else if (event.key === 'Enter') {
        event.preventDefault()
        if (filteredItems[activeIndex]) {
          filteredItems[activeIndex].action()
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, filteredItems, activeIndex, handleClose])

  if (!isOpen) return null

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Command search"
      className="fixed inset-0 z-50 flex items-start justify-center bg-neutral-900/40 p-4 pt-[15vh] backdrop-blur-sm animate-in fade-in duration-150"
      onClick={handleClose}
    >
      <div
        className="w-full max-w-xl overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-2xl animate-in zoom-in-95 duration-150"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Search Input Box */}
        <div className="flex items-center border-b border-neutral-150 px-4 py-3.5">
          <Search className="size-4 shrink-0 text-neutral-400" aria-hidden="true" />
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => handleQueryChange(e.target.value)}
            placeholder="Search dashboard, services, commands..."
            autoFocus
            className="w-full border-0 bg-transparent px-3 text-sm text-neutral-900 placeholder:text-neutral-400 focus:outline-none"
          />
          <button
            type="button"
            onClick={handleClose}
            aria-label="Close command search"
            className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
          >
            <X className="size-4" />
          </button>
        </div>

        {/* Results List */}
        <div className="max-h-80 overflow-y-auto p-2">
          {filteredItems.length === 0 ? (
            <div className="py-10 text-center text-sm text-neutral-400">
              No matching commands or pages found.
            </div>
          ) : (
            <div className="space-y-1">
              {filteredItems.map((item, index) => {
                const isSelected = index === activeIndex
                const Icon = item.icon
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={item.action}
                    onMouseEnter={() => setSelectedIndex(index)}
                    className={`flex w-full items-center justify-between rounded-xl px-3 py-2.5 text-left text-sm transition-colors ${
                      isSelected
                        ? 'bg-neutral-900 text-white'
                        : 'text-neutral-700 hover:bg-neutral-100'
                    }`}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <Icon
                        className={`size-4 shrink-0 ${
                          isSelected ? 'text-white' : 'text-neutral-500'
                        }`}
                        aria-hidden="true"
                      />
                      <span className="font-medium truncate">{item.title}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      <span
                        className={`text-xs ${
                          isSelected ? 'text-neutral-400' : 'text-neutral-400'
                        }`}
                      >
                        {item.category}
                      </span>
                      {item.shortcut && (
                        <kbd
                          className={`rounded px-1.5 py-0.5 font-mono text-[10px] ${
                            isSelected
                              ? 'bg-neutral-800 text-neutral-300'
                              : 'bg-neutral-100 text-neutral-500'
                          }`}
                        >
                          {item.shortcut}
                        </kbd>
                      )}
                    </div>
                  </button>
                )
              })}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-neutral-100 bg-neutral-50/70 px-4 py-2 text-xs text-neutral-400">
          <div className="flex items-center gap-3">
            <span>Navigation: <kbd className="rounded bg-white px-1 py-0.5 shadow-sm border text-[10px]">↑</kbd> <kbd className="rounded bg-white px-1 py-0.5 shadow-sm border text-[10px]">↓</kbd></span>
            <span>Select: <kbd className="rounded bg-white px-1 py-0.5 shadow-sm border text-[10px]">↵</kbd></span>
          </div>
          <span>Close: <kbd className="rounded bg-white px-1 py-0.5 shadow-sm border text-[10px]">Esc</kbd></span>
        </div>
      </div>
    </div>
  )
}
