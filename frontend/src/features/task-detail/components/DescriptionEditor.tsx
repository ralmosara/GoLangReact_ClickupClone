import { useEffect } from 'react'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { cn } from '../../../lib/utils'

/**
 * Minimal TipTap rich-text editor used for task descriptions. M5 extends this
 * with slash commands, mentions, and Y.js collab for Docs.
 */
export function DescriptionEditor({
  value,
  onSave,
}: {
  value: string
  onSave: (next: string) => void
}) {
  const editor = useEditor({
    extensions: [StarterKit],
    content: value,
    editorProps: {
      attributes: {
        class: cn(
          'prose prose-sm max-w-none focus:outline-none text-ink-1 min-h-[90px]',
          '[&_p]:my-1.5 [&_ul]:pl-5 [&_ol]:pl-5 [&_code]:bg-ink-1/5 [&_code]:px-1 [&_code]:rounded',
        ),
      },
    },
    onBlur: ({ editor }) => {
      const html = editor.getHTML()
      if (html !== value) onSave(html)
    },
  })

  useEffect(() => {
    if (editor && editor.getHTML() !== value) {
      editor.commands.setContent(value)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value])

  if (!editor) return null

  return (
    <div className="bg-canvas/50 border border-ink-5/30 rounded-xl px-3.5 py-3 focus-within:border-brand-400 focus-within:ring-2 focus-within:ring-brand-400/40 transition-all">
      <div className="flex items-center gap-1 mb-2 border-b border-ink-5/20 pb-2">
        <FmtButton editor={editor} action="toggleBold" active={editor.isActive('bold')}>B</FmtButton>
        <FmtButton editor={editor} action="toggleItalic" active={editor.isActive('italic')} className="italic">I</FmtButton>
        <FmtButton editor={editor} action="toggleStrike" active={editor.isActive('strike')} className="line-through">S</FmtButton>
        <div className="w-px h-4 bg-ink-5/30 mx-1" />
        <FmtButton editor={editor} action="toggleBulletList" active={editor.isActive('bulletList')}>• List</FmtButton>
        <FmtButton editor={editor} action="toggleOrderedList" active={editor.isActive('orderedList')}>1. List</FmtButton>
        <FmtButton editor={editor} action="toggleCodeBlock" active={editor.isActive('codeBlock')}>{'</>'}</FmtButton>
      </div>
      <EditorContent editor={editor} />
    </div>
  )
}

type EditorType = ReturnType<typeof useEditor>

function FmtButton({
  editor,
  action,
  active,
  children,
  className,
}: {
  editor: EditorType
  action:
    | 'toggleBold'
    | 'toggleItalic'
    | 'toggleStrike'
    | 'toggleBulletList'
    | 'toggleOrderedList'
    | 'toggleCodeBlock'
  active?: boolean
  children: React.ReactNode
  className?: string
}) {
  if (!editor) return null
  return (
    <button
      type="button"
      onClick={() => {
        const chain = editor.chain().focus()
        // TS doesn't know the dynamic key is a method; cast through unknown.
        ;((chain as unknown as Record<string, () => typeof chain>)[action])().run()
      }}
      className={cn(
        'h-6 min-w-6 px-1.5 rounded text-[11px] font-semibold transition-colors',
        active ? 'bg-brand-500 text-white' : 'text-ink-3 hover:bg-ink-1/5',
        className,
      )}
    >
      {children}
    </button>
  )
}
