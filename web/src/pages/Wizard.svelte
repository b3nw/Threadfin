<script lang="ts">
  import { server, send } from '../lib/stores'

  // Mirrors the four wizard steps in the legacy UI (ts/configuration_ts.ts).
  // Each step submits one value; the server replies with the next step index
  // and reloads the page when the wizard is complete.
  let step = $derived($server?.wizard ?? 0)

  let tuner = $state(1)
  let epgSource = $state('XEPG')
  let m3u = $state('')
  let xmltv = $state('')
  let busy = $state(false)

  const steps = [
    {
      title: 'Number of tuners',
      description:
        'How many concurrent streams Threadfin should offer to clients (Plex, Jellyfin, Emby).',
    },
    {
      title: 'EPG source',
      description:
        'XEPG lets Threadfin manage the EPG and channel mapping (recommended). PMS delegates the EPG to Plex.',
    },
    {
      title: 'M3U playlist',
      description: 'URL or local file path of your provider M3U playlist.',
    },
    {
      title: 'XMLTV file',
      description: 'URL or local file path of the matching XMLTV guide data.',
    },
  ]

  async function next(e: SubmitEvent) {
    e.preventDefault()
    busy = true
    try {
      switch (step) {
        case 0:
          await send('saveWizard', { wizard: { tuner: Number(tuner) } })
          break
        case 1:
          await send('saveWizard', { wizard: { epgSource } })
          break
        case 2:
          if (!m3u.trim()) return
          await send('saveWizard', { wizard: { m3u: m3u.trim() } })
          break
        case 3:
          if (!xmltv.trim()) return
          await send('saveWizard', { wizard: { xmltv: xmltv.trim() } })
          break
      }
    } finally {
      busy = false
    }
  }
</script>

<div class="wrap">
  <form class="panel card" onsubmit={next}>
    <div class="logo">
      <img src="/web/img/threadfin.png" alt="" width="48" height="48" />
      <h2>Initial setup</h2>
    </div>

    <div class="progress">
      {#each steps as s, i (s.title)}
        <div class="dot" class:done={i < step} class:current={i === step}></div>
      {/each}
    </div>

    <h3>{steps[step]?.title}</h3>
    <p class="muted">{steps[step]?.description}</p>

    {#if step === 0}
      <select bind:value={tuner}>
        {#each Array.from({ length: 100 }, (_, i) => i + 1) as n (n)}
          <option value={n}>{n}</option>
        {/each}
      </select>
    {:else if step === 1}
      <select bind:value={epgSource}>
        <option value="XEPG">XEPG</option>
        <option value="PMS">PMS</option>
      </select>
    {:else if step === 2}
      <input type="text" bind:value={m3u} placeholder="https://provider.example/playlist.m3u" required />
    {:else if step === 3}
      <input type="text" bind:value={xmltv} placeholder="https://provider.example/guide.xml" required />
    {/if}

    <button class="primary" type="submit" disabled={busy}>
      {step === steps.length - 1 ? 'Finish' : 'Next'}
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
    max-width: 420px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .logo {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .logo h2,
  h3 {
    margin: 0;
  }

  p {
    margin: 0;
  }

  .progress {
    display: flex;
    gap: 8px;
    justify-content: center;
    margin: 6px 0;
  }

  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--border);
  }

  .dot.done {
    background: var(--accent);
    opacity: 0.5;
  }

  .dot.current {
    background: var(--accent);
  }
</style>
