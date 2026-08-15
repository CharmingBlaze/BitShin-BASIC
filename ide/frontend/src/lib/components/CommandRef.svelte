<script lang="ts">
  import { Search, Copy, Plus, BookOpen, Check } from 'lucide-svelte';
  import commandsData from '../data/commands.json';
  import type { CommandItem } from '../types';
  import { editorStore } from '../stores/editorState.svelte';

  let searchQuery = $state('');
  let selectedCategory = $state('All');
  let selectedCommand = $state<CommandItem | null>(commandsData.commands[0] || null);
  let copied = $state(false);

  const categories = ['All', ...commandsData.categories.map(c => c.name)];

  let filteredCommands = $derived.by(() => {
    let list = commandsData.commands;
    if (selectedCategory !== 'All') {
      list = list.filter(c => c.category === selectedCategory);
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(c =>
        c.name.toLowerCase().includes(q) ||
        c.syntax.toLowerCase().includes(q) ||
        c.description.toLowerCase().includes(q)
      );
    }
    return list;
  });

  function insertCommand(cmd: CommandItem) {
    if (!editorStore.activeTab) return;
    editorStore.updateActiveContent(editorStore.activeTab.content + '\n' + cmd.syntax);
  }

  function copySyntax(syntax: string) {
    navigator.clipboard.writeText(syntax);
    copied = true;
    setTimeout(() => copied = false, 2000);
  }
</script>

<div class="command-ref flex flex-col h-full bg-slate-900/90 border-r border-slate-800 select-none">
  <!-- Header & Search -->
  <div class="p-3 border-b border-slate-800 space-y-2 bg-slate-950/40">
    <div class="flex items-center justify-between">
      <span class="text-[10px] font-bold tracking-wider uppercase text-slate-400">
        Command Reference
      </span>
      <span class="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">
        {filteredCommands.length} / {commandsData.commands.length}
      </span>
    </div>

    <!-- Search box -->
    <div class="relative">
      <Search size={13} class="absolute left-2.5 top-2.5 text-slate-500" />
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Search commands (Camera, Physics)..."
        class="w-full bg-slate-950 border border-slate-800 rounded-md pl-8 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-sky-500 transition"
      />
    </div>

    <!-- Category filter selector -->
    <div class="px-2 py-1 rounded bg-slate-950 border border-slate-800">
      <select
        bind:value={selectedCategory}
        class="w-full bg-transparent text-[11px] text-slate-300 focus:outline-none cursor-pointer"
      >
        {#each categories as cat}
          <option value={cat}>{cat}</option>
        {/each}
      </select>
    </div>
  </div>

  <!-- Main Content: Split List + Detail Preview -->
  <div class="flex-1 flex flex-col min-h-0">
    <!-- Command List -->
    <div class="flex-1 overflow-y-auto divide-y divide-slate-800/40 scrollbar-thin">
      {#if filteredCommands.length === 0}
        <div class="p-6 text-center text-slate-500 text-xs">
          No commands matching "{searchQuery}"
        </div>
      {:else}
        {#each filteredCommands as cmd}
          <button
            onclick={() => selectedCommand = cmd}
            class="w-full text-left px-3 py-1.5 text-xs flex flex-col gap-0.5 hover:bg-slate-800/60 transition cursor-pointer border-l-2 {selectedCommand?.name === cmd.name ? 'bg-sky-500/10 border-sky-400' : 'border-transparent'}"
          >
            <div class="flex items-center justify-between">
              <span class="font-mono font-medium text-slate-200 text-[12px]">{cmd.name}</span>
              <span class="text-[9px] px-1.5 py-0.2 rounded bg-slate-800 text-slate-400 truncate max-w-[120px]">
                {cmd.category}
              </span>
            </div>
            <div class="text-[10px] text-slate-400 font-mono truncate">{cmd.syntax}</div>
          </button>
        {/each}
      {/if}
    </div>

    <!-- Selected Command Detail Card -->
    {#if selectedCommand}
      <div class="p-3 bg-slate-950 border-t border-slate-800 flex flex-col gap-2 max-h-56 overflow-y-auto">
        <div class="flex items-start justify-between gap-2">
          <div>
            <div class="text-xs font-bold text-sky-300 font-mono">{selectedCommand.name}</div>
            <div class="text-[10px] text-slate-400">{selectedCommand.category}</div>
          </div>
          <div class="flex items-center gap-1">
            <button
              onclick={() => copySyntax(selectedCommand?.syntax || '')}
              class="p-1 rounded bg-slate-800 hover:bg-slate-700 text-slate-300 transition"
              title="Copy Syntax"
            >
              {#if copied}
                <Check size={12} class="text-emerald-400" />
              {:else}
                <Copy size={12} />
              {/if}
            </button>
            <button
              onclick={() => insertCommand(selectedCommand!)}
              class="flex items-center gap-1 px-2 py-1 rounded bg-sky-600 hover:bg-sky-500 text-white text-[10px] font-medium transition"
              title="Insert into code"
            >
              <Plus size={11} />
              <span>Insert</span>
            </button>
          </div>
        </div>

        <!-- Syntax Box -->
        <div class="p-2 rounded bg-slate-900 border border-slate-800 text-[11px] font-mono text-amber-300 break-words select-text">
          {selectedCommand.syntax}
        </div>

        <!-- Description -->
        <div class="text-xs text-slate-300 leading-relaxed select-text">
          {selectedCommand.description}
        </div>
      </div>
    {/if}
  </div>
</div>
