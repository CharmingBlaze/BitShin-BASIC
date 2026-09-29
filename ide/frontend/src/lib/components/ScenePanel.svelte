<script lang="ts">
  import { editorStore } from '../stores/editorState.svelte';

  type Row = { kind: string; name: string; pos: string };

  let rows = $derived(readScene(editorStore.activeTab?.content ?? '', editorStore.activeTab?.name ?? ''));
  let note = $derived(sceneNote(editorStore.activeTab?.name ?? '', editorStore.activeTab?.content ?? ''));

  function sceneNote(name: string, content: string): string {
    if (!name) return 'Open a scene JSON file to list entities.';
    if (!name.toLowerCase().endsWith('.json')) return 'The open file is not JSON. SaveScene writes a JSON list of entities.';
    if (!content.trim()) return 'This scene file is empty.';
    return '';
  }

  function readScene(content: string, name: string): Row[] {
    if (!name.toLowerCase().endsWith('.json') || !content.trim()) return [];
    try {
      const data = JSON.parse(content);
      const list = Array.isArray(data) ? data : Array.isArray(data?.entities) ? data.entities : [];
      return list.map((item: any) => ({
        kind: String(item.kind || item.type || 'entity'),
        name: String(item.name || item.src || ''),
        pos: [item.x, item.y, item.z].filter((n) => n !== undefined).join(', ')
      }));
    } catch {
      return [];
    }
  }
</script>

<div class="ide-sidebar">
  <div class="ide-side-head">Scene</div>
  {#if note}
    <p class="ide-side-note">{note}</p>
  {:else if rows.length === 0}
    <p class="ide-side-note">No entities in this JSON.</p>
  {:else}
    <ul class="ide-scene-list">
      {#each rows as row, i (i)}
        <li>
          <strong>{row.kind}</strong>
          {#if row.name}<span>{row.name}</span>{/if}
          {#if row.pos}<span class="muted">{row.pos}</span>{/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
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
  }
  .ide-scene-list li {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 6px 12px;
    font-size: 12px;
  }
  .muted {
    opacity: 0.7;
  }
</style>
