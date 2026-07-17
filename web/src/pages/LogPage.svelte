<script lang="ts">
  import { log, send } from '../lib/stores'

  let lines = $derived($log?.log ?? [])

  function lineClass(line: string): string {
    if (line.includes('[ERROR]')) return 'err'
    if (line.includes('[WARNING]')) return 'warn'
    if (line.includes('[DEBUG]')) return 'debug'
    return ''
  }

  async function reset() {
    await send('resetLogs')
  }
</script>

<div class="toolbar">
  <h2 style="margin:0">Log</h2>
  {#if ($log?.errors ?? 0) > 0}
    <span class="badge err">{$log?.errors} errors</span>
  {/if}
  {#if ($log?.warnings ?? 0) > 0}
    <span class="badge warn">{$log?.warnings} warnings</span>
  {/if}
  <div class="spacer"></div>
  <button onclick={reset}>Clear log</button>
</div>

<div class="panel log">
  {#if lines.length === 0}
    <p class="muted">Log is empty.</p>
  {:else}
    {#each lines as line, i (i)}
      <div class="line {lineClass(line)}">{line}</div>
    {/each}
  {/if}
</div>

<style>
  .log {
    font-family: ui-monospace, 'Cascadia Code', 'Source Code Pro', Menlo, monospace;
    font-size: 12.5px;
    line-height: 1.7;
    overflow-x: auto;
  }

  .line {
    white-space: pre-wrap;
    word-break: break-all;
  }

  .line.err {
    color: var(--danger);
  }

  .line.warn {
    color: var(--warning);
  }

  .line.debug {
    color: var(--text-dim);
  }
</style>
