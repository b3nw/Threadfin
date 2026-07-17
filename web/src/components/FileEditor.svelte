<script lang="ts">
  import Modal from './Modal.svelte'
  import FormRow from './FormRow.svelte'
  import { send, settings } from '../lib/stores'
  import type { PlaylistFile } from '../lib/types'

  let {
    kind,
    id,
    data,
    onclose,
  }: {
    kind: 'm3u' | 'hdhr' | 'xmltv'
    id: string // "-" for a new entry
    data: PlaylistFile | null
    onclose: () => void
  } = $props()

  const titles = { m3u: 'M3U Playlist', hdhr: 'HDHomeRun Tuner', xmltv: 'XMLTV File' }
  const commands = {
    m3u: { save: 'saveFilesM3U', update: 'updateFileM3U' },
    hdhr: { save: 'saveFilesHDHR', update: 'updateFileHDHR' },
    xmltv: { save: 'saveFilesXMLTV', update: 'updateFileXMLTV' },
  } as const

  const isNew = id === '-'

  let name = $state(data?.name ?? '')
  let description = $state(data?.description ?? '')
  let source = $state((data?.['file.source'] as string) ?? '')
  let buffer = $state((data?.buffer as string) ?? $settings?.buffer ?? '-')
  let tuner = $state(Number(data?.tuner ?? 1))
  let proxyIP = $state((data?.['http_proxy.ip'] as string) ?? '')
  let proxyPort = $state((data?.['http_proxy.port'] as string) ?? '')
  let headerOrigin = $state((data?.['http_headers.origin'] as string) ?? '')
  let headerReferer = $state((data?.['http_headers.referer'] as string) ?? '')

  function payload(): Record<string, unknown> {
    const fields: Record<string, unknown> = {
      name,
      description,
      'file.source': source,
      'http_proxy.ip': proxyIP,
      'http_proxy.port': proxyPort,
    }
    if (kind !== 'xmltv') {
      fields.buffer = buffer
      fields.tuner = tuner
      fields['http_headers.origin'] = headerOrigin
      fields['http_headers.referer'] = headerReferer
    }
    return fields
  }

  async function save(update: boolean) {
    const cmd = update ? commands[kind].update : commands[kind].save
    const result = await send(cmd, { files: { [kind]: { [id]: payload() } } })
    if (result) onclose()
  }

  async function remove() {
    if (!confirm(`Delete "${name}"?`)) return
    const result = await send(commands[kind].save, {
      files: { [kind]: { [id]: { ...payload(), delete: true } } },
    })
    if (result) onclose()
  }
</script>

<Modal title={isNew ? `New ${titles[kind]}` : titles[kind]} {onclose}>
  <FormRow label="Name">
    <input type="text" bind:value={name} placeholder="Display name" />
  </FormRow>
  <FormRow label="Description">
    <input type="text" bind:value={description} />
  </FormRow>
  <FormRow label={kind === 'xmltv' ? 'XMLTV URL / path' : 'Source URL / path'}>
    <input type="text" bind:value={source} placeholder={kind === 'xmltv' ? 'https://…/guide.xml' : 'https://…/playlist.m3u'} />
  </FormRow>

  {#if kind !== 'xmltv'}
    <FormRow label="Buffer">
      <select bind:value={buffer}>
        <option value="-">-</option>
        <option value="ffmpeg">FFmpeg</option>
        <option value="vlc">VLC</option>
      </select>
    </FormRow>
    <FormRow label="Tuner" hint="Concurrent streams allowed for this source (needs an active buffer).">
      <select bind:value={tuner}>
        {#each Array.from({ length: 100 }, (_, i) => i + 1) as n (n)}
          <option value={n}>{n}</option>
        {/each}
      </select>
    </FormRow>
  {/if}

  <FormRow label="HTTP proxy IP" hint="Optional proxy used when fetching this source.">
    <input type="text" bind:value={proxyIP} />
  </FormRow>
  <FormRow label="HTTP proxy port">
    <input type="text" bind:value={proxyPort} />
  </FormRow>

  {#if kind !== 'xmltv'}
    <FormRow label="Origin header" hint="Optional Origin header sent with stream requests.">
      <input type="text" bind:value={headerOrigin} />
    </FormRow>
    <FormRow label="Referer header" hint="Optional Referer header sent with stream requests.">
      <input type="text" bind:value={headerReferer} />
    </FormRow>
  {/if}

  {#snippet footer()}
    {#if !isNew}
      <button class="danger" onclick={remove}>Delete</button>
      <button onclick={() => save(true)}>Save &amp; update</button>
    {/if}
    <button onclick={onclose}>Cancel</button>
    <button class="primary" onclick={() => save(false)}>Save</button>
  {/snippet}
</Modal>
