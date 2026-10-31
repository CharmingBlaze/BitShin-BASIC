<script lang="ts">
  import { Clapperboard, FilePlus, RefreshCw } from 'lucide-svelte';
  import { editorStore, isSceneFile, baseName, samePath } from '../stores/editorState.svelte';

  // One entity row, nested under its parent.
  type Row = { id: number; kind: string; name: string; pos: string; depth: number };

  // The active tab when it is JSON, else the scene last shown or edited.
  let source = $derived.by(() => {
    const tab = editorStore.activeTab;
    if (tab && tab.name.toLowerCase().endsWith('.json')) {
      return { path: tab.path, name: tab.name, content: tab.content };
    }
    const s = editorStore.sceneFile;
    return s ? { path: s.path, name: baseName(s.path), content: s.content } : null;
  });
  let parsed = $derived(readScene(source?.content ?? ''));
  let note = $derived(sceneNote());
  let editing = $derived(!!source && editorStore.sceneEditors.some((f) => samePath(f, source!.path)));

  function sceneNote(): string {
    if (!source) return 'Open a .scene.json file, or double-click one in the Explorer to edit it in the scene editor.';
    if (!source.content.trim()) return 'This scene file is empty.';
    if (parsed.error) return `Not a scene: ${parsed.error}`;
    return '';
  }

  function num(v: any): string {
    const n = Number(v);
    return Number.isFinite(n) ? String(Math.round(n * 100) / 100) : '';
  }

  // position: [x,y,z] (v2), transform.position, or flat x/y/z (SaveScene).
  function posOf(item: any): string {
    const p = Array.isArray(item.position) ? item.position : Array.isArray(item.transform?.position) ? item.transform.position : null;
    if (p) return p.map(num).join(', ');
    return [item.x, item.y, item.z].filter((n) => n !== undefined).map(num).join(', ');
  }

  function readScene(content: string): { version: number; rows: Row[]; error: string } {
    if (!content.trim()) return { version: 0, rows: [], error: '' };
    try {
      const data = JSON.parse(content);
      const list: any[] = Array.isArray(data) ? data : Array.isArray(data?.entities) ? data.entities : [];
      const version = Number(data?.version) || 1;
      const ids = new Set(list.map((it) => Number(it?.id)).filter((n) => n));
      const children = new Map<number, any[]>();
      for (const item of list) {
        const parent = Number(item?.parent) || 0;
        const key = ids.has(parent) ? parent : 0;
        if (!children.has(key)) children.set(key, []);
        children.get(key)!.push(item);
      }
      const rows: Row[] = [];
      const seen = new Set<any>();
      const walk = (parent: number, depth: number) => {
        for (const item of children.get(parent) ?? []) {
          if (seen.has(item)) continue;
          seen.add(item);
          rows.push({
            id: Number(item.id) || 0,
            kind: String(item.kind || item.type || 'entity'),
            name: String(item.name || item.src || ''),
            pos: posOf(item),
            depth
          });
          const id = Number(item.id) || 0;
          if (id && depth < 32) walk(id, depth + 1);
        }
      };
      walk(0, 0);
      return { version, rows, error: '' };
    } catch (err: any) {
      return { version: 0, rows: [], error: err.message || String(err) };
    }
  }
</script>

<div class="ide-sidebar">
  <div class="ide-side-head">
    <span>Scene</span>
    <div class="flex items-center">
      <button class="ide-iconbtn" onclick={() => (editorStore.showNewScene = true)} title="New scene">
        <FilePlus size={13} />
      </button>
      {#if source && editorStore.sceneFile && source.path === editorStore.sceneFile.path}
        <button class="ide-iconbtn" onclick={() => editorStore.showScene(source!.path)} title="Reload from disk">
          <RefreshCw size={13} />
        </button>
      {/if}
    </div>
  </div>
  {#if source}
    <div class="sp-file">
      <span class="truncate" title={source.path}>{source.name}{parsed.version ? ` · v${parsed.version}` : ''}</span>
      {#if isSceneFile(source.path) && !source.path.includes('://')}
        <button class="ide-btn" onclick={() => editorStore.editScene(source!.path)} title="Open in the scene editor (moon edit)">
          <Clapperboard size={12} /> {editing ? 'Focus editor' : 'Edit Scene'}
        </button>
      {/if}
    </div>
  {/if}
  {#if note}
    <p class="ide-side-note">{note}</p>
  {:else if parsed.rows.length === 0}
    <p class="ide-side-note">No entities in this scene yet.</p>
  {:else}
    <ul class="ide-scene-list">
      {#each parsed.rows as row, i (i)}
        <li style="padding-left: {12 + row.depth * 14}px;">
          <div>
            <strong>{row.name || row.kind}</strong>
            {#if row.name}<span class="muted"> {row.kind}</span>{/if}
          </div>
          {#if row.pos}<span class="muted">{row.pos}</span>{/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .sp-file {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 6px 12px;
    font-size: 12px;
    border-bottom: 1px solid var(--border);
  }
  .ide-side-note {
    margin: 0;
    padding: 10px 12px;
    color: var(--text-dim, #9aa0a6);
    font-size: 12px;
    line-height: 1.45;
  }
  .ide-scene-list {
    list-style: none;
    margin: 0;
    padding: 4px 0 12px;
    overflow-y: auto;
  }
  .ide-scene-list li {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 5px 12px;
    font-size: 12px;
  }
  .muted {
    opacity: 0.7;
  }
</style>
