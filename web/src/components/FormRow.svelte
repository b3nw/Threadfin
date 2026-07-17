<script lang="ts">
  import type { Snippet } from 'svelte'

  let {
    label,
    hint,
    children,
  }: {
    label: string
    hint?: string
    children: Snippet
  } = $props()
</script>

<div class="row">
  <div class="label">{label}</div>
  <div class="field">
    {@render children()}
    {#if hint}
      <div class="hint">{hint}</div>
    {/if}
  </div>
</div>

<style>
  .row {
    display: grid;
    grid-template-columns: 180px 1fr;
    gap: 12px;
    align-items: start;
    margin-bottom: 12px;
  }

  .label {
    color: var(--text-dim);
    padding-top: 7px;
    font-weight: 500;
  }

  .hint {
    color: var(--text-dim);
    font-size: 12px;
    margin-top: 4px;
    white-space: pre-line;
  }

  @media (max-width: 600px) {
    .row {
      grid-template-columns: 1fr;
      gap: 4px;
    }

    .label {
      padding-top: 0;
    }
  }
</style>
