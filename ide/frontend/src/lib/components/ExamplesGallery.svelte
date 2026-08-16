<script lang="ts">
  import { Search, FileCode } from 'lucide-svelte';
  import examplesData from '../data/examples.json';
  import { editorStore } from '../stores/editorState.svelte';

  let searchQuery = $state('');

  let filteredExamples = $derived.by(() => {
    if (!searchQuery.trim()) return examplesData;
    const q = searchQuery.toLowerCase();
    return examplesData.filter(e =>
      e.name.toLowerCase().includes(q) ||
      e.filename.toLowerCase().includes(q) ||
      e.description.toLowerCase().includes(q)
    );
  });
</script>

<div class="ide-sidebar">
  <div class="ide-side-head">
    <span>Examples</span>
    <span class="font-mono" style="font-size:10px;letter-spacing:0;text-transform:none;">{filteredExamples.length}</span>
  </div>

  <div class="ide-search">
    <Search size={12} />
    <input type="text" bind:value={searchQuery} placeholder="Filter examples…" />
  </div>

  <div class="flex-1 overflow-y-auto">
    {#if filteredExamples.length === 0}
      <div class="p-6 text-center text-xs" style="color: var(--text-muted);">No examples match</div>
    {:else}
      {#each filteredExamples as ex}
        <button class="ide-list-row flex-col items-start" onclick={() => editorStore.loadExample(ex.filename)}>
          <span class="flex items-center gap-1.5">
            <FileCode size={12} style="color: var(--accent);" />
            <span style="color: var(--text-primary);">{ex.name}</span>
          </span>
          <span class="font-mono" style="font-size:10px;color:var(--text-dim);">{ex.filename}</span>
          {#if ex.description}
            <span style="font-size:11px;color:var(--text-muted);line-height:1.4;">{ex.description}</span>
          {/if}
        </button>
      {/each}
    {/if}
  </div>
</div>
