<script lang="ts">
  import { editorStore } from '../stores/editorState.svelte';

  let prof = $derived(editorStore.profile);
</script>

<div class="ide-sidebar">
  <div class="ide-side-head">
    <span>Profiler</span>
  </div>
  <div class="flex-1 overflow-y-auto">
    {#if !prof}
      <div class="ide-quiet" style="text-align:left;">
        Run from the IDE. Function times and asset sizes appear here about once a second.
      </div>
    {:else}
      {#each prof.lines ?? [] as line}
        <div class="ide-quiet" style="text-align:left;">{line}</div>
      {/each}
      <div class="ide-side-head"><span>Functions</span></div>
      {#each prof.funcs ?? [] as fn}
        <div class="ide-quiet" style="text-align:left;">{fn.name} {fn.selfMs.toFixed(3)} ms × {fn.calls}</div>
      {:else}
        <div class="ide-quiet" style="text-align:left;">No function samples yet.</div>
      {/each}
      <div class="ide-side-head"><span>Assets</span></div>
      {#each prof.assets ?? [] as asset}
        <div class="ide-quiet" style="text-align:left;">{asset.name} {asset.kb.toFixed(1)} KB</div>
      {:else}
        <div class="ide-quiet" style="text-align:left;">No loaded assets yet.</div>
      {/each}
    {/if}
  </div>
</div>
