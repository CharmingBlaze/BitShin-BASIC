<script lang="ts">
  import { Bookmark } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  function jumpToLine(line: number) {
    const editor = (window as any).__monacoEditor;
    if (editor) {
      editor.revealLineInCenter(line);
      editor.setPosition({ lineNumber: line, column: 1 });
      editor.focus();
    }
  }
</script>

<div class="ide-sidebar">
  <div class="ide-side-head">
    <span>Outline</span>
    <span class="ide-count">{editorStore.symbols.length}</span>
  </div>

  <div class="flex-1 overflow-y-auto py-1">
    {#if editorStore.symbols.length === 0}
      <div class="ide-quiet">No functions or types in the active file.</div>
    {:else}
      {#each editorStore.symbols as sym}
        <button class="ide-list-row justify-between" onclick={() => jumpToLine(sym.line)}>
          <span class="flex items-center gap-2 truncate">
            {#if sym.type === 'function'}
              <span class="ide-mark ide-glyph fn">f</span>
              <span class="font-mono truncate" style="font-size:11px;">{sym.signature || sym.name}</span>
            {:else if sym.type === 'type'}
              <span class="ide-mark ide-glyph ty">T</span>
              <span class="font-mono truncate" style="font-size:11px;">Type {sym.name}</span>
            {:else if sym.type === 'const'}
              <span class="ide-mark ide-glyph">C</span>
              <span class="font-mono truncate" style="font-size:11px;">Const {sym.name}</span>
            {:else}
              <Bookmark size={12} style="color: var(--err-text);" />
              <span class="font-mono truncate" style="font-size:11px;">.{sym.name}</span>
            {/if}
          </span>
          <span class="font-mono ide-row-file">:{sym.line}</span>
        </button>
      {/each}
    {/if}
  </div>
</div>
