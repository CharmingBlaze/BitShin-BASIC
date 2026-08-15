<script lang="ts">
  import { Code2, Bookmark } from 'lucide-svelte';
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

<div class="symbol-outline flex flex-col h-full bg-slate-900/90 border-r border-slate-800 select-none">
  <!-- Header -->
  <div class="px-3 py-2 border-b border-slate-800 flex items-center justify-between bg-slate-950/40">
    <span class="text-[10px] font-bold tracking-wider uppercase text-slate-400">
      Document Outline
    </span>
    <span class="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-400 font-mono">
      {editorStore.symbols.length}
    </span>
  </div>

  <!-- Symbol List -->
  <div class="flex-1 overflow-y-auto py-1 space-y-0.5 scrollbar-thin">
    {#if editorStore.symbols.length === 0}
      <div class="p-6 text-center text-slate-500 text-xs">
        No functions or types declared in active document.
      </div>
    {:else}
      {#each editorStore.symbols as sym}
        <button
          onclick={() => jumpToLine(sym.line)}
          class="w-full text-left px-3 py-1 hover:bg-slate-800/60 flex items-center justify-between text-xs transition group cursor-pointer"
        >
          <div class="flex items-center gap-2 truncate">
            {#if sym.type === 'function'}
              <div class="w-4 h-4 rounded bg-indigo-500/20 text-indigo-400 flex items-center justify-center text-[10px] font-bold shrink-0">
                f
              </div>
              <span class="font-mono text-slate-200 truncate text-[11px]">{sym.signature || sym.name}</span>
            {:else if sym.type === 'type'}
              <div class="w-4 h-4 rounded bg-emerald-500/20 text-emerald-400 flex items-center justify-center text-[10px] font-bold shrink-0">
                T
              </div>
              <span class="font-mono text-emerald-300 truncate text-[11px]">Type {sym.name}</span>
            {:else if sym.type === 'const'}
              <div class="w-4 h-4 rounded bg-amber-500/20 text-amber-400 flex items-center justify-center text-[10px] font-bold shrink-0">
                C
              </div>
              <span class="font-mono text-amber-300 truncate text-[11px]">Const {sym.name}</span>
            {:else}
              <Bookmark size={13} class="text-rose-400 shrink-0" />
              <span class="font-mono text-rose-300 truncate text-[11px]">.{sym.name}</span>
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
