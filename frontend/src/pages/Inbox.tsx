import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import { RecipientGroups } from '../components/inbox/RecipientGroups'
import { EmailList } from '../components/inbox/EmailList'
import { EmailViewer } from '../components/inbox/EmailViewer'

export function Inbox() {
  const [recipient, setRecipient] = useState<string>()
  const [activeEmailId, setActiveEmailId] = useState<number>()
  const recipientsQuery = useQuery({
    queryKey: ['recipients'],
    queryFn: api.recipients,
    refetchInterval: 2000,
    refetchIntervalInBackground: false,
  })
  const emailsQuery = useQuery({
    queryKey: ['emails', recipient],
    queryFn: () => api.emails(recipient),
    refetchInterval: 2000,
    refetchIntervalInBackground: false,
  })

  const emails = emailsQuery.data?.emails ?? []
  const recipients = recipientsQuery.data?.recipients ?? []

  useEffect(() => {
    if (!emails.length) {
      setActiveEmailId(undefined)
      return
    }
    if (!activeEmailId || !emails.some((email) => email.id === activeEmailId)) {
      setActiveEmailId(emails[0].id)
    }
  }, [activeEmailId, emails])

  const activeEmail = useMemo(() => emails.find((email) => email.id === activeEmailId), [activeEmailId, emails])
  const activeRecipientSummary = useMemo(() => recipients.find((item) => item.address === recipient), [recipient, recipients])

  const inboxStats = [
    { label: 'Recipient groups', value: recipients.length },
    { label: 'Messages shown', value: emails.length },
    { label: 'Unread', value: recipients.reduce((sum, item) => sum + item.unreadCount, 0) },
  ]

  return (
    <div className="grid gap-6">
      <section className="grid gap-4 lg:grid-cols-3">
        {inboxStats.map((stat) => (
          <div key={stat.label} className="stat-card">
            <div className="section-label">{stat.label}</div>
            <div className="mt-2 text-2xl font-semibold tracking-tight text-white">{stat.value}</div>
          </div>
        ))}
      </section>

      <section className="grid gap-6 xl:grid-cols-[280px_360px_minmax(0,1fr)]">
        <RecipientGroups
          recipients={recipients}
          activeRecipient={recipient}
          activeRecipientSummary={activeRecipientSummary}
          isLoading={recipientsQuery.isLoading}
          error={recipientsQuery.error as Error | null}
          onSelect={(nextRecipient) => {
            setRecipient(nextRecipient)
            setActiveEmailId(undefined)
          }}
        />
        <EmailList
          emails={emails}
          activeEmailId={activeEmail?.id}
          isLoading={emailsQuery.isLoading}
          error={emailsQuery.error as Error | null}
          recipient={recipient}
          onSelect={(email) => setActiveEmailId(email.id)}
        />
        <EmailViewer email={activeEmail} isLoading={emailsQuery.isLoading && !emails.length} recipient={recipient} />
      </section>
    </div>
  )
}
