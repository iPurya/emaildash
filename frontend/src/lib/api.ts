export type SetupStatus = { initialized: boolean }
export type Zone = { id: string; name: string; accountId: string; selected: boolean; status: string }
export type CloudflareStatus = {
  zoneId: string
  zoneName: string
  accountId: string
  emailRoutingEnabled: boolean
  emailRoutingStatus: string
  workerScriptName: string
  catchAllEnabled: boolean
  catchAllDestination: string
}
export type RecipientSummary = {
  address: string
  count: number
  unreadCount: number
  latestEmailId?: number
  latestSubject?: string
  latestReceived?: string
}
export type Email = {
  id: number
  provider: string
  providerMessageId: string
  mailFrom: string
  recipients: string[]
  subject: string
  textBody: string
  htmlBody: string
  headers: Record<string, string[]>
  rawSize: number
  readAt?: string
  receivedAt: string
  createdAt: string
  attachments: Array<{ id: number; filename: string; contentType: string; size: number; sha256: string; storagePath?: string }>
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers ?? {})
  if (!headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const response = await fetch(path, {
    credentials: 'include',
    ...init,
    headers,
  })
  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(data.error ?? 'Request failed')
  }
  if (response.status === 204) {
    return undefined as T
  }
  return response.json() as Promise<T>
}

export const api = {
  setupStatus: () => request<SetupStatus>('/api/setup/status'),
  initialize: (password: string) => request<void>('/api/setup/initialize', { method: 'POST', body: JSON.stringify({ password }) }),
  login: (password: string) => request<{ csrfToken: string; expiresAt: string }>('/api/auth/login', { method: 'POST', body: JSON.stringify({ password }) }),
  me: () => request<{ authenticated: boolean; csrfToken: string; expiresAt: string }>('/api/auth/me'),
  logout: () => request<void>('/api/auth/logout', { method: 'POST' }),
  saveCloudflareCredentials: (email: string, apiKey: string) => request<{ zones: Zone[] }>('/api/cloudflare/credentials', { method: 'POST', body: JSON.stringify({ email, apiKey }) }),
  zones: () => request<{ zones: Zone[] }>('/api/cloudflare/zones'),
  provisionZone: (zoneId: string) => request<CloudflareStatus>('/api/cloudflare/zones/' + zoneId + '/provision', { method: 'POST' }),
  cloudflareStatus: () => request<CloudflareStatus>('/api/cloudflare/status'),
  emails: (recipient?: string) => request<{ emails: Email[] }>('/api/emails' + (recipient ? `?recipient=${encodeURIComponent(recipient)}` : '')),
  email: (id: number) => request<Email>('/api/emails/' + id),
  recipients: () => request<{ recipients: RecipientSummary[] }>('/api/recipients'),
  markRead: (id: number) => request<void>('/api/emails/' + id + '/read', { method: 'PATCH' }),
  changePassword: (oldPassword: string, newPassword: string) => request<void>('/api/settings/password', { method: 'POST', body: JSON.stringify({ oldPassword, newPassword }) }),
}
