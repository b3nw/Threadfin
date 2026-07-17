<script lang="ts">
  import FormRow from '../components/FormRow.svelte'
  import { settings, clientInfo, send, showToast } from '../lib/stores'

  type FieldType = 'bool' | 'text' | 'int' | 'float' | 'select' | 'times'

  interface Field {
    key: string // key shown from settings.json
    sendKey?: string // key expected by the save request when it differs
    label: string
    type: FieldType
    options?: { label: string; value: string }[]
    hint?: string
    authOnly?: boolean
  }

  interface Category {
    title: string
    fields: Field[]
  }

  const categories: Category[] = [
    {
      title: 'General',
      fields: [
        { key: 'ssdp', label: 'SSDP / DLNA', type: 'bool', hint: 'Announce Threadfin on the network via SSDP.' },
        { key: 'tuner', label: 'Tuner', type: 'int', hint: 'Number of concurrent streams (used when no buffer is active).' },
        {
          key: 'epgSource',
          label: 'EPG source',
          type: 'select',
          options: [
            { label: 'PMS', value: 'PMS' },
            { label: 'XEPG', value: 'XEPG' },
          ],
          hint: 'XEPG lets Threadfin manage the EPG (recommended).',
        },
        { key: 'epgCategories', label: 'EPG categories', type: 'text', hint: 'Format: Label:value|Label:value' },
        { key: 'epgCategoriesColors', label: 'EPG category colors', type: 'text', hint: 'Format: category:#hex|category:#hex' },
        { key: 'dummy', label: 'Dummy EPG', type: 'bool', hint: 'Generate placeholder guide data for unmapped channels.' },
        {
          key: 'dummyChannel',
          label: 'Dummy program length',
          type: 'select',
          options: [
            { label: 'PPV', value: 'PPV' },
            { label: '30 minutes', value: '30_Minutes' },
            { label: '60 minutes', value: '60_Minutes' },
            { label: '90 minutes', value: '90_Minutes' },
            { label: '120 minutes', value: '120_Minutes' },
            { label: '180 minutes', value: '180_Minutes' },
            { label: '240 minutes', value: '240_Minutes' },
            { label: '360 minutes', value: '360_Minutes' },
          ],
        },
        { key: 'ignoreFilters', label: 'Ignore filters', type: 'bool', hint: 'Process all streams regardless of filters.' },
        { key: 'api', label: 'API', type: 'bool', hint: 'Enable the /api/ interface.' },
      ],
    },
    {
      title: 'Files',
      fields: [
        { key: 'update', label: 'Update schedule', type: 'times', hint: 'Comma-separated times (0000–2359) for automatic playlist/EPG updates.' },
        { key: 'files.update', label: 'Update at startup', type: 'bool' },
        { key: 'temp.path', label: 'Temp folder', type: 'text' },
        { key: 'cache.images', label: 'Cache images', type: 'bool' },
        { key: 'bindIpAddress', label: 'Bind IP address', type: 'text', hint: 'Leave empty to listen on all interfaces.' },
        { key: 'httpThreadfinDomain', label: 'HTTP domain', type: 'text' },
        { key: 'forceHttps', label: 'Force HTTPS', type: 'bool' },
        { key: 'excludeStreamHttps', sendKey: 'excludeStreamsHttps', label: 'Exclude streams from HTTPS', type: 'bool' },
        { key: 'httpsPort', label: 'HTTPS port', type: 'int' },
        { key: 'httpsThreadfinDomain', label: 'HTTPS domain', type: 'text' },
        { key: 'xepg.replace.missing.images', label: 'Replace missing program images', type: 'bool' },
        { key: 'xepg.replace.channel.title', label: 'Append category to channel title', type: 'bool' },
        { key: 'enableNonAscii', label: 'Allow non-ASCII characters', type: 'bool' },
      ],
    },
    {
      title: 'Streaming',
      fields: [
        { key: 'udpxy', label: 'UDPxy address', type: 'text', hint: 'Optional udpxy gateway for UDP multicast streams.' },
        { key: 'buffer.size.kb', label: 'Buffer size (KB)', type: 'int' },
        { key: 'buffer.timeout', label: 'Buffer timeout (ms)', type: 'float' },
        { key: 'user.agent', label: 'User agent', type: 'text' },
        { key: 'ffmpeg.path', label: 'FFmpeg path', type: 'text' },
        { key: 'ffmpeg.options', label: 'FFmpeg options', type: 'text' },
        { key: 'ffmpeg.forceHttp', label: 'FFmpeg: force HTTP', type: 'bool' },
        { key: 'vlc.path', label: 'VLC path', type: 'text' },
        { key: 'vlc.options', label: 'VLC options', type: 'text' },
      ],
    },
    {
      title: 'Backup',
      fields: [
        { key: 'backup.path', label: 'Backup folder', type: 'text' },
        { key: 'backup.keep', label: 'Backups to keep', type: 'int' },
      ],
    },
    {
      title: 'Authentication',
      fields: [
        { key: 'authentication.web', label: 'Web interface', type: 'bool', hint: 'Require login for this web interface.' },
        { key: 'authentication.pms', label: 'PMS / DVR', type: 'bool', authOnly: true },
        { key: 'authentication.m3u', label: 'M3U', type: 'bool', authOnly: true },
        { key: 'authentication.xml', label: 'XML', type: 'bool', authOnly: true },
        { key: 'authentication.api', label: 'API', type: 'bool', authOnly: true },
      ],
    },
  ]

  // Draft holds edited values; only keys in it are sent (like the legacy
  // ".changed" tracking), with type conversion at save time.
  let draft = $state<Record<string, unknown>>({})
  let dirty = $derived(Object.keys(draft).length > 0)

  function current(field: Field): unknown {
    if (field.key in draft) return draft[field.key]
    const value = $settings?.[field.key]
    if (field.type === 'times') return Array.isArray(value) ? value.join(', ') : ''
    return value
  }

  function setValue(field: Field, value: unknown) {
    draft[field.key] = value
  }

  async function save() {
    const payload: Record<string, unknown> = {}
    for (const cat of categories) {
      for (const field of cat.fields) {
        if (!(field.key in draft)) continue
        let value = draft[field.key]
        switch (field.type) {
          case 'int':
            value = parseInt(String(value), 10)
            if (Number.isNaN(value)) continue
            break
          case 'float':
            value = parseFloat(String(value))
            if (Number.isNaN(value)) continue
            break
          case 'times':
            value = String(value)
              .split(',')
              .map((v) => v.trim())
              .filter(Boolean)
            break
        }
        payload[field.sendKey ?? field.key] = value
      }
    }
    const result = await send('saveSettings', { settings: payload })
    if (result) {
      draft = {}
      showToast('success', 'Settings saved.')
    }
  }

  async function backupNow() {
    await send('ThreadfinBackup')
  }

  let restoreBusy = $state(false)
  async function restore(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (!file) return
    if (!confirm('Restore this backup? Current configuration will be replaced.')) return
    restoreBusy = true
    const reader = new FileReader()
    reader.onload = async () => {
      await send('ThreadfinRestore', { base64: String(reader.result) })
      restoreBusy = false
    }
    reader.readAsDataURL(file)
  }

  let authEnabled = $derived(
    'authentication.web' in draft
      ? draft['authentication.web'] === true
      : $settings?.['authentication.web'] === true,
  )

  async function copy(text: string | undefined) {
    if (!text) return
    await navigator.clipboard.writeText(text)
    showToast('success', 'Copied to clipboard.')
  }
</script>

<div class="toolbar">
  <h2 style="margin:0">Settings</h2>
  {#if dirty}
    <span class="badge warn">unsaved changes</span>
  {/if}
  <div class="spacer"></div>
  <button class="primary" onclick={save} disabled={!dirty}>Save settings</button>
</div>

<div class="panel" style="margin-bottom:18px">
  <h3 style="margin-top:0">Service addresses</h3>
  {#each [
    { label: 'DVR (Plex/Emby)', value: $clientInfo?.DVR },
    { label: 'M3U URL', value: $clientInfo?.['m3u-url'] },
    { label: 'XEPG URL', value: $clientInfo?.['xepg-url'] },
  ] as addr (addr.label)}
    <div class="addr">
      <span class="muted">{addr.label}</span>
      <code>{addr.value ?? '-'}</code>
      <button onclick={() => copy(addr.value)}>Copy</button>
    </div>
  {/each}
</div>

{#each categories as cat (cat.title)}
  <div class="panel" style="margin-bottom:18px">
    <h3 style="margin-top:0">{cat.title}</h3>
    {#each cat.fields as field (field.key)}
      {#if !field.authOnly || authEnabled}
        <FormRow label={field.label} hint={field.hint}>
          {#if field.type === 'bool'}
            <input
              type="checkbox"
              checked={current(field) === true}
              onchange={(e) => setValue(field, (e.target as HTMLInputElement).checked)}
            />
          {:else if field.type === 'select'}
            <select
              value={String(current(field) ?? '')}
              onchange={(e) => setValue(field, (e.target as HTMLSelectElement).value)}
            >
              {#each field.options ?? [] as opt (opt.value)}
                <option value={opt.value}>{opt.label}</option>
              {/each}
            </select>
          {:else}
            <input
              type="text"
              value={String(current(field) ?? '')}
              onchange={(e) => setValue(field, (e.target as HTMLInputElement).value)}
            />
          {/if}
        </FormRow>
      {/if}
    {/each}

    {#if cat.title === 'Backup'}
      <div class="toolbar" style="margin:6px 0 0">
        <button onclick={backupNow}>Download backup</button>
        <label class="restore">
          {restoreBusy ? 'Restoring…' : 'Restore from file'}
          <input type="file" accept=".zip" onchange={restore} hidden disabled={restoreBusy} />
        </label>
      </div>
    {/if}
  </div>
{/each}

<style>
  .addr {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 6px 0;
    flex-wrap: wrap;
  }

  .addr span {
    width: 140px;
    flex-shrink: 0;
  }

  .addr code {
    background: var(--bg-input);
    padding: 4px 8px;
    border-radius: 4px;
    word-break: break-all;
  }

  .restore {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 7px 14px;
    cursor: pointer;
  }

  .restore:hover {
    background: var(--bg-hover);
  }
</style>
