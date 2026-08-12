import type { ReactNode } from 'react'

interface StateMessageProps {
  variant?: 'loading' | 'error' | 'empty' | 'info' | 'success'
  title: string
  message?: string
  children?: ReactNode
}

export function StateMessage({ variant = 'info', title, message, children }: StateMessageProps) {
  return (
    <div className={`state-message state-message-${variant}`} role={variant === 'error' ? 'alert' : 'status'}>
      {variant === 'loading' && <div className="spinner spinner-sm" aria-hidden="true" />}
      <div>
        <strong>{title}</strong>
        {message && <p>{message}</p>}
      </div>
      {children}
    </div>
  )
}
