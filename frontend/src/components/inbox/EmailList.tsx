import type { Email } from '../../lib/api'

type Props = {
  emails: Email[]
  activeEmailId?: number
  onSelect: (email: Email) => void
}

export function EmailList({ emails, activeEmailId, onSelect }: Props) {
  return (
    <section className="rounded-2xl border border-slate-800 bg-slate-900/70 p-4">
      <div className="mb-3 text-sm font-semibold text-slate-200">Messages</div>
      <div className="space-y-2">
        {emails.map((email) => (
          <button
            key={email.id}
            className={`w-full rounded-xl border px-3 py-3 text-left ${activeEmailId === email.id ? 'border-blue-500 bg-slate-800' : 'border-slate-800 bg-slate-950/40'}`}
            onClick={() => onSelect(email)}
          >
            <div className="flex items-center justify-between gap-3">
              <span className="truncate text-sm font-medium text-slate-100">{email.subject || '(no subject)'}</span>
              <span className="shrink-0 text-xs text-slate-400">{new Date(email.receivedAt).toLocaleString()}</span>
            </div>
            <div className="truncate pt-1 text-sm text-slate-300">{email.mailFrom}</div>
            <div className="truncate pt-1 text-xs text-slate-400">{email.textBody || email.htmlBody.replace(/<[^>]+>/g, ' ')}</div>
          </button>
        ))}
      </div>
    </section>
  )
}
