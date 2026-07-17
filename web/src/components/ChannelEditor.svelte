<script lang="ts">
  import Modal from './Modal.svelte'
  import FormRow from './FormRow.svelte'
  import { send, settings, xmltvMap, showToast } from '../lib/stores'
  import type { EpgChannel, ProbeInfo } from '../lib/types'

  let {
    ids,
    channels,
    onapply,
    onclose,
  }: {
    ids: string[] // one id = single edit, several = bulk edit
    channels: Record<string, EpgChannel>
    onapply: (ids: string[], changes: Partial<EpgChannel>, renumberStart?: number) => void
    onclose: () => void
  } = $props()

  const bulk = ids.length > 1
  const first = channels[ids[0]]

  // Draft state; on Done only fields that differ from `initial` are applied,
  // mirroring the legacy UI's `.changed` tracking (menu_ts.ts donePopupData).
  const initial: Partial<EpgChannel> = {
    'x-active': first['x-active'],
    'x-name': first['x-name'],
    'x-description': first['x-description'] ?? '',
    'x-update-channel-name': first['x-update-channel-name'] ?? false,
    'tvg-logo': first['tvg-logo'] ?? '',
    'x-update-channel-icon': first['x-update-channel-icon'] ?? false,
    'x-category': first['x-category'] ?? '',
    'x-group-title': first['x-group-title'] ?? '',
    'x-xmltv-file': first['x-xmltv-file'] ?? '-',
    'x-mapping': first['x-mapping'] ?? '-',
    'x-ppv-extra': first['x-ppv-extra'] ?? '',
    'x-backup-channel-1': first['x-backup-channel-1'] ?? '',
    'x-backup-channel-2': first['x-backup-channel-2'] ?? '',
    'x-backup-channel-3': first['x-backup-channel-3'] ?? '',
  }

  let draft = $state({ ...initial })
  let renumberStart = $state(String(first['x-channelID'] ?? ''))
  let probeInfo = $state<ProbeInfo | null>(null)
  let probing = $state(false)

  let categories = $derived(
    ($settings?.epgCategories ?? '')
      .split('|')
      .filter(Boolean)
      .map((c) => {
        const [label, value] = c.split(':')
        return { label, value }
      }),
  )

  // xmltvMap is keyed by the generated XMLTV filename ("X<id>.xml") plus
  // "Threadfin Dummy"; display names come from settings.files.xmltv.
  let xmltvFiles = $derived(
    Object.keys($xmltvMap).map((key) => {
      let label = key
      if (key !== 'Threadfin Dummy') {
        const fileID = key.substring(0, key.lastIndexOf('.'))
        label = ($settings?.files?.xmltv?.[fileID]?.name as string) ?? key
      }
      return { value: key, label }
    }),
  )

  let epgChannels = $derived(Object.entries($xmltvMap[draft['x-xmltv-file'] ?? ''] ?? {}))

  let channelNames = $derived(
    Object.values(channels)
      .map((c) => c['x-name'])
      .filter(Boolean)
      .sort(),
  )

  function onFileChange() {
    // Selecting a new guide file resets the mapping to the stream's tvg-id
    // when that channel exists in the file (legacy setXmltvChannel behavior).
    const tvgId = first['tvg-id'] ?? '-'
    const map = $xmltvMap[draft['x-xmltv-file'] ?? ''] ?? {}
    draft['x-mapping'] = tvgId && map[tvgId] ? tvgId : '-'
    onMappingChange()
  }

  function onMappingChange() {
    const mapping = draft['x-mapping'] ?? '-'
    const file = draft['x-xmltv-file'] ?? '-'
    draft['x-active'] = mapping !== '-' && file !== '-'

    if (file !== 'Threadfin Dummy' && draft['x-active'] && (!bulk || draft['x-update-channel-icon'])) {
      const icon = $xmltvMap[file]?.[mapping]?.icon
      if (icon) draft['tvg-logo'] = icon
    }
  }

  async function probe() {
    if (!first.url) return
    probing = true
    try {
      const response = await send('probeChannel', { probeURL: first.url })
      probeInfo = response?.probeInfo ?? null
      if (!probeInfo?.resolution) showToast('info', 'Probe returned no stream details.')
    } finally {
      probing = false
    }
  }

  async function uploadLogo(e: Event) {
    const file = (e.target as HTMLInputElement).files?.[0]
    if (!file) return
    const reader = new FileReader()
    reader.onload = async () => {
      const response = await send('uploadLogo', {
        base64: String(reader.result),
        filename: file.name,
      })
      if (response?.logoURL) {
        draft['tvg-logo'] = response.logoURL
        showToast('success', 'Logo uploaded.')
      }
    }
    reader.readAsDataURL(file)
  }

  function done() {
    const changes: Partial<EpgChannel> = {}
    for (const key of Object.keys(initial) as (keyof EpgChannel)[]) {
      if (draft[key] !== initial[key]) (changes as Record<string, unknown>)[key] = draft[key]
    }
    const start = bulk && renumberStart.trim() !== '' ? parseFloat(renumberStart) : undefined
    onapply(ids, changes, Number.isFinite(start) ? start : undefined)
    onclose()
  }
</script>

<Modal title={bulk ? `Bulk edit — ${ids.length} channels` : (first['x-name'] ?? 'Channel')} wide {onclose}>
  {#if bulk}
    <FormRow label="Start channel number" hint="Selected channels are renumbered sequentially from here.">
      <input type="text" bind:value={renumberStart} />
    </FormRow>
  {/if}

  <FormRow label="Active">
    <input type="checkbox" bind:checked={draft['x-active']} />
  </FormRow>

  <FormRow label="Channel name" hint={`${first['tvg-id'] ?? ''} (${first['x-epg'] ?? ''})`}>
    <input type="text" bind:value={draft['x-name']} readonly={bulk} />
  </FormRow>

  <FormRow label="Description">
    <input type="text" bind:value={draft['x-description']} />
  </FormRow>

  {#if first['_uuid.key']}
    <FormRow label="Auto-update name" hint="Keep the channel name in sync with the playlist.">
      <input type="checkbox" bind:checked={draft['x-update-channel-name']} />
    </FormRow>
  {/if}

  <FormRow label="Logo URL">
    <div class="logo-row">
      {#if draft['tvg-logo']}
        <img class="logo-preview" src={draft['tvg-logo']} alt="" />
      {/if}
      <input type="text" bind:value={draft['tvg-logo']} />
      <label class="upload">
        Upload
        <input type="file" accept="image/*" onchange={uploadLogo} hidden />
      </label>
    </div>
  </FormRow>

  <FormRow label="Auto-update logo" hint="Take the logo from the XMLTV guide when available.">
    <input type="checkbox" bind:checked={draft['x-update-channel-icon']} />
  </FormRow>

  <FormRow label="EPG category">
    <select bind:value={draft['x-category']}>
      <option value="">-</option>
      {#each categories as cat (cat.value)}
        <option value={cat.value}>{cat.label}</option>
      {/each}
    </select>
  </FormRow>

  <FormRow label="M3U group title" hint={first['group-title'] ?? ''}>
    <input type="text" bind:value={draft['x-group-title']} />
  </FormRow>

  <FormRow label="XMLTV file">
    <select bind:value={draft['x-xmltv-file']} onchange={onFileChange}>
      <option value="-">-</option>
      {#each xmltvFiles as f (f.value)}
        <option value={f.value}>{f.label}</option>
      {/each}
    </select>
  </FormRow>

  <FormRow label="XMLTV channel" hint="Guide channel mapped to this stream. '-' disables the channel.">
    <input type="text" list="epg-channel-list" bind:value={draft['x-mapping']} onchange={onMappingChange} />
    <datalist id="epg-channel-list">
      <option value="-">-</option>
      {#each epgChannels as [chId, ch] (chId)}
        <option value={chId}>{ch['display-name'] ?? chId}</option>
      {/each}
    </datalist>
  </FormRow>

  {#if draft['x-mapping'] === 'PPV'}
    <FormRow label="PPV extra">
      <input type="text" bind:value={draft['x-ppv-extra']} />
    </FormRow>
  {/if}

  {#each [1, 2, 3] as n (n)}
    <FormRow label={`Backup channel ${n}`} hint="Fallback stream used when the primary is offline.">
      <input type="text" list="channel-name-list" bind:value={draft[`x-backup-channel-${n}`]} />
    </FormRow>
  {/each}
  <datalist id="channel-name-list">
    {#each channelNames as name (name)}
      <option value={name}></option>
    {/each}
  </datalist>

  {#if probeInfo?.resolution}
    <div class="probe panel">
      <strong>Probe:</strong>
      Resolution {probeInfo.resolution} · {probeInfo.frameRate} FPS · Audio {probeInfo.audioChannel}
    </div>
  {/if}

  {#snippet footer()}
    {#if !bulk && first.url}
      <button onclick={probe} disabled={probing}>{probing ? 'Probing…' : 'Probe stream'}</button>
    {/if}
    <button onclick={onclose}>Cancel</button>
    <button class="primary" onclick={done}>Done</button>
  {/snippet}
</Modal>

<style>
  .logo-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .logo-preview {
    height: 32px;
    max-width: 64px;
    object-fit: contain;
    background: var(--bg-input);
    border-radius: 4px;
  }

  .upload {
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 6px 12px;
    cursor: pointer;
    white-space: nowrap;
  }

  .upload:hover {
    background: var(--bg-hover);
  }

  .probe {
    margin-top: 12px;
    padding: 10px 14px;
  }
</style>
