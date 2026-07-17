<script lang="ts">
  import FileEditor from '../components/FileEditor.svelte'
  import { settings } from '../lib/stores'
  import type { PlaylistFile } from '../lib/types'

  let rows = $derived(
    Object.entries($settings?.files?.xmltv ?? {}).sort((a, b) =>
      (a[1].name ?? '').localeCompare(b[1].name ?? ''),
    ),
  )

  let editing = $state<{ id: string; file: PlaylistFile | null } | null>(null)
</script>

<div class="toolbar">
  <h2 style="margin:0">XMLTV</h2>
  <div class="spacer"></div>
  <button class="primary" onclick={() => (editing = { id: '-', file: null })}>New XMLTV file</button>
</div>

<div class="panel scroll-x">
  {#if rows.length === 0}
    <p class="muted">No XMLTV files yet.</p>
  {:else}
    <table class="data">
      <thead>
        <tr>
          <th>Name</th>
          <th>Last update</th>
          <th>Availability</th>
          <th>Channels</th>
          <th>Programs</th>
        </tr>
      </thead>
      <tbody>
        {#each rows as [id, file] (id)}
          <tr class="clickable" onclick={() => (editing = { id, file })}>
            <td>{file.name}</td>
            <td class="muted">{file['last.update'] ?? '-'}</td>
            <td>{file['provider.availability'] ?? 0}%</td>
            <td>{file.compatibility?.['xmltv.channels'] ?? '-'}</td>
            <td>{file.compatibility?.['xmltv.programs'] ?? '-'}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

{#if editing}
  <FileEditor kind="xmltv" id={editing.id} data={editing.file} onclose={() => (editing = null)} />
{/if}
