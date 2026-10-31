<script lang="ts">
  import { Play, Redo2, ArrowDownToLine, ArrowUpFromLine, Square, Plus, X, ChevronDown, ChevronRight, Bug } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
  import type { DebugVar } from '../types';

  let newWatch = $state('');
  let open = $state({ stack: true, locals: true, globals: true, watch: true });
  let selectedFrame = $state(0);

  let st = $derived(editorStore.debugState);
  let paused = $derived(editorStore.debugPaused);

  $effect(() => {
    // A new pause selects the innermost frame again.
    const _ = editorStore.debugState;
    selectedFrame = 0;
  });

  function jump(i: number, line: number) {
    selectedFrame = i;
    editorStore.revealLine(line);
  }

  function submitWatch(e: Event) {
    e.preventDefault();
    editorStore.addWatch(newWatch);
    newWatch = '';
  }

  function watchVar(v: DebugVar) {
    editorStore.addWatch(v.name);
  }
</script>

<div class="ide-sidebar">
  <div class="ide-side-head">
    <span>Debug</span>
    {#if paused}
      <span class="ide-badge err">Paused</span>
    {:else if editorStore.isRunning && editorStore.debugSession}
      <span class="ide-badge run">Running</span>
    {/if}
  </div>

  <div class="ide-dbg-tools">
    <button class="ide-iconbtn" title="Continue (F5)" disabled={!paused} onclick={() => editorStore.debugContinue()}>
      <Play size={13} />
    </button>
    <button class="ide-iconbtn" title="Step Over (F10)" disabled={!paused} onclick={() => editorStore.debugStepOver()}>
      <Redo2 size={13} />
    </button>
    <button class="ide-iconbtn" title="Step Into (F11)" disabled={!paused} onclick={() => editorStore.debugStepInto()}>
      <ArrowDownToLine size={13} />
    </button>
    <button class="ide-iconbtn" title="Step Out (Shift+F11)" disabled={!paused} onclick={() => editorStore.debugStepOut()}>
      <ArrowUpFromLine size={13} />
    </button>
    <button class="ide-iconbtn" title="Stop (Shift+F5)" disabled={!editorStore.isRunning} onclick={() => editorStore.stopProgram()}>
      <Square size={12} />
    </button>
    {#if !editorStore.isRunning}
      <button class="ide-btn" style="margin-left:auto;" title="Debug (Ctrl+F5)" onclick={() => editorStore.runActiveProgram(true)}>
        <Bug size={12} /> Debug
      </button>
    {/if}
  </div>

  <div class="flex-1 overflow-y-auto">
    {#if !st}
      <div class="ide-quiet" style="text-align:left;">
        Press Ctrl+F5 to debug. Set breakpoints with F9 or by clicking the gutter, or call DebugBreak in the script.
      </div>
    {/if}

    <button class="ide-dbg-section" onclick={() => (open.stack = !open.stack)}>
      {#if open.stack}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
      Call Stack
    </button>
    {#if open.stack && st}
      <div class:ide-dbg-stale={!paused}>
        {#each st.stack as f, i}
          <button class="ide-list-row justify-between {i === selectedFrame ? 'active' : ''}" onclick={() => jump(i, f.line)}>
            <span class="font-mono truncate" style="font-size:11px;">{f.name}</span>
            <span class="font-mono ide-row-file">{f.file}:{f.line}</span>
          </button>
        {/each}
      </div>
    {/if}

    <button class="ide-dbg-section" onclick={() => (open.locals = !open.locals)}>
      {#if open.locals}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
      Locals{st ? ` (${st.locals.length})` : ''}
    </button>
    {#if open.locals && st}
      <div class:ide-dbg-stale={!paused}>
        {#if st.locals.length === 0}
          <div class="ide-dbg-empty">{st.depth > 1 ? 'No locals.' : 'At the top level: see Globals.'}</div>
        {/if}
        {#each st.locals as v}
          <button class="ide-dbg-var" title="{v.kind} · double-click to watch" ondblclick={() => watchVar(v)}>
            <span class="ide-dbg-name">{v.name}</span>
            <span class="ide-dbg-val">{v.value}</span>
          </button>
        {/each}
      </div>
    {/if}

    <button class="ide-dbg-section" onclick={() => (open.globals = !open.globals)}>
      {#if open.globals}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
      Globals{st ? ` (${st.globals.length})` : ''}
    </button>
    {#if open.globals && st}
      <div class:ide-dbg-stale={!paused}>
        {#if st.globals.length === 0}
          <div class="ide-dbg-empty">No globals yet.</div>
        {/if}
        {#each st.globals as v}
          <button class="ide-dbg-var" title="{v.kind} · double-click to watch" ondblclick={() => watchVar(v)}>
            <span class="ide-dbg-name">{v.name}</span>
            <span class="ide-dbg-val">{v.value}</span>
          </button>
        {/each}
      </div>
    {/if}

    <button class="ide-dbg-section" onclick={() => (open.watch = !open.watch)}>
      {#if open.watch}<ChevronDown size={12} />{:else}<ChevronRight size={12} />{/if}
      Watch{editorStore.watches.length ? ` (${editorStore.watches.length})` : ''}
    </button>
    {#if open.watch}
      <form class="ide-dbg-watch-add" onsubmit={submitWatch}>
        <input class="ide-field font-mono" placeholder="Expression, e.g. player.x + 1" bind:value={newWatch} />
        <button class="ide-iconbtn" type="submit" title="Add watch"><Plus size={13} /></button>
      </form>
      <div class:ide-dbg-stale={!paused}>
        {#each editorStore.watches as w (w.expr)}
          <div class="ide-dbg-var">
            <span class="ide-dbg-name">{w.expr}</span>
            {#if w.error}
              <span class="ide-dbg-val ide-dbg-err">{w.error}</span>
            {:else}
              <span class="ide-dbg-val">{w.value ?? (paused ? '…' : 'not paused')}</span>
            {/if}
            <button class="ide-iconbtn ide-dbg-x" title="Remove watch" onclick={() => editorStore.removeWatch(w.expr)}>
              <X size={11} />
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
