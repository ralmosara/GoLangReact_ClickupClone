import { useDropzone } from 'react-dropzone'
import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { cn } from '../../../lib/utils'
import type { Attachment } from '../../../types'

function humanSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

export function AttachmentsPanel({ taskId }: { taskId: string }) {
  const { data: attachments } = useQuery({
    queryKey: ['attachments', taskId],
    queryFn: () => api.get(`tasks/${taskId}/attachments`).json<Attachment[]>(),
  })

  const upload = useMutation({
    mutationFn: async (file: File) => {
      const form = new FormData()
      form.append('file', file)
      return api.post(`tasks/${taskId}/attachments`, { body: form, timeout: 60_000 }).json<Attachment>()
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['attachments', taskId] }),
  })

  const del = useMutation({
    mutationFn: (id: string) => api.delete(`attachments/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['attachments', taskId] }),
  })

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop: (files) => files.forEach((f) => upload.mutate(f)),
    multiple: true,
  })

  return (
    <div>
      <div
        {...getRootProps()}
        className={cn(
          'border border-dashed rounded-xl px-4 py-5 text-center cursor-pointer transition-colors',
          isDragActive
            ? 'border-brand-400 bg-brand-50'
            : 'border-ink-5/40 hover:border-brand-300 hover:bg-canvas/60',
        )}
      >
        <input {...getInputProps()} />
        <svg className="w-5 h-5 text-ink-4 mx-auto mb-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.9A5 5 0 0115.9 6.1A5.5 5.5 0 0118 17H7z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 10v8m0-8l-3 3m3-3l3 3" />
        </svg>
        <p className="text-xs text-ink-2 font-medium">
          {isDragActive ? 'Drop files here' : 'Drop files or click to upload'}
        </p>
        <p className="text-[10px] text-ink-4 mt-0.5">Max 32 MB per file</p>
      </div>

      {upload.isPending && (
        <p className="text-[10px] text-ink-4 mt-2">Uploading…</p>
      )}

      {(attachments?.length ?? 0) > 0 && (
        <ul className="mt-3 space-y-1.5">
          {attachments!.map((a) => (
            <li
              key={a.id}
              className="flex items-center gap-2 px-3 py-2 bg-canvas/60 border border-ink-5/20 rounded-lg"
            >
              <svg className="w-4 h-4 text-ink-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
              </svg>
              <a
                href={`/api/v1/attachments/${a.id}`}
                target="_blank"
                rel="noreferrer"
                className="flex-1 min-w-0 text-xs text-ink-1 hover:text-brand-600 truncate font-medium"
              >
                {a.filename}
              </a>
              <span className="text-[10px] text-ink-4 shrink-0">{humanSize(a.size_bytes)}</span>
              <a
                href={`/api/v1/attachments/${a.id}?download=1`}
                className="text-ink-4 hover:text-ink-1 p-1 rounded hover:bg-ink-1/5"
                title="Download"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2M7 10l5 5 5-5M12 15V3" />
                </svg>
              </a>
              <button
                onClick={() => del.mutate(a.id)}
                className="text-ink-4 hover:text-red-500 p-1 rounded hover:bg-red-50"
                title="Delete"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
