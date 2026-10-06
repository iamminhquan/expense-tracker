import { useRef, useState, type DragEvent } from 'react'
import { LoaderCircle, Upload } from 'lucide-react'
import { FieldErrorText } from '../../components/ui/FieldErrorText'
import { buttonClass } from '../../lib/formStyles'

interface UploadStepProps {
  reading: boolean
  error: string | null
  onFile: (file: File) => void
}

export function UploadStep({ reading, error, onFile }: UploadStepProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)

  function onDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setDragging(false)
    const file = e.dataTransfer.files[0]
    if (file) onFile(file)
  }

  return (
    <div>
      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
        className={`flex flex-col items-center rounded-[24px] border-2 border-dashed px-6 py-8 text-center md:p-12 ${
          dragging ? 'border-accent bg-surface-2' : 'border-border-strong'
        }`}
      >
        <span
          aria-hidden="true"
          className={`mb-4 flex size-14 items-center justify-center rounded-full ${dragging ? 'bg-accent text-on-accent' : 'bg-surface-2 text-ink'}`}
        >
          {reading ? <LoaderCircle className="size-6 animate-spin motion-reduce:animate-none" /> : <Upload className="size-6" />}
        </span>
        <p className="font-display text-[20px] leading-[26px] font-bold text-ink">{reading ? 'Reading your file…' : 'Drop a CSV file here'}</p>
        <p className="mt-1.5 max-w-[420px] text-[14px] leading-[22px] text-ink-muted">
          A file exported from $pend, or one from your bank or another app. If we can't tell how to read it, we'll ask.
        </p>
        <input
          ref={inputRef}
          type="file"
          accept=".csv,text/csv"
          className="sr-only"
          tabIndex={-1}
          aria-hidden="true"
          onChange={(e) => {
            const file = e.target.files?.[0]
            e.target.value = ''
            if (file) onFile(file)
          }}
        />
        <button type="button" disabled={reading} aria-busy={reading} onClick={() => inputRef.current?.click()} className={`${buttonClass('primary')} mt-5`}>
          Choose file
        </button>
      </div>
      {error && (
        <div role="alert">
          <FieldErrorText>{error}</FieldErrorText>
        </div>
      )}
    </div>
  )
}
