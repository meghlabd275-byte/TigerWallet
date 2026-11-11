'use client'

import React from 'react'
import { safeExternalUrl } from '@/lib/safeUrl'

/**
 * Renders an anchor only when the href is a real http(s) URL.
 *
 * dApp and launchpad entries carry URLs authored by third parties, so the
 * value cannot be trusted as an href: a `javascript:` URL would run in the
 * wallet origin on click. When the value is rejected the children render as
 * inert text so the entry is still visible.
 */
export function SafeExternalLink({
  href,
  className,
  children,
  ...rest
}: {
  href: unknown
  className?: string
  children: React.ReactNode
} & Omit<React.AnchorHTMLAttributes<HTMLAnchorElement>, 'href'>) {
  const safe = safeExternalUrl(href)

  if (!safe) {
    return <span className={className}>{children}</span>
  }

  return (
    <a href={safe} className={className} target="_blank" rel="noopener noreferrer nofollow" {...rest}>
      {children}
    </a>
  )
}
