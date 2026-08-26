// SPDX-License-Identifier: AGPL-3.0-only

import { useEffect, useRef, useState } from 'react'
import {
  BookOpen,
  ChevronRight,
  ChevronUp,
  Code2,
  Heart,
  LogOut,
  MessageSquare,
  Sparkles,
  Sun,
  User as UserIcon,
} from 'lucide-react'

import type { User, VersionInfo } from '@/lib/api'

type UserMenuProps = {
  user: User
  version?: VersionInfo
  collapsed?: boolean
  onLogout: () => void
  onNavigateToProfile?: () => void
}

export function UserMenu({
  user,
  version,
  collapsed = false,
  onLogout,
}: UserMenuProps) {
  const [isOpen, setIsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setIsOpen(false)
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside)
      document.addEventListener('keydown', handleKeyDown)
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [isOpen])

  const initial = user.username.charAt(0).toUpperCase() || 'U'
  const userEmail = `${user.username}@cluster.local`

  return (
    <div ref={menuRef} className="relative">
      {/* Trigger Button */}
      {collapsed ? (
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          aria-expanded={isOpen}
          aria-haspopup="menu"
          aria-label={`User menu for ${user.username}`}
          className="group relative flex size-9 items-center justify-center rounded-full bg-neutral-200 text-neutral-800 transition hover:bg-neutral-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-900"
          title={`${user.username} (${user.role})`}
        >
          <span className="text-xs font-semibold">{initial}</span>
          <div
            role="tooltip"
            className="pointer-events-none absolute left-full ml-3 hidden items-center rounded-lg bg-neutral-900 px-2.5 py-1 text-xs font-medium text-white shadow-lg whitespace-nowrap group-hover:flex z-50 animate-in fade-in zoom-in-95 duration-150"
          >
            {user.username} ({user.role})
          </div>
        </button>
      ) : (
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          aria-expanded={isOpen}
          aria-haspopup="menu"
          className="flex w-full items-center justify-between gap-2 rounded-xl bg-neutral-100/80 px-2.5 py-1.5 text-left transition hover:bg-neutral-200/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-neutral-900"
        >
          <div className="flex items-center gap-2 min-w-0">
            <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-neutral-200 text-xs font-semibold text-neutral-800">
              {initial}
            </span>
            <span className="truncate text-sm font-medium text-neutral-800">{user.username}</span>
          </div>
          <ChevronUp
            className={`size-4 shrink-0 text-neutral-500 transition-transform duration-200 ${
              isOpen ? 'rotate-180' : ''
            }`}
            aria-hidden="true"
          />
        </button>
      )}

      {/* Floating Dropdown Menu (tasks/2.png reference) */}
      {isOpen && (
        <div
          role="menu"
          className={`absolute bottom-full z-50 mb-2 w-64 origin-bottom-left rounded-2xl border border-neutral-200/80 bg-white p-1.5 shadow-xl shadow-neutral-900/10 animate-in fade-in zoom-in-95 duration-150 ${
            collapsed ? 'left-2' : 'left-0'
          }`}
        >
          {/* User Header */}
          <div className="border-b border-neutral-100 px-3 py-2.5">
            <p className="text-sm font-semibold text-neutral-900 leading-tight">{user.username}</p>
            <p className="text-xs text-neutral-500 truncate mt-0.5">{userEmail}</p>
            <div className="mt-1.5 flex items-center gap-1.5">
              <span className="inline-block size-1.5 rounded-full bg-emerald-500" />
              <span className="text-[11px] font-medium text-neutral-600 capitalize">
                Role: {user.role}
              </span>
            </div>
          </div>

          {/* Navigation & Action Items */}
          <div className="py-1">
            <button
              type="button"
              role="menuitem"
              onClick={() => setIsOpen(false)}
              className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
            >
              <UserIcon className="size-4 text-neutral-500" aria-hidden="true" />
              <span>Profile</span>
            </button>

            <button
              type="button"
              role="menuitem"
              onClick={() => setIsOpen(false)}
              className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
            >
              <div className="flex items-center gap-2.5">
                <Sun className="size-4 text-neutral-500" aria-hidden="true" />
                <span>Appearance</span>
              </div>
              <ChevronRight className="size-3.5 text-neutral-400" aria-hidden="true" />
            </button>

            <button
              type="button"
              role="menuitem"
              onClick={() => setIsOpen(false)}
              className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
            >
              <div className="flex items-center gap-2.5">
                <Sparkles className="size-4 text-neutral-500" aria-hidden="true" />
                <span>What's New</span>
              </div>
              <span className="rounded-full bg-rose-500 px-1.5 py-0.5 text-[10px] font-bold text-white leading-none">
                {version ? `v${version.version}` : 'v0.1.0'}
              </span>
            </button>

            <a
              href="https://github.com/nectarops/nectar#readme"
              target="_blank"
              rel="noreferrer"
              role="menuitem"
              onClick={() => setIsOpen(false)}
              className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
            >
              <BookOpen className="size-4 text-neutral-500" aria-hidden="true" />
              <span>Documentation</span>
            </a>

            {version?.sourceUrl ? (
              <a
                href={version.sourceUrl}
                target="_blank"
                rel="noreferrer"
                role="menuitem"
                onClick={() => setIsOpen(false)}
                className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
              >
                <Code2 className="size-4 text-neutral-500" aria-hidden="true" />
                <span>Source Code (AGPL-3.0)</span>
              </a>
            ) : null}

            <a
              href="https://github.com/nectarops/nectar/issues"
              target="_blank"
              rel="noreferrer"
              role="menuitem"
              onClick={() => setIsOpen(false)}
              className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
            >
              <MessageSquare className="size-4 text-neutral-500" aria-hidden="true" />
              <span>Feedback</span>
            </a>

            <a
              href="https://github.com/nectarops/nectar"
              target="_blank"
              rel="noreferrer"
              role="menuitem"
              onClick={() => setIsOpen(false)}
              className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-100 hover:text-neutral-900 transition-colors"
            >
              <Heart className="size-4 text-rose-500" aria-hidden="true" />
              <span>Sponsor us</span>
            </a>
          </div>

          <div className="border-t border-neutral-100 pt-1">
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setIsOpen(false)
                onLogout()
              }}
              className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium text-rose-600 hover:bg-rose-50 transition-colors"
            >
              <LogOut className="size-4" aria-hidden="true" />
              <span>Log out</span>
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
