import type { RecipientSummary } from '../../lib/api'

type Props = {
  recipients: RecipientSummary[]
  activeRecipient?: string
  onSelect: (recipient?: string) => void
}

export function RecipientGroups({ recipients, activeRecipient, onSelect }: Props) {
  return (
    <aside className="rounded-2xl border border-slate-800 bg-slate-900/70 p-4">
      <button className={`mb-3 w-full rounded-xl px-3 py-2 text-left ${!activeRecipient ? 'bg-blue-600 text-white' : 'bg-slate-800 text-slate-200'}`} onClick={() => onSelect(undefined)}>
        All recipients
      </button>
      <div className="space-y-2">
        {recipients.map((recipient) => (
          <button
            key={recipient.address}
            className={`w-full rounded-xl px-3 py-2 text-left ${activeRecipient === recipient.address ? 'bg-blue-600 text-white' : 'bg-slate-800 text-slate-200'}`}
            onClick={() => onSelect(recipient.address)}
          >
            <div className="flex items-center justify-between">
              <span className="truncate text-sm font-medium">{recipient.address}</span>
              <span className="text-xs opacity-80">{recipient.count}</span>
            </div>
            <div className="truncate pt-1 text-xs opacity-70">{recipient.latestSubject ?? 'No messages yet'}</div>
          </button>
        ))}
      </div>
    </aside>
  )
}
