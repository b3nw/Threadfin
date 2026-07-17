<script lang="ts">
  import Modal from '../components/Modal.svelte'
  import FormRow from '../components/FormRow.svelte'
  import { users, send } from '../lib/stores'

  type Draft = {
    username: string
    password: string
    confirm: string
    web: boolean
    pms: boolean
    m3u: boolean
    xml: boolean
    api: boolean
  }

  let rows = $derived(Object.entries($users))
  let editing = $state<{ id: string; draft: Draft } | null>(null)
  let error = $state('')

  function open(id: string) {
    const data = id === '-' ? {} : ($users[id]?.data ?? {})
    error = ''
    editing = {
      id,
      draft: {
        username: (data.username as string) ?? '',
        password: '',
        confirm: '',
        web: data['authentication.web'] === true,
        pms: data['authentication.pms'] === true,
        m3u: data['authentication.m3u'] === true,
        xml: data['authentication.xml'] === true,
        api: data['authentication.api'] === true,
      },
    }
  }

  async function save(remove = false) {
    if (!editing) return
    const d = editing.draft

    if (remove && !confirm(`Delete user "${d.username}"?`)) return
    if (!remove && d.password !== d.confirm) {
      error = 'Passwords do not match.'
      return
    }

    const fields: Record<string, unknown> = {
      username: d.username,
      password: d.password,
      confirm: d.confirm,
      'authentication.web': d.web,
      'authentication.pms': d.pms,
      'authentication.m3u': d.m3u,
      'authentication.xml': d.xml,
      'authentication.api': d.api,
    }
    if (remove) fields.delete = true

    const result =
      editing.id === '-'
        ? await send('saveNewUser', { userData: fields })
        : await send('saveUserData', { userData: { [editing.id]: fields } })
    if (result) editing = null
  }

  function check(v: unknown): string {
    return v === true ? '✓' : '–'
  }
</script>

<div class="toolbar">
  <h2 style="margin:0">Users</h2>
  <div class="spacer"></div>
  <button class="primary" onclick={() => open('-')}>New user</button>
</div>

<div class="panel scroll-x">
  {#if rows.length === 0}
    <p class="muted">No users.</p>
  {:else}
    <table class="data">
      <thead>
        <tr>
          <th>Username</th>
          <th>Web</th>
          <th>PMS</th>
          <th>M3U</th>
          <th>XML</th>
          <th>API</th>
        </tr>
      </thead>
      <tbody>
        {#each rows as [id, user] (id)}
          <tr class="clickable" onclick={() => open(id)}>
            <td>{user.data?.username}</td>
            <td>{check(user.data?.['authentication.web'])}</td>
            <td>{check(user.data?.['authentication.pms'])}</td>
            <td>{check(user.data?.['authentication.m3u'])}</td>
            <td>{check(user.data?.['authentication.xml'])}</td>
            <td>{check(user.data?.['authentication.api'])}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

{#if editing}
  <Modal title={editing.id === '-' ? 'New user' : 'Edit user'} onclose={() => (editing = null)}>
    <FormRow label="Username">
      <input type="text" bind:value={editing.draft.username} autocomplete="off" />
    </FormRow>
    <FormRow label="New password">
      <input type="password" bind:value={editing.draft.password} autocomplete="new-password" />
    </FormRow>
    <FormRow label="Confirm password">
      <input type="password" bind:value={editing.draft.confirm} autocomplete="new-password" />
    </FormRow>
    <FormRow label="Web access">
      <input type="checkbox" bind:checked={editing.draft.web} />
    </FormRow>
    <FormRow label="PMS / DVR access">
      <input type="checkbox" bind:checked={editing.draft.pms} />
    </FormRow>
    <FormRow label="M3U access">
      <input type="checkbox" bind:checked={editing.draft.m3u} />
    </FormRow>
    <FormRow label="XML access">
      <input type="checkbox" bind:checked={editing.draft.xml} />
    </FormRow>
    <FormRow label="API access">
      <input type="checkbox" bind:checked={editing.draft.api} />
    </FormRow>

    {#if error}
      <p style="color: var(--danger)">{error}</p>
    {/if}

    {#snippet footer()}
      {#if editing && editing.id !== '-'}
        <button class="danger" onclick={() => save(true)}>Delete</button>
      {/if}
      <button onclick={() => (editing = null)}>Cancel</button>
      <button class="primary" onclick={() => save(false)}>Save</button>
    {/snippet}
  </Modal>
{/if}
