import DOMPurify from 'dompurify'
import type { Email } from '../../lib/api'

type Props = {
  email?: Email
}

export function EmailViewer({ email }: Props) {
  if (!email) {
    return <section className="rounded-2xl border border-slate-800 bg-slate-900/70 p-6 text-slate-400">Select message.</section>
  }

  return (
    <section className="rounded-2xl border border-slate-800 bg-slate-900/70 p-6">
      <div className="mb-4 border-b border-slate-800 pb-4">
        <h2 className="text-xl font-semibold text-white">{email.subject || '(no subject)'}</h2>
        <div className="mt-2 text-sm text-slate-300">From {email.mailFrom}</div>
        <div className="text-sm text-slate-400">To {email.recipients.join(', ')}</div>
      </div>
      {email.htmlBody ? (
        <div className="prose prose-invert max-w-none" dangerouslySetInnerHTML={{ __html: DOMPurify.sanitize(email.htmlBody) }} />
      ) : (
        <pre className="whitespace-pre-wrap break-words text-sm text-slate-200">{email.textBody}</pre>
      )}
      {email.attachments.length > 0 && (
        <div className="mt-6 border-t border-slate-800 pt-4">
          <div className="mb-2 text-sm font-medium text-slate-200">Attachments</div>
          <ul className="space-y-2 text-sm text-slate-300">
            {email.attachments.map((attachment) => (
              <li key={attachment.id}>{attachment.filename} · {attachment.size} bytes</li>
            ))}
          </ul>
        </div>
      )}
    </section>
  )
}
