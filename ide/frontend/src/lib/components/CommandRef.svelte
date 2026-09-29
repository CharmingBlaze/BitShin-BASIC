<script lang="ts">
  import { Search, Copy, Plus, Check } from 'lucide-svelte';
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

<div class="ide-sidebar">
  <div class="ide-side-head">
    <span>Commands</span>
    <span class="ide-count">{filteredCommands.length}</span>
  </div>

  <div class="ide-search">
    <Search size={12} />
    <input type="text" bind:value={searchQuery} placeholder="Search commands…" />
  </div>
  <div style="padding:0 10px 8px;">
    <select bind:value={selectedCategory} class="ide-field">
      {#each categories as cat}
        <option value={cat}>{cat}</option>
      {/each}
    </select>
  </div>

  <div class="flex-1 overflow-y-auto min-h-0">
    {#if filteredCommands.length === 0}
      <div class="ide-quiet">No commands match that filter.</div>
    {:else}
      {#each filteredCommands as cmd}
        <button
          onclick={() => selectedCommand = cmd}
          class="ide-list-row flex-col items-start"
          class:active={selectedCommand?.name === cmd.name}
        >
          <span class="font-mono ide-row-name">{cmd.name}</span>
          <span class="truncate font-mono ide-row-meta">{cmd.syntax}</span>
        </button>
      {/each}
    {/if}
  </div>

  {#if selectedCommand}
    <div class="ide-detail">
      <div class="flex items-center justify-between gap-2">
        <div>
          <div class="font-mono" style="font-size:12px;color:var(--accent-bright);">{selectedCommand.name}</div>
          <div class="ide-row-meta">{selectedCommand.category}</div>
        </div>
        <div class="flex items-center gap-1">
          <button class="ide-iconbtn" onclick={() => copySyntax(selectedCommand?.syntax || '')} title="Copy">
            {#if copied}<Check size={12} />{:else}<Copy size={12} />{/if}
          </button>
          <button class="ide-btn primary" style="height:22px;padding:0 8px;font-size:10px;" onclick={() => insertCommand(selectedCommand!)}>
            <Plus size={11} /> Insert
          </button>
        </div>
      </div>
      <div class="ide-codeblock">{selectedCommand.syntax}</div>
      <div class="ide-prose">{selectedCommand.description}</div>
    </div>
  {/if}
</div>
