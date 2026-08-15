<script lang="ts">
  import { Sparkles, Search, Play, FolderCode, FileText } from 'lucide-svelte';
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

  function openExample(filename: string) {
    editorStore.loadExample(filename);
  }
</script>

<div class="examples-gallery flex flex-col h-full bg-slate-900 border-r border-slate-800 select-none">
  <!-- Header & Search -->
  <div class="p-3 border-b border-slate-800 space-y-2 bg-slate-950/40">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-1.5 text-xs font-bold text-sky-400">
        <Sparkles size={14} />
        <span>Sample Demos & Games</span>
      </div>
      <span class="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">
        {filteredExamples.length} / {examplesData.length}
      </span>
    </div>

    <!-- Search box -->
    <div class="relative">
      <Search size={13} class="absolute left-2.5 top-2.5 text-slate-500" />
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Filter 69 demos (e.g. 3D, physics, terrain, 2D)..."
        class="w-full bg-slate-900 border border-slate-700/80 rounded-md pl-8 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-sky-500 transition"
      />
    </div>
  </div>

  <!-- Examples List -->
  <div class="flex-1 overflow-y-auto p-2 space-y-2 scrollbar-thin">
    {#if filteredExamples.length === 0}
      <div class="p-6 text-center text-slate-500 text-xs">
        No examples found matching "{searchQuery}"
      </div>
    {:else}
      {#each filteredExamples as ex}
        <div class="p-2.5 rounded-lg bg-slate-950/80 border border-slate-800/80 hover:border-sky-500/50 hover:bg-slate-950 transition flex flex-col gap-1.5 group">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-1.5">
              <FileText size={13} class="text-sky-400" />
              <span class="text-xs font-semibold text-slate-200">{ex.name}</span>
            </div>
            <span class="text-[10px] font-mono text-slate-500">{ex.filename}</span>
          </div>

          {#if ex.description}
            <p class="text-[11px] text-slate-400 line-clamp-2 leading-relaxed">
              {ex.description}
            </p>
          {/if}

          <div class="flex items-center justify-end pt-1">
            <button
              onclick={() => openExample(ex.filename)}
              class="flex items-center gap-1 px-2.5 py-1 rounded bg-sky-600/90 hover:bg-sky-500 text-white text-[10px] font-medium transition shadow-sm group-hover:scale-105 active:scale-95"
            >
              <FolderCode size={11} />
              <span>Open in Editor</span>
            </button>
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>
