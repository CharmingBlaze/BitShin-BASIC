<script lang="ts">
  import { Code2, Braces, Tag, Bookmark, ChevronRight } from 'lucide-svelte';
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

<div class="symbol-outline flex flex-col h-full bg-slate-900 border-r border-slate-800 select-none">
  <!-- Header -->
  <div class="p-3 border-b border-slate-800 flex items-center justify-between bg-slate-950/40">
    <div class="flex items-center gap-1.5 text-xs font-bold text-sky-400">
      <Code2 size={14} />
      <span>Document Outline</span>
    </div>
    <span class="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">
      {editorStore.symbols.length} symbols
    </span>
  </div>

  <!-- Symbol List -->
  <div class="flex-1 overflow-y-auto p-2 space-y-1 scrollbar-thin">
    {#if editorStore.symbols.length === 0}
      <div class="p-6 text-center text-slate-500 text-xs">
        No functions or types declared in active document.
      </div>
    {:else}
      {#each editorStore.symbols as sym}
        <button
          onclick={() => jumpToLine(sym.line)}
          class="w-full text-left px-2.5 py-1.5 rounded hover:bg-slate-800/80 flex items-center justify-between text-xs transition group"
        >
          <div class="flex items-center gap-2 truncate">
            {#if sym.type === 'function'}
              <div class="w-4 h-4 rounded bg-indigo-500/20 text-indigo-400 flex items-center justify-center text-[10px] font-bold">
                f
              </div>
              <span class="font-mono text-slate-200 truncate">{sym.signature || sym.name}</span>
            {:else if sym.type === 'type'}
              <div class="w-4 h-4 rounded bg-emerald-500/20 text-emerald-400 flex items-center justify-center text-[10px] font-bold">
                T
              </div>
              <span class="font-mono text-emerald-300 truncate">Type {sym.name}</span>
            {:else if sym.type === 'const'}
              <div class="w-4 h-4 rounded bg-amber-500/20 text-amber-400 flex items-center justify-center text-[10px] font-bold">
                C
              </div>
              <span class="font-mono text-amber-300 truncate">Const {sym.name}</span>
            {:else}
              <Bookmark size={13} class="text-rose-400" />
              <span class="font-mono text-rose-300 truncate">.{sym.name}</span>
            {/if}
          </div>

          <span class="text-[10px] font-mono text-slate-500 group-hover:text-sky-400 pl-2">
            :{sym.line}
          </span>
        </button>
      {/each}
    {/if}
  </div>
</div>
