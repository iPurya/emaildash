import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import { RecipientGroups } from '../components/inbox/RecipientGroups'
import { EmailList } from '../components/inbox/EmailList'
import { EmailViewer } from '../components/inbox/EmailViewer'

export function Inbox() {
  const [recipient, setRecipient] = useState<string>()
  const recipientsQuery = useQuery({ queryKey: ['recipients'], queryFn: api.recipients })
  const emailsQuery = useQuery({ queryKey: ['emails', recipient], queryFn: () => api.emails(recipient) })
  const [activeEmailId, setActiveEmailId] = useState<number>()

  const activeEmail = useMemo(() => emailsQuery.data?.emails.find((email) => email.id === activeEmailId) ?? emailsQuery.data?.emails[0], [activeEmailId, emailsQuery.data])

  return (
    <div className="grid gap-6 lg:grid-cols-[280px_360px_1fr]">
      <RecipientGroups recipients={recipientsQuery.data?.recipients ?? []} activeRecipient={recipient} onSelect={setRecipient} />
      <EmailList emails={emailsQuery.data?.emails ?? []} activeEmailId={activeEmail?.id} onSelect={(email) => setActiveEmailId(email.id)} />
      <EmailViewer email={activeEmail} />
    </div>
  )
}
