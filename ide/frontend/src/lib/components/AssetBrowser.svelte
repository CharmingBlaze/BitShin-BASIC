<script lang="ts">
  import { Box, Image as ImageIcon, Music, Code, FileText, Copy, Check, Search } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
  import type { FileNode } from '../types';

  let searchQuery = $state('');
  let copiedPath = $state<string | null>(null);

  interface AssetItem {
    name: string;
    path: string;
    relPath: string;
    ext: string;
    category: 'model' | 'image' | 'audio' | 'shader' | 'data' | 'other';
    snippet: string;
  }

  function categorize(name: string, relPath: string): AssetItem | null {
    const ext = name.split('.').pop()?.toLowerCase() || '';
    const normRel = relPath.replace(/\\/g, '/');

    if (['glb', 'gltf', 'obj', 'dae', 'b3d'].includes(ext)) {
      return {
        name,
        path: relPath,
        relPath: normRel,
        ext,
        category: 'model',
        snippet: `mesh = LoadMesh("${normRel}")`
      };
    }
    if (['png', 'jpg', 'jpeg', 'bmp', 'tga'].includes(ext)) {
      return {
        name,
        path: relPath,
        relPath: normRel,
        ext,
        category: 'image',
        snippet: `tex = LoadTexture("${normRel}")`
      };
    }
    if (['wav', 'ogg', 'mp3'].includes(ext)) {
      return {
        name,
        path: relPath,
        relPath: normRel,
        ext,
        category: 'audio',
        snippet: `snd = LoadSound("${normRel}")`
      };
    }
    if (['glsl', 'vert', 'frag'].includes(ext)) {
      return {
        name,
        path: relPath,
        relPath: normRel,
        ext,
        category: 'shader',
        snippet: `shader = LoadShader("${normRel}")`
      };
    }
    if (['json', 'yaml', 'geojson', 'tmx'].includes(ext)) {
      return {
        name,
        path: relPath,
        relPath: normRel,
        ext,
        category: 'data',
        snippet: `data = JSONLoad("${normRel}")`
      };
    }
    return null;
  }

  function collectAssets(node: FileNode | null, parentPath = ''): AssetItem[] {
    if (!node) return [];
    const results: AssetItem[] = [];
    const currentRel = parentPath ? `${parentPath}/${node.name}` : (node.isDir ? '' : node.name);

    if (!node.isDir) {
      const item = categorize(node.name, currentRel || node.name);
      if (item) results.push(item);
    }

    if (node.children) {
      for (const child of node.children) {
        results.push(...collectAssets(child, currentRel));
      }
    }
    return results;
  }

  let allAssets = $derived(collectAssets(editorStore.projectTree));

  let filteredAssets = $derived(
    allAssets.filter(a => {
      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase();
      return a.name.toLowerCase().includes(q) || a.relPath.toLowerCase().includes(q) || a.category.includes(q);
    })
  );

  async function copySnippet(item: AssetItem) {
    try {
      await navigator.clipboard.writeText(item.snippet);
      copiedPath = item.relPath;
      setTimeout(() => {
        if (copiedPath === item.relPath) copiedPath = null;
      }, 1800);
    } catch (e) {
      console.error('Clipboard copy failed', e);
    }
  }
</script>

<div class="ide-sidebar">
  <div class="ide-side-head">
    <span>Assets</span>
    <span class="ide-count">{filteredAssets.length}</span>
  </div>

  <div class="ide-search">
    <Search size={12} />
    <input type="text" placeholder="Filter models, textures, audio" bind:value={searchQuery} />
  </div>

  <div class="flex-1 min-h-0 overflow-y-auto">
    {#if filteredAssets.length === 0}
      <div class="ide-quiet">
        {#if allAssets.length === 0}
          No media assets in this folder. Models, images, and sounds show up here.
        {:else}
          No assets match that filter.
        {/if}
      </div>
    {:else}
      {#each filteredAssets as item}
        <div class="ide-list-row ide-asset">
          <span class="flex items-center gap-2 min-w-0">
            {#if item.category === 'model'}
              <Box size={14} class="ide-cat model" />
            {:else if item.category === 'image'}
              <ImageIcon size={14} class="ide-cat image" />
            {:else if item.category === 'audio'}
              <Music size={14} class="ide-cat audio" />
            {:else if item.category === 'shader'}
              <Code size={14} class="ide-cat shader" />
            {:else}
              <FileText size={14} class="ide-cat data" />
            {/if}
            <span class="min-w-0">
              <span class="font-mono ide-row-name truncate" style="display:block;font-size:11px;">{item.name}</span>
              <span class="font-mono ide-row-file truncate" style="display:block;">{item.relPath}</span>
            </span>
          </span>
          <button class="ide-btn" style="height:22px;padding:0 8px;font-size:10px;" onclick={() => copySnippet(item)} title="Copy load snippet">
            {#if copiedPath === item.relPath}
              <Check size={11} /> Copied
            {:else}
              <Copy size={11} /> Copy
            {/if}
          </button>
        </div>
      {/each}
    {/if}
  </div>
</div>
