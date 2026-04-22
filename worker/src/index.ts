import PostalMime from 'postal-mime'

type Env = {
  EMAILDASH_WEBHOOK_URL: string
  EMAILDASH_WEBHOOK_SECRET: string
}

type ParsedAttachment = {
  filename: string
  contentType: string
  size: number
  sha256: string
  content?: string
}

async function hex(input: ArrayBuffer): Promise<string> {
  return Array.from(new Uint8Array(input)).map((b) => b.toString(16).padStart(2, '0')).join('')
}

async function sign(secret: string, timestamp: string, body: string): Promise<string> {
  const key = await crypto.subtle.importKey('raw', new TextEncoder().encode(secret), { name: 'HMAC', hash: 'SHA-256' }, false, ['sign'])
  const payload = new TextEncoder().encode(`${timestamp}.${body}`)
  const signature = await crypto.subtle.sign('HMAC', key, payload)
  return `v1=${await hex(signature)}`
}

async function digestBase64(content: string): Promise<string> {
  const bytes = Uint8Array.from(atob(content), (c) => c.charCodeAt(0))
  const hash = await crypto.subtle.digest('SHA-256', bytes)
  return hex(hash)
}

export default {
  async email(message: ForwardableEmailMessage, env: Env): Promise<void> {
    const parser = new PostalMime()
    const parsed = await parser.parse(message.raw)
    const attachments: ParsedAttachment[] = await Promise.all(
      (parsed.attachments ?? []).map(async (attachment) => {
        const content = typeof attachment.content === 'string' ? attachment.content : btoa(String.fromCharCode(...new Uint8Array(attachment.content as ArrayBuffer)))
        return {
          filename: attachment.filename ?? 'attachment.bin',
          contentType: attachment.mimeType ?? 'application/octet-stream',
          size: attachment.content ? (attachment.content as ArrayBuffer).byteLength ?? content.length : content.length,
          sha256: await digestBase64(content),
          content,
        }
      }),
    )
    const payload = {
      provider: 'cloudflare',
      receivedAt: new Date().toISOString(),
      messageId: parsed.messageId ?? '',
      mailFrom: message.from,
      rcptTo: [message.to],
      subject: parsed.subject ?? '',
      text: parsed.text ?? '',
      html: parsed.html ?? '',
      headers: Object.fromEntries((parsed.headers ?? []).map((header) => [header.key, Array.isArray(header.value) ? header.value : [String(header.value)]])),
      attachments,
      rawSize: Number(message.rawSize ?? 0),
    }
    const body = JSON.stringify(payload)
    const timestamp = Math.floor(Date.now() / 1000).toString()
    const response = await fetch(env.EMAILDASH_WEBHOOK_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Emaildash-Timestamp': timestamp,
        'X-Emaildash-Signature': await sign(env.EMAILDASH_WEBHOOK_SECRET, timestamp, body),
      },
      body,
    })
    if (!response.ok) {
      throw new Error(`Webhook delivery failed: ${response.status} ${await response.text()}`)
    }
  },
}
