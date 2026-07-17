<script lang="ts">
  import ChannelEditor from '../components/ChannelEditor.svelte'
  import { epgMapping, settings, send, showToast } from '../lib/stores'
  import type { EpgChannel } from '../lib/types'

  // Local working copy; edits stay here until "Save" pushes the whole map
  // back via saveEpgMapping (the server expects the complete map).
  let working = $state<Record<string, EpgChannel>>({})
  let dirty = $state(false)

  // Sync from the server snapshot only while there are no local edits —
  // other commands (e.g. probing a stream from the editor) also refresh the
  // snapshot and must not wipe unsaved changes. Saving sets dirty back to
  // false, which re-runs this effect and adopts the post-save snapshot.
  $effect(() => {
    const snapshot = $epgMapping
    if (!dirty) {
      working = structuredClone(snapshot)
    }
  })

  let search = $state('')
  let sortBy = $state<'number' | 'name' | 'playlist' | 'group'>('number')
  let sortDir = $state(1)
  let selected = $state<Record<string, boolean>>({})
  let bulkMode = $state(false)
  let editorIds = $state<string[] | null>(null)

  let categoryColors = $derived.by(() => {
    const out: Record<string, string> = {}
    for (const pair of ($settings?.epgCategoriesColors ?? '').split('|')) {
      const [cat, color] = pair.split(':')
      if (cat && color) out[cat] = color
    }
    return out
  })

  function playlistName(ch: EpgChannel): string {
    const id = ch['_file.m3u.id'] ?? ''
    return ($settings?.files?.m3u?.[id]?.name as string) ?? (ch['_file.m3u.name'] ?? '-')
  }

  function xmltvName(ch: EpgChannel): string {
    const file = ch['x-xmltv-file'] ?? '-'
    if (file === '-' || file === 'Threadfin Dummy') return file
    const fileID = file.substring(0, file.lastIndexOf('.'))
    return ($settings?.files?.xmltv?.[fileID]?.name as string) ?? file
  }

  function matches(ch: EpgChannel): boolean {
    if (!search) return true
    const q = search.toLowerCase()
    return [
      ch['x-name'],
      ch['x-group-title'],
      ch['x-mapping'],
      ch['tvg-id'],
      String(ch['x-channelID']),
      playlistName(ch),
    ]
      .filter(Boolean)
      .some((v) => String(v).toLowerCase().includes(q))
  }

  function sorted(entries: [string, EpgChannel][]): [string, EpgChannel][] {
    return entries.sort((a, b) => {
      let result = 0
      switch (sortBy) {
        case 'number':
          result = parseFloat(String(a[1]['x-channelID'])) - parseFloat(String(b[1]['x-channelID']))
          break
        case 'name':
          result = String(a[1]['x-name']).localeCompare(String(b[1]['x-name']))
          break
        case 'playlist':
          result = playlistName(a[1]).localeCompare(playlistName(b[1]))
          break
        case 'group':
          result = String(a[1]['x-group-title'] ?? '').localeCompare(String(b[1]['x-group-title'] ?? ''))
          break
      }
      return result * sortDir
    })
  }

  let active = $derived(
    sorted(Object.entries(working).filter(([, ch]) => ch['x-active'] && matches(ch))),
  )
  let inactive = $derived(
    sorted(Object.entries(working).filter(([, ch]) => !ch['x-active'] && matches(ch))),
  )
  let activeTotal = $derived(Object.values(working).filter((ch) => ch['x-active']).length)
  let selectedIds = $derived(Object.keys(selected).filter((k) => selected[k]))

  function setSort(column: typeof sortBy) {
    if (sortBy === column) {
      sortDir = -sortDir
    } else {
      sortBy = column
      sortDir = 1
    }
  }

  function openEditor(id: string) {
    editorIds = bulkMode && selectedIds.length > 0 ? selectedIds : [id]
  }

  function applyChanges(ids: string[], changes: Partial<EpgChannel>, renumberStart?: number) {
    for (const id of ids) {
      Object.assign(working[id], changes)
      // Mapping/file cleared to "-" always deactivates (legacy behavior).
      if (working[id]['x-mapping'] === '-' || working[id]['x-xmltv-file'] === '-') {
        working[id]['x-active'] = false
      }
    }

    if (renumberStart !== undefined && ids.length > 1) {
      const byNumber = [...ids].sort(
        (a, b) => parseFloat(String(working[a]['x-channelID'])) - parseFloat(String(working[b]['x-channelID'])),
      )
      let n = renumberStart
      for (const id of byNumber) {
        working[id]['x-channelID'] = String(n)
        n += 1
      }
    }

    dirty = true
    selected = {}
  }

  function changeNumber(id: string, value: string) {
    working[id]['x-channelID'] = value
    dirty = true
  }

  async function save() {
    const result = await send('saveEpgMapping', { epgMapping: working })
    if (result) {
      dirty = false
      showToast('success', 'Mapping saved.')
    }
  }

  function toggleAll(entries: [string, EpgChannel][], value: boolean) {
    for (const [id] of entries) selected[id] = value
  }
</script>

<div class="toolbar">
  <h2 style="margin:0">Mapping</h2>
  <span class="badge">{activeTotal} active / {Object.keys(working).length} total</span>
  {#if dirty}
    <span class="badge warn">unsaved changes</span>
  {/if}
  <div class="spacer"></div>
  <input type="search" placeholder="Search channels…" bind:value={search} style="max-width:240px" />
  <button
    class:primary={bulkMode}
    onclick={() => {
      bulkMode = !bulkMode
      if (!bulkMode) selected = {}
    }}
  >
    Bulk edit
  </button>
  {#if bulkMode && selectedIds.length > 0}
    <button class="primary" onclick={() => (editorIds = selectedIds)}>Edit {selectedIds.length} selected</button>
  {/if}
  <button class="primary" onclick={save} disabled={!dirty}>Save</button>
</div>

{#snippet channelTable(entries: [string, EpgChannel][], label: string)}
  <h3>{label} <span class="muted">({entries.length})</span></h3>
  <div class="panel scroll-x" style="margin-bottom:18px">
    {#if entries.length === 0}
      <p class="muted">No channels.</p>
    {:else}
      <table class="data">
        <thead>
          <tr>
            {#if bulkMode}
              <th><input type="checkbox" onchange={(e) => toggleAll(entries, (e.target as HTMLInputElement).checked)} /></th>
            {/if}
            <th class="sortable" onclick={() => setSort('number')}>#</th>
            <th></th>
            <th class="sortable" onclick={() => setSort('name')}>Name</th>
            <th class="sortable" onclick={() => setSort('playlist')}>Playlist</th>
            <th class="sortable" onclick={() => setSort('group')}>Group</th>
            <th>XMLTV file</th>
            <th>XMLTV channel</th>
          </tr>
        </thead>
        <tbody>
          {#each entries as [id, ch] (id)}
            <tr class="clickable">
              {#if bulkMode}
                <td onclick={(e) => e.stopPropagation()}>
                  <input type="checkbox" bind:checked={selected[id]} />
                </td>
              {/if}
              <td class="num" onclick={(e) => e.stopPropagation()}>
                <input
                  type="text"
                  value={ch['x-channelID']}
                  onchange={(e) => changeNumber(id, (e.target as HTMLInputElement).value)}
                />
              </td>
              <td class="logo" onclick={() => openEditor(id)}>
                {#if ch['tvg-logo']}
                  <img src={ch['tvg-logo']} alt="" loading="lazy" onerror={(e) => ((e.target as HTMLImageElement).style.display = 'none')} />
                {/if}
              </td>
              <td onclick={() => openEditor(id)}>
                <span
                  class="name"
                  style={categoryColors[(ch['x-category'] ?? '').split(':')[0]]
                    ? `border-left: 3px solid ${categoryColors[(ch['x-category'] ?? '').split(':')[0]]}; padding-left: 6px`
                    : ''}
                >
                  {ch['x-name']}
                </span>
              </td>
              <td onclick={() => openEditor(id)}>{playlistName(ch)}</td>
              <td onclick={() => openEditor(id)}>{ch['x-group-title']}</td>
              <td onclick={() => openEditor(id)}>{xmltvName(ch)}</td>
              <td class="muted" onclick={() => openEditor(id)}>
                {String(ch['x-mapping'] ?? '').length > 24
                  ? String(ch['x-mapping']).slice(0, 24) + '…'
                  : ch['x-mapping']}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
{/snippet}

{@render channelTable(active, 'Active channels')}
{@render channelTable(inactive, 'Inactive channels')}

{#if editorIds}
  <ChannelEditor ids={editorIds} channels={working} onapply={applyChanges} onclose={() => (editorIds = null)} />
{/if}

<style>
  td.num input {
    width: 70px;
    padding: 4px 6px;
  }

  td.logo img {
    height: 26px;
    max-width: 52px;
    object-fit: contain;
  }

  h3 {
    margin: 6px 0 10px;
  }
</style>
