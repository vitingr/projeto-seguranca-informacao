'use client'

import { cn } from 'cn'
import { useEffect, useRef, useState } from 'react'

import type { RevealProps } from './types'

type RevealState = 'pending' | 'shown' | 'static'

// Fades its content in the first time it scrolls into view. The server
// renders it visible, so it stays visible without JavaScript, when printed
// and with reduced motion; only content still below the fold waits for the
// observer. It only toggles data attributes (no inline styles).
export const Reveal: React.FC<RevealProps> = ({ children, className }) => {
  const ref = useRef<HTMLDivElement>(null)
  const [state, setState] = useState<RevealState>('static')

  useEffect(() => {
    const element = ref.current
    if (
      !element ||
      window.matchMedia('(prefers-reduced-motion: reduce)').matches ||
      element.getBoundingClientRect().top < window.innerHeight
    ) {
      return
    }
    const observer = new IntersectionObserver(
      entries => {
        if (entries.some(entry => entry.isIntersecting)) {
          setState('shown')
          observer.disconnect()
        }
      },
      { rootMargin: '0px 0px -10% 0px' }
    )
    // Hidden only once the observer is ready to show it again.
    setState('pending')
    observer.observe(element)
    return () => observer.disconnect()
  }, [])

  return (
    <div
      className={cn(
        'data-shown:animate-rise data-pending:translate-y-4 data-pending:opacity-0 print:translate-y-0 print:opacity-100',
        className
      )}
      data-pending={state === 'pending' ? '' : undefined}
      data-shown={state === 'shown' ? '' : undefined}
      ref={ref}
    >
      {children}
    </div>
  )
}
