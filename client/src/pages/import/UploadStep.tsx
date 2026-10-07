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
        className={`flex flex-col items-center rounded-card border-2 border-dashed px-6 py-10 text-center md:p-14 ${
          dragging ? 'border-accent-text bg-accent-tint' : 'border-border-strong/60 bg-app'
        }`}
      >
        <span
          aria-hidden="true"
          className={`mb-4 flex size-16 items-center justify-center rounded-[20px] ${dragging ? 'bg-accent text-on-accent' : 'bg-accent-tint text-accent-text'}`}
        >
          {reading ? <LoaderCircle className="size-6 animate-spin motion-reduce:animate-none" /> : <Upload className="size-6" />}
        </span>
        <p className="font-display text-[22px] leading-7 font-semibold tracking-[-0.015em] text-ink">{reading ? 'Reading your file…' : 'Drop a CSV file here'}</p>
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
