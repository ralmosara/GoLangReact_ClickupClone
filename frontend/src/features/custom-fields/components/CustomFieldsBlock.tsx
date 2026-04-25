import { useMemo } from 'react'
import { Settings } from 'lucide-react'
import { cn } from '../../../lib/utils'
import type { CustomField, CustomValue } from '../../../types'
import { CustomFieldInput } from './CustomFieldInput'
import { useFieldsForList, useUpsertValue, useValuesForTask } from '../hooks/useCustomFields'

/**
 * Renders all custom fields configured for a task's list, with polymorphic
 * inputs wired to the upsert-value endpoint.
 */
export function CustomFieldsBlock({
  taskId,
  listId,
  workspaceId,
  onManage,
}: {
  taskId: string
  listId: string
  workspaceId: string
  onManage: () => void
}) {
  const { data: fields } = useFieldsForList(listId)
  const { data: values } = useValuesForTask(taskId)
  const upsert = useUpsertValue(taskId)

  const valueByField = useMemo(() => {
    const m = new Map<string, CustomValue>()
    ;(values ?? []).forEach((v) => m.set(v.field_id, v))
    return m
  }, [values])

  if (!fields || fields.length === 0) {
    return (
      <button
        onClick={onManage}
        className="text-xs text-ink-4 hover:text-brand-600 underline-offset-2 hover:underline"
      >
        + Add a custom field
      </button>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-[10px] font-semibold text-ink-4 uppercase tracking-wider">Custom fields</p>
        <button
          onClick={onManage}
          className="text-ink-4 hover:text-ink-1 p-1 rounded hover:bg-ink-1/5"
          title="Manage fields"
        >
          <Settings className="w-3.5 h-3.5" />
        </button>
      </div>
      <div className="grid grid-cols-[140px_minmax(0,1fr)] gap-x-3 gap-y-2 items-center">
        {fields.map((f) => (
          <FieldRow
            key={f.id}
            field={f}
            value={parseValue(valueByField.get(f.id))}
            onChange={(v) => upsert.mutate({ field_id: f.id, value: v })}
            workspaceId={workspaceId}
          />
        ))}
      </div>
    </div>
  )
}

function FieldRow({
  field,
  value,
  onChange,
  workspaceId,
}: {
  field: CustomField
  value: unknown
  onChange: (v: unknown) => void
  workspaceId: string
}) {
  return (
    <>
      <div className={cn('text-xs font-medium text-ink-2 flex items-center gap-1', field.required && "after:content-['*'] after:text-red-500 after:ml-0.5")}>
        {field.name}
      </div>
      <div>
        <CustomFieldInput field={field} value={value} onChange={onChange} workspaceId={workspaceId} />
      </div>
    </>
  )
}

function parseValue(v: CustomValue | undefined): unknown {
  if (!v) return null
  // Backend stores JSONB; the api wrapper already JSON-decodes so `v.value` is ready-to-use.
  return v.value
}
