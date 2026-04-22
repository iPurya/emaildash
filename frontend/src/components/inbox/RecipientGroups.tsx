import type { RecipientSummary } from '../../lib/api'

type Props = {
  recipients: RecipientSummary[]
  activeRecipient?: string
  activeRecipientSummary?: RecipientSummary
  isLoading?: boolean
  error?: Error | null
  onSelect: (recipient?: string) => void
}

export function RecipientGroups({ recipients, activeRecipient, activeRecipientSummary, isLoading, error, onSelect }: Props) {
  return (
    <aside className="surface px-4 py-4 sm:px-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="section-label">Recipients</div>
          <h2 className="mt-2 text-lg font-semibold text-white">Mail groups</h2>
        </div>
        <span className="badge-muted">{recipients.length}</span>
      </div>

      <p className="mt-3 text-sm leading-6 text-slate-400">
        Filter inbox by recipient address. Unread counts stay visible so new mail stands out quickly.
      </p>

      <button
        className={`mt-5 w-full rounded-2xl border px-4 py-3 text-left transition ${!activeRecipient ? 'border-blue-400/40 bg-blue-500/10 text-white' : 'border-white/10 bg-slate-950/50 text-slate-200 hover:bg-white/5'}`}
        onClick={() => onSelect(undefined)}
      >
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm font-semibold">All recipients</span>
          <span className={!activeRecipient ? 'badge-info' : 'badge-muted'}>{recipients.reduce((sum, item) => sum + item.unreadCount, 0)} unread</span>
        </div>
        <div className="mt-2 text-xs text-slate-400">Everything currently stored in inbox.</div>
      </button>

      {activeRecipientSummary && (
        <div className="notice-info mt-4">
          Viewing <span className="font-semibold text-white">{activeRecipientSummary.address}</span> with {activeRecipientSummary.count} message{activeRecipientSummary.count === 1 ? '' : 's'} and {activeRecipientSummary.unreadCount} unread.
        </div>
      )}

      {isLoading ? (
        <div className="mt-4 space-y-3">
          {Array.from({ length: 5 }).map((_, index) => (
            <div key={index} className="skeleton h-20 w-full" />
          ))}
        </div>
      ) : error ? (
        <div className="notice-error mt-4">{error.message}</div>
      ) : recipients.length === 0 ? (
        <div className="empty-panel mt-4 min-h-[260px]">
          <div className="text-lg font-semibold text-white">No recipient groups yet</div>
          <p className="mt-2 max-w-xs text-sm leading-6 text-slate-400">
            Once inbound email arrives, recipient addresses will appear here for quick filtering.
          </p>
        </div>
      ) : (
        <div className="pane-scroll mt-4 max-h-[calc(100vh-24rem)] space-y-3 overflow-auto pr-1">
          {recipients.map((recipient) => {
            const isActive = activeRecipient === recipient.address
            return (
              <button
                key={recipient.address}
                className={`w-full rounded-2xl border px-4 py-4 text-left transition ${isActive ? 'border-blue-400/40 bg-blue-500/10 text-white' : 'border-white/10 bg-slate-950/50 text-slate-200 hover:bg-white/5'}`}
                onClick={() => onSelect(recipient.address)}
              >
                <div className="flex items-start justify-between gap-3">
                  <span className="truncate text-sm font-semibold">{recipient.address}</span>
                  <div className="flex items-center gap-2">
                    {recipient.unreadCount > 0 && <span className="badge-info">{recipient.unreadCount} new</span>}
                    <span className="badge-muted">{recipient.count}</span>
                  </div>
                </div>
                <div className="mt-2 truncate text-xs leading-5 text-slate-400">{recipient.latestSubject ?? 'No messages yet'}</div>
                {recipient.latestReceived && <div className="mt-2 text-xs text-slate-500">Last email {new Date(recipient.latestReceived).toLocaleString()}</div>}
              </button>
            )
          })}
        </div>
      )}
    </aside>
  )
}
