import { useRef, useState, type DragEvent } from 'react'
import { LoaderCircle } from 'lucide-react'
import { Illustration } from '../../components/ui/Illustration'
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
        className={`flex flex-col items-center rounded-panel border-2 border-dashed px-6 py-8 text-center md:p-12 ${
          dragging ? 'border-accent bg-accent-tint' : 'border-border-strong/50'
        }`}
      >
        {reading ? (
          <span aria-hidden="true" className="mb-5 flex h-[104px] items-center">
            <LoaderCircle className="size-10 animate-spin text-expense motion-reduce:animate-none" />
          </span>
        ) : (
          <Illustration name="upload" className="mb-5 h-[104px]" />
        )}
        <p className="heading text-[20px] leading-[26px] text-ink">{reading ? 'Reading your file…' : 'Drop a CSV file here'}</p>
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
        <button type="button" disabled={reading} aria-busy={reading} onClick={() => inputRef.current?.click()} className={`${buttonClass('primary', 'lg')} mt-6`}>
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
