<script lang="ts">
  import { FolderTree, Folder, FolderOpen, FileCode, FileText, Image, RefreshCw, FilePlus, ChevronDown, ChevronRight } from 'lucide-svelte';
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
    if (lower.endsWith('.bb') || lower.endsWith('.basic')) return FileCode;
    if (lower.endsWith('.png') || lower.endsWith('.jpg') || lower.endsWith('.bmp')) return Image;
    return FileText;
  }
</script>

<div class="file-tree flex flex-col h-full bg-slate-900 border-r border-slate-800 select-none">
  <!-- Header -->
  <div class="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-950/40">
    <div class="flex items-center gap-1.5 text-xs font-bold text-sky-400">
      <FolderTree size={14} />
      <span>Project Explorer</span>
    </div>
    <div class="flex items-center gap-1">
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
        title="Refresh Tree"
      >
        <RefreshCw size={13} />
      </button>
    </div>
  </div>

  <!-- Tree View -->
  <div class="flex-1 overflow-y-auto p-2 scrollbar-thin">
    {#if !editorStore.projectTree}
      <div class="p-6 text-center text-slate-500 text-xs">
        Loading directory tree...
      </div>
    {:else}
      {#snippet renderNode(node: FileNode, depth: number)}
        {@const FileIcon = getFileIcon(node.name)}
        <div class="flex flex-col">
          <button
            onclick={() => openFileNode(node)}
            style="padding-left: {depth * 12 + 6}px;"
            class="w-full text-left py-1 pr-2 rounded hover:bg-slate-800/80 flex items-center gap-1.5 text-xs transition group {editorStore.activeTab?.path === node.path ? 'bg-sky-950 text-sky-300 font-medium' : 'text-slate-300'}"
          >
            {#if node.isDir}
              <span class="text-slate-500">
                {#if expandedPaths[node.path]}
                  <ChevronDown size={12} />
                {:else}
                  <ChevronRight size={12} />
                {/if}
              </span>
              {#if expandedPaths[node.path]}
                <FolderOpen size={14} class="text-sky-400 shrink-0" />
              {:else}
                <Folder size={14} class="text-sky-500 shrink-0" />
              {/if}
              <span class="truncate font-medium">{node.name}</span>
            {:else}
              <span class="w-3"></span>
              <FileIcon size={13} class="text-slate-400 shrink-0 group-hover:text-sky-400" />
              <span class="truncate">{node.name}</span>
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
