import type { Command, ServerResponse } from './types'

// The Go server handles one command per WebSocket connection and closes it
// after replying (see WS() in src/webserver.go), so each request opens a
// fresh connection. Auth uses the Token cookie passed as a query parameter;
// every response carries a refreshed token that must be written back.

function getCookie(name: string): string {
  const match = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'))
  return match ? decodeURIComponent(match[1]) : ''
}

function wsURL(): string {
  const proto = window.location.protocol === 'https:' ? 'wss://' : 'ws://'
  return `${proto}${window.location.host}/data/?Token=${getCookie('Token')}`
}

export class ApiError extends Error {
  reload: boolean
  constructor(message: string, reload = false) {
    super(message)
    this.reload = reload
  }
}

let queue: Promise<unknown> = Promise.resolve()

/** Send one command to Threadfin and resolve with its response. */
export function request(cmd: Command, data: Record<string, unknown> = {}): Promise<ServerResponse> {
  // Serialize requests: the backend mutates shared state per command and the
  // old UI enforced one in-flight request; we keep that behavior.
  const next = queue.then(() => doRequest(cmd, data))
  queue = next.catch(() => undefined)
  return next
}

function doRequest(cmd: Command, data: Record<string, unknown>): Promise<ServerResponse> {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsURL())
    let settled = false

    ws.onopen = () => {
      ws.send(JSON.stringify({ ...data, cmd }))
    }

    ws.onerror = () => {
      if (!settled) {
        settled = true
        reject(new ApiError('No WebSocket connection to Threadfin could be established.'))
      }
    }

    ws.onclose = () => {
      if (!settled) {
        settled = true
        reject(new ApiError('Connection closed before a response was received.'))
      }
    }

    ws.onmessage = (e) => {
      settled = true
      let response: ServerResponse
      try {
        response = JSON.parse(e.data)
      } catch {
        reject(new ApiError('Invalid response from server.'))
        ws.close()
        return
      }

      if (response.token) {
        document.cookie = `Token=${response.token}; path=/`
      }

      if (response.status === false) {
        reject(new ApiError(response.err ?? 'Unknown server error', response.reload ?? false))
      } else {
        resolve(response)
      }
      ws.close()
    }
  })
}

/** Login via the classic /web/ form endpoint, which sets the Token cookie. */
export async function login(username: string, password: string, confirm?: string): Promise<boolean> {
  const body = new URLSearchParams({ username, password })
  if (confirm !== undefined) body.set('confirm', confirm)

  const res = await fetch('/web/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body,
    credentials: 'same-origin',
  })

  if (!res.ok) return false
  // On success the server redirects to /web and sets the Token cookie; on
  // failure it re-renders the login page without a cookie.
  return getCookie('Token') !== '' && getCookie('Token') !== '-'
}

export function logout(): void {
  document.cookie = 'Token=-; path=/'
  window.location.reload()
}

export function hasToken(): boolean {
  const token = getCookie('Token')
  return token !== '' && token !== '-'
}
