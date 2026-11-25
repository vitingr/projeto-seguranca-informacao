'use client'

import { MenuIcon, XIcon } from 'lucide-react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { useId, useRef, useState } from 'react'

import { Button } from '@/components/ui/shadcn/button'

import type { MobileMenuProps } from './types'

/**
 * The role links and the role name on screens below `md`, behind a "Menu"
 * button, so the header fits on one line from 360 px. The panel is a plain
 * disclosure: it closes on Escape (focus back on the button), on a link
 * click and on any route change.
 */
export const MobileMenu = ({ links, roleName }: MobileMenuProps) => {
  const panelId = useId()
  const buttonRef = useRef<HTMLButtonElement>(null)
  const pathname = usePathname()
  const [open, setOpen] = useState<boolean>(false)
  const [openedAt, setOpenedAt] = useState<string>(pathname)

  // A route change (Back included) closes the panel.
  if (open && openedAt !== pathname) {
    setOpen(false)
  }

  const toggle = () => {
    setOpenedAt(pathname)
    setOpen(!open)
  }

  const closeOnEscape = (event: React.KeyboardEvent) => {
    if (event.key === 'Escape' && open) {
      setOpen(false)
      buttonRef.current?.focus()
    }
  }

  return (
    <div className="md:hidden" onKeyDown={closeOnEscape}>
      <Button
        aria-controls={panelId}
        aria-expanded={open}
        onClick={toggle}
        ref={buttonRef}
        variant="ghost"
      >
        {open ? <XIcon aria-hidden /> : <MenuIcon aria-hidden />}
        Menu
      </Button>
      <div
        className="border-border bg-background absolute inset-x-0 top-full z-10 border-b"
        hidden={!open}
        id={panelId}
      >
        <div className="mx-auto flex w-full max-w-7xl flex-col gap-1 px-4 py-3 sm:px-6">
          <p className="text-muted-foreground px-2 py-1 text-sm">
            Papel: {roleName}
          </p>
          {links.map(link => (
            <Link
              className="hover:bg-muted focus-visible:ring-ring/50 rounded-sm px-2 py-2 text-sm outline-none focus-visible:ring-3"
              href={link.href}
              key={link.href}
              onClick={() => setOpen(false)}
            >
              {link.label}
            </Link>
          ))}
        </div>
      </div>
    </div>
  )
}
