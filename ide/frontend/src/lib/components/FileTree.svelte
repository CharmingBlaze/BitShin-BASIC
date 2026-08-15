<script lang="ts">
  import {
    Folder, FolderOpen, FileCode, FileText, Image, RefreshCw,
    FilePlus, ChevronDown, ChevronRight, Binary, FileSpreadsheet
  } from 'lucide-svelte';
  import type { FileNode } from '../types';
  import { editorStore } from '../stores/editorState.svelte';
  import { AppAPI } from '../wailsBridge';

  let expandedPaths = $state<Record<string, boolean>>({ '': true });

  function toggleExpand(path: string) {
    expandedPaths[path] = !expandedPaths[path];
  }

  async function openFileNode(node: FileNode) {
    if (node.isDir) {
      toggleExpand(node.path);
      return;
    }
    try {
      const res = await AppAPI.openFile(node.path);
      if (res) {
        editorStore.openTab({
          path: res.path,
          name: res.name,
          content: res.content
        });
      }
    } catch (err) {
      console.error('Failed to open file:', err);
    }
  }

  function getFileIcon(name: string) {
    const lower = name.toLowerCase();
    if (lower.endsWith('.bb') || lower.endsWith('.basic') || lower.endsWith('.b3d')) {
      return { icon: FileCode, color: 'text-sky-400' };
    }
    if (lower.endsWith('.png') || lower.endsWith('.jpg') || lower.endsWith('.jpeg') || lower.endsWith('.bmp')) {
      return { icon: Image, color: 'text-emerald-400' };
    }
    if (lower.endsWith('.exe') || lower.endsWith('.dll') || lower.endsWith('.so') || lower.endsWith('.dylib')) {
      return { icon: Binary, color: 'text-indigo-400' };
    }
    if (lower.endsWith('.json') || lower.endsWith('.yml') || lower.endsWith('.yaml') || lower.endsWith('.toml')) {
      return { icon: FileSpreadsheet, color: 'text-amber-400' };
    }
    return { icon: FileText, color: 'text-slate-400' };
  }
</script>

<div class="file-tree flex flex-col h-full bg-slate-900/90 border-r border-slate-800 select-none">
  <!-- Header -->
  <div class="px-3 py-2 border-b border-slate-800 flex items-center justify-between bg-slate-950/40">
    <span class="text-[10px] font-bold tracking-wider uppercase text-slate-400">
      Project Explorer
    </span>
    <div class="flex items-center gap-0.5">
      <button
        onclick={() => editorStore.showNewModal = true}
        class="p-1 rounded hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        title="New File"
      >
        <FilePlus size={13} />
      </button>
      <button
        onclick={() => editorStore.refreshProjectTree()}
        class="p-1 rounded hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
        title="Refresh Explorer"
      >
        <RefreshCw size={13} />
      </button>
    </div>
  </div>

  <!-- Tree View -->
  <div class="flex-1 overflow-y-auto py-1 scrollbar-thin">
    {#if !editorStore.projectTree}
      <div class="p-6 text-center text-slate-500 text-xs">
        Loading directory tree...
      </div>
    {:else}
      {#snippet renderNode(node: FileNode, depth: number)}
        {@const fileMeta = getFileIcon(node.name)}
        {@const Icon = fileMeta.icon}
        {@const isCurrentTab = editorStore.activeTab?.path === node.path}
        
        <div class="flex flex-col">
          <button
            onclick={() => openFileNode(node)}
            style="padding-left: {depth * 14 + 8}px;"
            class="w-full text-left py-1 pr-2 hover:bg-slate-800/60 flex items-center gap-1.5 text-xs transition group cursor-pointer border-l-2 {isCurrentTab ? 'bg-sky-500/10 text-sky-300 font-medium border-sky-400' : 'border-transparent text-slate-300 hover:text-slate-100'}"
          >
            {#if node.isDir}
              <span class="text-slate-500 flex items-center justify-center w-3 h-3">
                {#if expandedPaths[node.path]}
                  <ChevronDown size={11} />
                {:else}
                  <ChevronRight size={11} />
                {/if}
              </span>
              {#if expandedPaths[node.path]}
                <FolderOpen size={13} class="text-amber-400 shrink-0" />
              {:else}
                <Folder size={13} class="text-amber-400/90 shrink-0" />
              {/if}
              <span class="truncate text-[12px]">{node.name}</span>
            {:else}
              <span class="w-3"></span>
              <Icon size={13} class="{fileMeta.color} shrink-0" />
              <span class="truncate text-[12px]">{node.name}</span>
            {/if}
          </button>

          {#if node.isDir && expandedPaths[node.path] && node.children}
            <div class="flex flex-col">
              {#each node.children as child}
                {@render renderNode(child, depth + 1)}
              {/each}
            </div>
          {/if}
        </div>
      {/snippet}

      {#if editorStore.projectTree.children}
        {#each editorStore.projectTree.children as child}
          {@render renderNode(child, 0)}
        {/each}
      {/if}
    {/if}
  </div>
</div>
