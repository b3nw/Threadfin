<script lang="ts">
  import FileEditor from '../components/FileEditor.svelte'
  import { settings, clientInfo } from '../lib/stores'
  import type { PlaylistFile } from '../lib/types'

  type Row = { id: string; kind: 'm3u' | 'hdhr'; file: PlaylistFile }

  let rows = $derived.by(() => {
    const out: Row[] = []
    for (const kind of ['m3u', 'hdhr'] as const) {
      const files = $settings?.files?.[kind] ?? {}
      for (const [id, file] of Object.entries(files)) {
        out.push({ id, kind, file })
      }
    }
    return out.sort((a, b) => (a.file.name ?? '').localeCompare(b.file.name ?? ''))
  })

  let editing = $state<{ kind: 'm3u' | 'hdhr'; id: string; file: PlaylistFile | null } | null>(null)
  let choosingType = $state(false)

  function newPlaylist(kind: 'm3u' | 'hdhr') {
    choosingType = false
    editing = { kind, id: '-', file: null }
  }

  function availabilityBadge(v: unknown): string {
    const n = Number(v ?? 0)
    if (n >= 100) return 'ok'
    if (n >= 50) return 'warn'
    return 'err'
  }
</script>

<div class="toolbar">
  <h2 style="margin:0">Playlists</h2>
  <span class="badge">Streams {$clientInfo?.streams ?? '–'}</span>
  <div class="spacer"></div>
  <button class="primary" onclick={() => (choosingType = true)}>New playlist</button>
</div>

<div class="panel scroll-x">
  {#if rows.length === 0}
    <p class="muted">No playlists yet. Add an M3U playlist or HDHomeRun tuner to get started.</p>
  {:else}
    <table class="data">
      <thead>
        <tr>
          <th>Name</th>
          <th>Type</th>
          <th>Tuner</th>
          <th>Last update</th>
          <th>Availability</th>
          <th>Streams</th>
          <th>group-title</th>
          <th>tvg-id</th>
        </tr>
      </thead>
      <tbody>
        {#each rows as row (row.kind + row.id)}
          <tr class="clickable" onclick={() => (editing = { kind: row.kind, id: row.id, file: row.file })}>
            <td>{row.file.name}</td>
            <td>{(row.file.type ?? row.kind).toUpperCase()}</td>
            <td>{$settings?.buffer !== '-' ? (row.file.tuner ?? 1) : '-'}</td>
            <td class="muted">{row.file['last.update'] ?? '-'}</td>
            <td><span class="badge {availabilityBadge(row.file['provider.availability'])}">{row.file['provider.availability'] ?? 0}%</span></td>
            <td>{row.file.compatibility?.streams ?? '-'}</td>
            <td>{row.file.compatibility?.['group.title'] ?? '-'}%</td>
            <td>{row.file.compatibility?.['tvg.id'] ?? '-'}%</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

{#if choosingType}
  <div class="toolbar" style="margin-top:14px">
    <span class="muted">Playlist type:</span>
    <button onclick={() => newPlaylist('m3u')}>M3U</button>
    <button onclick={() => newPlaylist('hdhr')}>HDHomeRun</button>
    <button onclick={() => (choosingType = false)}>Cancel</button>
  </div>
{/if}

{#if editing}
  <FileEditor kind={editing.kind} id={editing.id} data={editing.file} onclose={() => (editing = null)} />
{/if}
