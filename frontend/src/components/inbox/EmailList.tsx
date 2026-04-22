import type { Email } from '../../lib/api'

type Props = {
  emails: Email[]
  activeEmailId?: number
  isLoading?: boolean
  error?: Error | null
  recipient?: string
  onSelect: (email: Email) => void
}

function previewText(email: Email) {
  const text = email.textBody?.trim()
  if (text) {
    return text
  }
  return email.htmlBody.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim()
}

export function EmailList({ emails, activeEmailId, isLoading, error, recipient, onSelect }: Props) {
  return (
    <section className="surface px-4 py-4 sm:px-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="section-label">Messages</div>
          <h2 className="mt-2 text-lg font-semibold text-white">{recipient ? 'Filtered inbox' : 'Recent inbox'}</h2>
        </div>
        <span className="badge-muted">{emails.length}</span>
      </div>

      <p className="mt-3 text-sm leading-6 text-slate-400">
        {recipient ? `Showing messages sent to ${recipient}.` : 'Showing newest received messages across all recipients.'}
      </p>

      {isLoading ? (
        <div className="mt-4 space-y-3">
          {Array.from({ length: 6 }).map((_, index) => (
            <div key={index} className="skeleton h-28 w-full" />
          ))}
        </div>
      ) : error ? (
        <div className="notice-error mt-4">{error.message}</div>
      ) : emails.length === 0 ? (
        <div className="empty-panel mt-4 min-h-[320px]">
          <div className="text-lg font-semibold text-white">No messages to show</div>
          <p className="mt-2 max-w-xs text-sm leading-6 text-slate-400">
            {recipient ? 'Selected recipient has no messages yet.' : 'Inbox is empty. Send test email to provisioned domain to populate it.'}
          </p>
        </div>
      ) : (
        <div className="pane-scroll mt-4 max-h-[calc(100vh-24rem)] space-y-3 overflow-auto pr-1">
          {emails.map((email) => {
            const isActive = activeEmailId === email.id
            const isUnread = !email.readAt
            return (
              <button
                key={email.id}
                className={`w-full rounded-2xl border px-4 py-4 text-left transition ${isActive ? 'border-blue-400/40 bg-blue-500/10' : 'border-white/10 bg-slate-950/50 hover:bg-white/5'}`}
                onClick={() => onSelect(email)}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="truncate text-sm font-semibold text-white">{email.subject || '(no subject)'}</span>
                      {isUnread && <span className="badge-info">Unread</span>}
                      {(email.attachments ?? []).length > 0 && <span className="badge-muted">{email.attachments.length} attachment{email.attachments.length === 1 ? '' : 's'}</span>}
                    </div>
                    <div className="mt-2 truncate text-sm text-slate-300">{email.mailFrom}</div>
                  </div>
                  <span className="shrink-0 text-xs text-slate-400">{new Date(email.receivedAt).toLocaleString()}</span>
                </div>
                <div className="mt-3 line-clamp-2 text-sm leading-6 text-slate-400">{previewText(email) || 'No preview available.'}</div>
              </button>
            )
          })}
        </div>
      )}
    </section>
  )
}
