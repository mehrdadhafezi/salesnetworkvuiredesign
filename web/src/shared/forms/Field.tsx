import { useId, type InputHTMLAttributes } from 'react'
import { Icon } from '../ui/icons'

/** SN-207 §10: label above, helper text, inline error with icon + text. */
export function Field({ label, help, error, ...input }:
  InputHTMLAttributes<HTMLInputElement> & { label: string; help?: string; error?: string }) {
  const id = useId()
  const helpId = help ? `${id}-help` : undefined
  const errId = error ? `${id}-err` : undefined
  const describedBy = [helpId, errId].filter(Boolean).join(' ') || undefined
  return (
    <div>
      <label className="lbl" htmlFor={id}>
        {label}
        {help ? <span className="help" id={helpId}>{help}</span> : null}
      </label>
      <input {...input} id={id} className="input" aria-invalid={error ? true : undefined} aria-describedby={describedBy} />
      {error ? <p className="field-err" id={errId}><Icon name="alert" />{error}</p> : null}
    </div>
  )
}
