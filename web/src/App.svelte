<script lang="ts">
  import { onMount } from 'svelte'
  import { server, needsLogin, loading, toast, refresh, pollLog, clientInfo, settings } from './lib/stores'
  import Login from './pages/Login.svelte'
  import Wizard from './pages/Wizard.svelte'
  import Playlist from './pages/Playlist.svelte'
  import Xmltv from './pages/Xmltv.svelte'
  import Filters from './pages/Filters.svelte'
  import Mapping from './pages/Mapping.svelte'
  import Users from './pages/Users.svelte'
  import SettingsPage from './pages/SettingsPage.svelte'
  import LogPage from './pages/LogPage.svelte'
  import { logout } from './lib/api'

  type PageKey = 'playlist' | 'xmltv' | 'filter' | 'mapping' | 'users' | 'settings' | 'log'

  let page: PageKey = $state('playlist')

  const nav: { key: PageKey; label: string; icon: string }[] = [
    { key: 'playlist', label: 'Playlist', icon: '≣' },
    { key: 'xmltv', label: 'XMLTV', icon: '𝄪' },
    { key: 'filter', label: 'Filter', icon: '⧩' },
    { key: 'mapping', label: 'Mapping', icon: '⇄' },
    { key: 'users', label: 'Users', icon: '☺' },
    { key: 'settings', label: 'Settings', icon: '⚙' },
    { key: 'log', label: 'Log', icon: '≡' },
  ]

  let authEnabled = $derived($settings?.['authentication.web'] === true)
  let visibleNav = $derived(nav.filter((n) => n.key !== 'users' || authEnabled))

  onMount(() => {
    refresh()
    const timer = setInterval(() => {
      if ($server && !$needsLogin && !$server.configurationWizard) pollLog()
    }, 10000)
    return () => clearInterval(timer)
  })

  function connBadge(active: number, total: number): string {
    if (!total) return 'ok'
    const ratio = active / total
    if (ratio >= 0.8) return 'err'
    if (ratio >= 0.6) return 'warn'
    return 'ok'
  }
</script>

{#if $needsLogin}
  <Login />
{:else if !$server}
  <div class="boot">
    <img src="/web/img/threadfin.png" alt="" width="64" height="64" />
    <p class="muted">Connecting to Threadfin…</p>
  </div>
{:else if $server.configurationWizard}
  <Wizard />
{:else}
  <div class="layout">
    <aside class="sidebar">
      <div class="brand">
        <img src="/web/img/threadfin.png" alt="Threadfin" width="28" height="28" />
        <span>Threadfin</span>
      </div>
      <nav>
        {#each visibleNav as item (item.key)}
          <button class="nav-item" class:active={page === item.key} onclick={() => (page = item.key)}>
            <span class="icon">{item.icon}</span>{item.label}
          </button>
        {/each}
      </nav>
      <div class="sidebar-foot">
        {#if authEnabled}
          <button class="nav-item" onclick={logout}><span class="icon">⏻</span>Logout</button>
        {/if}
        <div class="version muted">v{$clientInfo?.version ?? '…'}</div>
      </div>
    </aside>

    <div class="main">
      <header class="topbar">
        <div class="stats">
          <span class="badge">Streams {$clientInfo?.streams ?? '–'}</span>
          <span class="badge">XEPG {$clientInfo?.xepg ?? 0}</span>
          <span class="badge {connBadge($clientInfo?.activePlaylist ?? 0, $clientInfo?.totalPlaylist ?? 0)}">
            Playlist {$clientInfo?.activePlaylist ?? 0}/{$clientInfo?.totalPlaylist ?? 0}
          </span>
          <span class="badge {connBadge($clientInfo?.activeClients ?? 0, $clientInfo?.totalClients ?? 0)}">
            Clients {$clientInfo?.activeClients ?? 0}/{$clientInfo?.totalClients ?? 0}
          </span>
          {#if ($clientInfo?.errors ?? 0) > 0}
            <span class="badge err">Errors {$clientInfo?.errors}</span>
          {/if}
          {#if ($clientInfo?.warnings ?? 0) > 0}
            <span class="badge warn">Warnings {$clientInfo?.warnings}</span>
          {/if}
        </div>
        {#if $loading}
          <div class="spinner" title="Working…"></div>
        {/if}
      </header>

      <main class="content">
        {#if page === 'playlist'}
          <Playlist />
        {:else if page === 'xmltv'}
          <Xmltv />
        {:else if page === 'filter'}
          <Filters />
        {:else if page === 'mapping'}
          <Mapping />
        {:else if page === 'users'}
          <Users />
        {:else if page === 'settings'}
          <SettingsPage />
        {:else if page === 'log'}
          <LogPage />
        {/if}
      </main>
    </div>
  </div>
{/if}

{#if $toast}
  <div class="toast {$toast.kind}">{$toast.text}</div>
{/if}

<style>
  .boot {
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
  }

  .layout {
    display: flex;
    height: 100%;
  }

  .sidebar {
    width: 200px;
    flex-shrink: 0;
    background: var(--bg-panel);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    padding: 14px 10px;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 17px;
    font-weight: 700;
    padding: 4px 10px 16px;
  }

  nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 9px 12px;
    border: none;
    background: none;
    border-radius: var(--radius);
    color: var(--text-dim);
    text-align: left;
    font-weight: 500;
  }

  .nav-item .icon {
    width: 18px;
    text-align: center;
  }

  .nav-item:hover {
    background: var(--bg-hover);
    color: var(--text);
  }

  .nav-item.active {
    background: var(--bg-hover);
    color: var(--accent);
  }

  .sidebar-foot {
    margin-top: 10px;
  }

  .version {
    padding: 8px 12px 0;
    font-size: 12px;
  }

  .main {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .topbar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 20px;
    border-bottom: 1px solid var(--border);
    min-height: 48px;
  }

  .stats {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-left: auto;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }

  .content {
    flex: 1;
    overflow-y: auto;
    padding: 20px;
  }

  .toast {
    position: fixed;
    bottom: 20px;
    right: 20px;
    max-width: 420px;
    padding: 12px 16px;
    border-radius: var(--radius);
    background: var(--bg-panel);
    border: 1px solid var(--border);
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
    white-space: pre-line;
    z-index: 200;
  }

  .toast.error {
    border-color: var(--danger);
  }

  .toast.success {
    border-color: var(--success);
  }

  @media (max-width: 720px) {
    .layout {
      flex-direction: column;
    }

    .sidebar {
      width: 100%;
      flex-direction: row;
      align-items: center;
      border-right: none;
      border-bottom: 1px solid var(--border);
      overflow-x: auto;
      padding: 8px;
    }

    .brand {
      padding: 0 10px;
    }

    .brand span {
      display: none;
    }

    nav {
      flex-direction: row;
    }

    .sidebar-foot {
      margin-top: 0;
      display: flex;
      align-items: center;
    }

    .version {
      display: none;
    }
  }
</style>
