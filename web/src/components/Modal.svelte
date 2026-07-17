<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    title,
    onclose,
    wide = false,
    children,
    footer,
  }: {
    title: string
    onclose: () => void
    wide?: boolean
    children: Snippet
    footer?: Snippet
  } = $props()

  function onBackdrop(e: MouseEvent) {
    if (e.target === e.currentTarget) onclose()
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={onKeydown} />

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="backdrop" onclick={onBackdrop}>
  <div class="modal" class:wide role="dialog" aria-modal="true" aria-label={title}>
    <div class="head">
      <h3>{title}</h3>
      <button class="close" onclick={onclose} aria-label="Close">&times;</button>
    </div>
    <div class="body">
      {@render children()}
    </div>
    {#if footer}
      <div class="foot">
        {@render footer()}
      </div>
    {/if}
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding: 6vh 16px 16px;
    z-index: 100;
    overflow-y: auto;
  }

  .modal {
    background: var(--bg-panel);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    width: 100%;
    max-width: 560px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  }

  .modal.wide {
    max-width: 760px;
  }

  .head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
    border-bottom: 1px solid var(--border);
  }

  .head h3 {
    margin: 0;
    font-size: 16px;
  }

  .close {
    border: none;
    background: none;
    font-size: 22px;
    line-height: 1;
    padding: 0 4px;
    color: var(--text-dim);
  }

  .close:hover {
    color: var(--text);
    background: none;
  }

  .body {
    padding: 18px;
    max-height: 70vh;
    overflow-y: auto;
  }

  .foot {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    padding: 14px 18px;
    border-top: 1px solid var(--border);
    flex-wrap: wrap;
  }
</style>
