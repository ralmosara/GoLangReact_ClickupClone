import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { Check } from 'lucide-react'

import { api } from '../../lib/api'
import { Button, PageSpinner } from '../../components/ui'
import { cn } from '../../lib/utils'
import type { FormFieldDef } from '../../types'

interface PublicForm {
  id: string
  name: string
  description: string
  fields: FormFieldDef[]
}

export function PublicFormPage() {
  const { formId } = useParams<{ formId: string }>()
  const [values, setValues] = useState<Record<string, unknown>>({})
  const [error, setError] = useState<string | null>(null)
  const [submitted, setSubmitted] = useState(false)

  // Public endpoint; ky in lib/api sends a Bearer token when present, but the
  // backend route is under /api/v1/public/* which doesn't require auth.
  const { data: form, isLoading } = useQuery({
    queryKey: ['public-form', formId],
    queryFn: () => api.get(`public/forms/${formId}`).json<PublicForm>(),
  })

  const submit = useMutation({
    mutationFn: () =>
      api.post(`public/forms/${formId}/submit`, { json: values }).json<unknown>(),
    onSuccess: () => setSubmitted(true),
    onError: async (err) => {
      const msg = err instanceof Error ? err.message : 'Submission failed'
      setError(msg)
    },
  })

  if (isLoading) return <PageSpinner />

  if (!form) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-canvas">
        <div className="bg-surface border border-ink-5/30 rounded-2xl p-8 text-center">
          <p className="text-sm text-ink-2">This form is not available.</p>
        </div>
      </div>
    )
  }

  if (submitted) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-canvas px-4">
        <div className="bg-surface border border-ink-5/30 rounded-2xl p-10 text-center max-w-md shadow-modal">
          <div className="w-14 h-14 rounded-full bg-green-50 border border-green-200 flex items-center justify-center mx-auto mb-4">
            <Check className="w-7 h-7 text-green-600" />
          </div>
          <h1 className="text-lg font-bold text-ink-1 mb-1.5">Thanks!</h1>
          <p className="text-sm text-ink-3">Your submission was received.</p>
        </div>
      </div>
    )
  }

  const canSubmit = form.fields.every((f) => !f.required || !isEmpty(values[f.id]))

  return (
    <div className="min-h-screen bg-canvas py-12 px-4">
      <div className="max-w-xl mx-auto bg-surface border border-ink-5/30 rounded-2xl shadow-modal p-8 animate-slide-up">
        <h1 className="text-2xl font-bold text-ink-1 mb-2">{form.name}</h1>
        {form.description && <p className="text-sm text-ink-3 mb-6 whitespace-pre-wrap">{form.description}</p>}

        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            setError(null)
            submit.mutate()
          }}
        >
          {form.fields.map((f) => (
            <FieldInput
              key={f.id}
              field={f}
              value={values[f.id]}
              onChange={(v) => setValues((prev) => ({ ...prev, [f.id]: v }))}
            />
          ))}

          {error && <p className="text-xs text-red-600 bg-red-50 border border-red-200 rounded px-3 py-2">{error}</p>}

          <Button type="submit" disabled={!canSubmit || submit.isPending} loading={submit.isPending} className="w-full">
            Submit
          </Button>
        </form>
      </div>
    </div>
  )
}

function FieldInput({
  field, value, onChange,
}: {
  field: FormFieldDef
  value: unknown
  onChange: (v: unknown) => void
}) {
  const label = (
    <label className={cn(
      'block text-xs font-semibold text-ink-2 mb-1',
      field.required && "after:content-['*'] after:text-red-500 after:ml-1",
    )}>
      {field.label}
    </label>
  )
  const inputClass = 'w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-3 py-2 text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400'

  switch (field.kind) {
    case 'text':
      return <div>{label}<input className={cn(inputClass, 'h-9')} value={(value as string) ?? ''} onChange={(e) => onChange(e.target.value)} /></div>
    case 'textarea':
      return <div>{label}<textarea className={cn(inputClass, 'h-24 resize-none')} value={(value as string) ?? ''} onChange={(e) => onChange(e.target.value)} /></div>
    case 'number':
      return <div>{label}<input type="number" className={cn(inputClass, 'h-9')} value={(value as number | '') ?? ''} onChange={(e) => onChange(e.target.value === '' ? null : Number(e.target.value))} /></div>
    case 'email':
      return <div>{label}<input type="email" className={cn(inputClass, 'h-9')} value={(value as string) ?? ''} onChange={(e) => onChange(e.target.value)} /></div>
    case 'select':
      return (
        <div>
          {label}
          <select className={cn(inputClass, 'h-9')} value={(value as string) ?? ''} onChange={(e) => onChange(e.target.value)}>
            <option value="">Pick one…</option>
            {(field.options ?? []).map((opt) => <option key={opt} value={opt}>{opt}</option>)}
          </select>
        </div>
      )
    case 'checkbox':
      return (
        <label className="flex items-center gap-2 text-sm text-ink-2 cursor-pointer">
          <input type="checkbox" checked={!!value} onChange={(e) => onChange(e.target.checked)} />
          {field.label}
        </label>
      )
  }
}

function isEmpty(v: unknown): boolean {
  if (v == null) return true
  if (typeof v === 'string') return v.trim() === ''
  if (typeof v === 'boolean') return v === false
  return false
}
