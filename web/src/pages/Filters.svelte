<script lang="ts">
  import Modal from '../components/Modal.svelte'
  import FormRow from '../components/FormRow.svelte'
  import { settings, server, send } from '../lib/stores'
  import type { Filter } from '../lib/types'

  let rows = $derived(
    Object.entries($settings?.filter ?? {})
      .filter(([id]) => id !== '-1')
      .sort((a, b) => (a[1].name ?? '').localeCompare(b[1].name ?? '')),
  )

  let m3uGroups = $derived($server?.data?.playlist?.m3u?.groups ?? { text: [], value: [] })

  // epgCategories is stored as "Label:value|Label:value|…"
  let categories = $derived(
    ($settings?.epgCategories ?? '')
      .split('|')
      .filter(Boolean)
      .map((c) => {
        const [label, value] = c.split(':')
        return { label, value }
      }),
  )

  let editing = $state<{ id: string; filter: Filter } | null>(null)
  let choosingType = $state(false)

  function newFilter(type: 'group-title' | 'custom-filter') {
    choosingType = false
    editing = {
      id: '-1',
      filter: {
        type,
        active: true,
        startingNumber: '1000',
        // Preselect the first group so saving without touching the select
        // doesn't create an empty group filter.
        filter: type === 'group-title' ? (m3uGroups.value[0] ?? '') : '',
      },
    }
  }

  function openFilter(id: string, filter: Filter) {
    editing = { id, filter: { ...filter } }
  }

  async function save(remove = false) {
    if (!editing) return
    const f = editing.filter
    if (remove && !confirm(`Delete filter "${f.name}"?`)) return

    const payload: Record<string, unknown> = {
      name: f.name ?? '',
      description: f.description ?? '',
      type: f.type,
      startingNumber: f.startingNumber ?? '1000',
      'x-category': f['x-category'] ?? '',
      caseSensitive: f.caseSensitive ?? false,
      filter: f.filter ?? '',
    }
    if (f.type === 'group-title') {
      payload.liveEvent = f.liveEvent ?? false
      payload.include = f.include ?? ''
      payload.exclude = f.exclude ?? ''
    }
    if (remove) payload.delete = true

    const result = await send('saveFilter', { filter: { [editing.id]: payload } })
    if (result) editing = null
  }

  function typeLabel(t?: string): string {
    return t === 'custom-filter' ? 'Custom filter' : 'Group'
  }
</script>

<div class="toolbar">
  <h2 style="margin:0">Filters</h2>
  <div class="spacer"></div>
  <button class="primary" onclick={() => (choosingType = true)}>New filter</button>
</div>

<div class="panel scroll-x">
  {#if rows.length === 0}
    <p class="muted">
      No filters defined. Without filters no streams are processed — add a group or custom filter.
    </p>
  {:else}
    <table class="data">
      <thead>
        <tr>
          <th>Starting number</th>
          <th>Name</th>
          <th>Type</th>
          <th>Filter</th>
        </tr>
      </thead>
      <tbody>
        {#each rows as [id, filter] (id)}
          <tr class="clickable" onclick={() => openFilter(id, filter)}>
            <td>{filter.startingNumber ?? '-'}</td>
            <td>{filter.name}</td>
            <td>{typeLabel(filter.type)}</td>
            <td class="muted">{filter.filter}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

{#if choosingType}
  <div class="toolbar" style="margin-top:14px">
    <span class="muted">Filter type:</span>
    <button onclick={() => newFilter('group-title')}>M3U group</button>
    <button onclick={() => newFilter('custom-filter')}>Custom filter</button>
    <button onclick={() => (choosingType = false)}>Cancel</button>
  </div>
{/if}

{#if editing}
  <Modal
    title={typeLabel(editing.filter.type)}
    onclose={() => (editing = null)}
  >
    <FormRow label="Name">
      <input type="text" bind:value={editing.filter.name} />
    </FormRow>
    <FormRow label="Description">
      <input type="text" bind:value={editing.filter.description} />
    </FormRow>

    {#if editing.filter.type === 'group-title'}
      <FormRow label="M3U group" hint="Streams from this group-title are processed.">
        <select bind:value={editing.filter.filter}>
          {#each m3uGroups.value as value, i (value)}
            <option {value}>{m3uGroups.text[i]}</option>
          {/each}
        </select>
      </FormRow>
      <FormRow label="Live event">
        <input type="checkbox" bind:checked={editing.filter.liveEvent} />
      </FormRow>
      <FormRow label="Case sensitive">
        <input type="checkbox" bind:checked={editing.filter.caseSensitive} />
      </FormRow>
      <FormRow label="Include" hint="Only keep streams whose name contains these words (comma separated).">
        <input type="text" bind:value={editing.filter.include} />
      </FormRow>
      <FormRow label="Exclude" hint="Drop streams whose name contains these words (comma separated).">
        <input type="text" bind:value={editing.filter.exclude} />
      </FormRow>
    {:else}
      <FormRow label="Case sensitive">
        <input type="checkbox" bind:checked={editing.filter.caseSensitive} />
      </FormRow>
      <FormRow label="Filter rule" hint="Matched against group-title and stream names.">
        <input type="text" bind:value={editing.filter.filter} />
      </FormRow>
    {/if}

    <FormRow label="Starting channel" hint="First channel number assigned to streams from this filter.">
      <input type="text" bind:value={editing.filter.startingNumber} />
    </FormRow>
    <FormRow label="EPG category">
      <select bind:value={editing.filter['x-category']}>
        <option value="">-</option>
        {#each categories as cat (cat.value)}
          <option value={cat.value}>{cat.label}</option>
        {/each}
      </select>
    </FormRow>

    {#snippet footer()}
      {#if editing && editing.id !== '-1'}
        <button class="danger" onclick={() => save(true)}>Delete</button>
      {/if}
      <button onclick={() => (editing = null)}>Cancel</button>
      <button class="primary" onclick={() => save(false)}>Save</button>
    {/snippet}
  </Modal>
{/if}
