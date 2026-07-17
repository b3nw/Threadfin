import { writable, derived, get } from 'svelte/store'
import { request, ApiError } from './api'
import type { Command, ServerResponse } from './types'

export const server = writable<ServerResponse | null>(null)
export const loading = writable(false)
export const needsLogin = writable(false)
export const toast = writable<{ kind: 'success' | 'error' | 'info'; text: string } | null>(null)

let toastTimer: ReturnType<typeof setTimeout> | undefined
export function showToast(kind: 'success' | 'error' | 'info', text: string) {
  toast.set({ kind, text })
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => toast.set(null), 6000)
}

export const clientInfo = derived(server, ($s) => $s?.clientInfo)
export const settings = derived(server, ($s) => $s?.settings)
export const epgMapping = derived(server, ($s) => $s?.xepg?.epgMapping ?? {})
export const xmltvMap = derived(server, ($s) => $s?.xepg?.xmltvMap ?? {})
export const users = derived(server, ($s) => $s?.users ?? {})
export const log = derived(server, ($s) => $s?.log)

/**
 * Send a command, merge the full response snapshot into the store and surface
 * alerts. Most commands return the complete state (setDefaultResponseData on
 * the Go side), so a successful call always refreshes everything.
 */
export async function send(cmd: Command, data: Record<string, unknown> = {}): Promise<ServerResponse | null> {
  loading.set(true)
  try {
    const response = await request(cmd, data)
    // Some commands (uploadLogo) reply without the full state snapshot;
    // only replace the store when the response carries one.
    if (response.settings) server.set(response)
    needsLogin.set(false)
    if (response.alert) showToast('info', response.alert)
    if (response.openLink) window.open(response.openLink, '_self')
    if (response.reload) {
      // e.g. restore finished or web auth was just enabled; give the toast a
      // moment to be seen before the page reloads.
      setTimeout(() => window.location.reload(), 1200)
    }
    return response
  } catch (e) {
    if (e instanceof ApiError && e.reload) {
      // Token expired or invalid: drop to the login screen.
      needsLogin.set(true)
    } else {
      showToast('error', e instanceof Error ? e.message : String(e))
    }
    return null
  } finally {
    loading.set(false)
  }
}

export async function refresh(): Promise<ServerResponse | null> {
  return send('getServerConfig')
}

/** Poll logs / connection counts without replacing the full state snapshot. */
export async function pollLog(): Promise<void> {
  try {
    const response = await request('updateLog')
    server.update((current) => {
      if (!current) return current
      return { ...current, log: response.log, clientInfo: { ...current.clientInfo!, ...response.clientInfo } }
    })
  } catch {
    // Polling failures are non-fatal; the next tick retries.
  }
}

export function isAuthEnabled(): boolean {
  return get(server)?.settings?.['authentication.web'] === true
}
