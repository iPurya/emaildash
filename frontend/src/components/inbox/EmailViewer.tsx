import DOMPurify from 'dompurify'
import type { Email } from '../../lib/api'

type Props = {
  email?: Email
  isLoading?: boolean
  recipient?: string
}

function formatBytes(size: number) {
  if (size < 1024) {
    return `${size} B`
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`
}

export function EmailViewer({ email, isLoading, recipient }: Props) {
  if (isLoading) {
    return (
      <section className="surface px-5 py-5 sm:px-6">
        <div className="section-label">Message</div>
        <div className="mt-4 space-y-4">
          <div className="skeleton h-8 w-2/3" />
          <div className="skeleton h-20 w-full" />
          <div className="skeleton h-80 w-full" />
        </div>
      </section>
    )
  }

  if (!email) {
    return (
      <section className="surface px-5 py-5 sm:px-6">
        <div className="section-label">Message</div>
        <div className="empty-panel mt-4 min-h-[520px]">
          <div className="text-lg font-semibold text-white">Select message</div>
          <p className="mt-2 max-w-md text-sm leading-6 text-slate-400">
            {recipient ? `Choose message from ${recipient} to inspect full content and attachments.` : 'Choose message from list to inspect full content and attachments.'}
          </p>
        </div>
      </section>
    )
  }

  return (
    <section className="surface px-5 py-5 sm:px-6">
      <div className="flex flex-col gap-3 border-b border-white/10 pb-5">
        <div className="flex flex-wrap items-center gap-2">
          <span className="badge-info">{email.readAt ? 'Read' : 'Unread'}</span>
          <span className="badge-muted">{new Date(email.receivedAt).toLocaleString()}</span>
          <span className="badge-muted">{formatBytes(email.rawSize)}</span>
        </div>

        <div>
          <h2 className="text-2xl font-semibold tracking-tight text-white">{email.subject || '(no subject)'}</h2>
          <p className="mt-2 text-sm leading-6 text-slate-400">
            Review sender, recipients, body, and attachment metadata in one pane.
          </p>
        </div>

        <div className="grid gap-3 lg:grid-cols-3">
          <div className="surface-muted px-4 py-3">
            <div className="section-label">From</div>
            <div className="mt-2 break-words text-sm text-slate-100">{email.mailFrom}</div>
          </div>
          <div className="surface-muted px-4 py-3 lg:col-span-2">
            <div className="section-label">To</div>
            <div className="mt-2 break-words text-sm text-slate-100">{email.recipients.join(', ')}</div>
          </div>
        </div>
      </div>

      <div className="mt-6 rounded-3xl border border-white/10 bg-slate-950/55 p-5 sm:p-6">
        {email.htmlBody ? (
          <div className="html-email" dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(email.htmlBody) }} />
        ) : (
          <pre className="m-0 whitespace-pre-wrap break-words text-sm leading-7 text-slate-200">{email.textBody || 'No body content.'}</pre>
        )}
      </div>

      {(email.attachments ?? []).length > 0 && (
        <div className="mt-6 border-t border-white/10 pt-6">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="section-label">Attachments</div>
              <div className="mt-2 text-sm text-slate-400">Stored attachment metadata for this message.</div>
            </div>
            <span className="badge-muted">{email.attachments.length}</span>
          </div>
          <ul className="mt-4 grid gap-3 lg:grid-cols-2">
            {(email.attachments ?? []).map((attachment) => (
              <li key={attachment.id} className="surface-muted px-4 py-4">
                <div className="text-sm font-semibold text-white">{attachment.filename}</div>
                <div className="mt-2 text-xs leading-5 text-slate-400">{attachment.contentType || 'Unknown type'}</div>
                <div className="mt-3 flex flex-wrap gap-2 text-xs text-slate-300">
                  <span className="badge-muted">{formatBytes(attachment.size)}</span>
                  <span className="badge-muted">SHA256 {attachment.sha256.slice(0, 12)}…</span>
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}
