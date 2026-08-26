// SPDX-License-Identifier: AGPL-3.0-only

import { ChevronsUpDown } from 'lucide-react'

type BreadcrumbItem = {
  label: string
  href?: string
  onClick?: () => void
  hasDropdown?: boolean
}

type BreadcrumbsProps = {
  items: BreadcrumbItem[]
}

export function Breadcrumbs({ items }: BreadcrumbsProps) {
  return (
    <nav aria-label="Breadcrumb" className="flex items-center gap-1.5 text-sm">
      {items.map((item, index) => {
        const isLast = index === items.length - 1
        return (
          <div key={index} className="flex items-center gap-1.5">
            {index > 0 && <span className="text-neutral-300 select-none">/</span>}
            {item.onClick ? (
              <button
                type="button"
                onClick={item.onClick}
                className="inline-flex items-center gap-1 rounded-md px-1.5 py-1 text-sm font-medium text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900 transition-colors cursor-pointer"
              >
                <span>{item.label}</span>
                {item.hasDropdown !== false && (
                  <ChevronsUpDown className="size-3 text-neutral-400" aria-hidden="true" />
                )}
              </button>
            ) : (
              <span
                className={`inline-flex items-center gap-1 px-1.5 py-1 text-sm ${
                  isLast ? 'font-semibold text-neutral-900' : 'font-medium text-neutral-600'
                }`}
              >
                <span>{item.label}</span>
                {item.hasDropdown !== false && (
                  <ChevronsUpDown className="size-3 text-neutral-400" aria-hidden="true" />
                )}
              </span>
            )}
          </div>
        )
      })}
    </nav>
  )
}
