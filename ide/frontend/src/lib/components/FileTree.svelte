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
      return { icon: FileCode, color: 'var(--accent)' };
    }
    if (lower.endsWith('.png') || lower.endsWith('.jpg') || lower.endsWith('.jpeg') || lower.endsWith('.bmp')) {
      return { icon: Image, color: 'var(--ok-text)' };
    }
    if (lower.endsWith('.exe') || lower.endsWith('.dll') || lower.endsWith('.so') || lower.endsWith('.dylib')) {
      return { icon: Binary, color: 'var(--info)' };
    }
    if (lower.endsWith('.json') || lower.endsWith('.yml') || lower.endsWith('.yaml') || lower.endsWith('.toml')) {
      return { icon: FileSpreadsheet, color: 'var(--warn)' };
    }
    return { icon: FileText, color: 'var(--text-muted)' };
  }
</script>

<div class="ide-sidebar">
  <div class="ide-side-head">
    <span>Explorer</span>
    <div class="flex items-center">
      <button class="ide-iconbtn" onclick={() => editorStore.showNewModal = true} title="New file">
        <FilePlus size={13} />
      </button>
      <button class="ide-iconbtn" onclick={() => editorStore.refreshProjectTree()} title="Refresh">
        <RefreshCw size={13} />
      </button>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto py-1">
    {#if !editorStore.projectTree}
      <div class="p-6 text-center text-xs" style="color: var(--text-muted);">Loading workspace…</div>
    {:else}
      {#snippet renderNode(node: FileNode, depth: number)}
        {@const fileMeta = getFileIcon(node.name)}
        {@const Icon = fileMeta.icon}
        {@const isCurrentTab = editorStore.activeTab?.path === node.path}

        <div class="flex flex-col">
          <button
            onclick={() => openFileNode(node)}
            style="padding-left: {depth * 12 + 8}px;"
            class="ide-tree-row {isCurrentTab ? 'active' : ''}"
          >
            {#if node.isDir}
              <span style="width:12px;color:var(--text-dim);display:inline-flex;">
                {#if expandedPaths[node.path]}
                  <ChevronDown size={11} />
                {:else}
                  <ChevronRight size={11} />
                {/if}
              </span>
              {#if expandedPaths[node.path]}
                <FolderOpen size={13} style="color: var(--accent); flex-shrink: 0;" />
              {:else}
                <Folder size={13} style="color: var(--accent); flex-shrink: 0;" />
              {/if}
              <span class="truncate">{node.name}</span>
            {:else}
              <span style="width:12px;"></span>
              <Icon size={13} style="color: {fileMeta.color}; flex-shrink: 0;" />
              <span class="truncate">{node.name}</span>
            {/if}
          </button>

          {#if node.isDir && expandedPaths[node.path] && node.children}
            {#each node.children as child}
              {@render renderNode(child, depth + 1)}
            {/each}
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
