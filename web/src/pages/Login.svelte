<script lang="ts">
  import { login } from '../lib/api'
  import { refresh } from '../lib/stores'

  let username = $state('')
  let password = $state('')
  let confirm = $state('')
  let firstUser = $state(false)
  let error = $state('')
  let busy = $state(false)

  async function submit(e: SubmitEvent) {
    e.preventDefault()
    error = ''

    if (firstUser && password !== confirm) {
      error = 'Passwords do not match.'
      return
    }

    busy = true
    try {
      const ok = await login(username, password, firstUser ? confirm : undefined)
      if (ok) {
        await refresh()
      } else {
        error = firstUser
          ? 'Could not create the user. It may already exist — try signing in.'
          : 'Sign-in failed. Check your username and password.'
      }
    } finally {
      busy = false
    }
  }
</script>

<div class="wrap">
  <form class="panel card" onsubmit={submit}>
    <div class="logo">
      <img src="/web/img/threadfin.png" alt="" width="48" height="48" />
      <h2>Threadfin</h2>
    </div>

    <label>
      Username
      <input type="text" bind:value={username} autocomplete="username" required />
    </label>

    <label>
      Password
      <input type="password" bind:value={password} autocomplete="current-password" required />
    </label>

    {#if firstUser}
      <label>
        Confirm password
        <input type="password" bind:value={confirm} autocomplete="new-password" required />
      </label>
    {/if}

    {#if error}
      <p class="error">{error}</p>
    {/if}

    <button class="primary" type="submit" disabled={busy}>
      {firstUser ? 'Create user' : 'Sign in'}
    </button>

    <button type="button" class="link" onclick={() => (firstUser = !firstUser)}>
      {firstUser ? 'Back to sign-in' : 'First run? Create the admin user'}
    </button>
  </form>
</div>

<style>
  .wrap {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
  }

  .card {
    width: 100%;
    max-width: 340px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .logo {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .logo h2 {
    margin: 0;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    color: var(--text-dim);
    font-weight: 500;
  }

  .error {
    color: var(--danger);
    margin: 0;
  }

  .link {
    border: none;
    background: none;
    color: var(--accent);
    padding: 0;
  }

  .link:hover {
    background: none;
    text-decoration: underline;
  }
</style>
