<script lang="ts">
  import { X, Hammer, CheckCircle2, AlertCircle, Loader2, FolderCheck } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
  import { AppAPI } from '../wailsBridge';

  let targetOS = $state(editorStore.settings.targetOS || 'windows');
  let outputDir = $state('dist');
  let isBuilding = $state(false);
  let buildResult = $state<{ success: boolean; message: string } | null>(null);

  async function startBuild() {
    if (!editorStore.activeTab) return;
    isBuilding = true;
    buildResult = null;

    try {
      // If temporary file, save first
      if (editorStore.activeTab.isTemporary) {
        await editorStore.saveCurrentFile();
      }
      const output = await AppAPI.buildExecutable(editorStore.activeTab.path, outputDir, targetOS);
      buildResult = {
        success: true,
        message: output || `Successfully built standalone bundle to '${outputDir}'. DLLs and assets packaged!`
      };
    } catch (err: any) {
      buildResult = {
        success: false,
        message: err.message || String(err)
      };
    } finally {
      isBuilding = false;
    }
  }
</script>

<div class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50 select-none animate-in fade-in duration-150">
  <div class="bg-slate-900 border border-slate-700/80 rounded-xl shadow-2xl w-full max-w-xl overflow-hidden flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-slate-800 bg-slate-950/60">
      <div class="flex items-center gap-2">
        <div class="p-1.5 rounded-lg bg-indigo-500/20 text-indigo-400">
          <Hammer size={18} />
        </div>
        <div>
          <h2 class="text-sm font-bold text-slate-100">Package Standalone Game (F7)</h2>
          <p class="text-xs text-slate-400">Create a zip-ready distribution folder with binary and assets</p>
        </div>
      </div>
      <button
        onclick={() => editorStore.showBuildModal = false}
        class="p-1.5 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
      >
        <X size={16} />
      </button>
    </div>

    <!-- Body -->
    <div class="p-5 space-y-4 text-xs">
      <div>
        <label class="block font-medium text-slate-300 mb-1">Target Platform</label>
        <div class="grid grid-cols-3 gap-2">
          {#each ['windows', 'linux', 'darwin'] as os}
            <button
              onclick={() => targetOS = os}
              class="py-2 px-3 rounded-lg border text-center font-medium capitalize transition {targetOS === os ? 'bg-sky-600 text-white border-sky-500 shadow-sm' : 'bg-slate-950 text-slate-300 border-slate-800 hover:border-slate-700'}"
            >
              {os === 'darwin' ? 'macOS (Darwin)' : os}
            </button>
          {/each}
        </div>
      </div>

      <div>
        <label class="block font-medium text-slate-300 mb-1">Output Folder</label>
        <input
          type="text"
          bind:value={outputDir}
          class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-slate-200 font-mono focus:outline-none focus:border-sky-500"
          placeholder="dist"
        />
        <p class="text-[11px] text-slate-500 mt-1">Natives, shaders, and referenced assets will be auto-collected next to the executable.</p>
      </div>

      {#if buildResult}
        <div class="p-3 rounded-lg border {buildResult.success ? 'bg-emerald-950/40 border-emerald-800/80 text-emerald-300' : 'bg-rose-950/40 border-rose-800/80 text-rose-300'} flex items-start gap-2">
          {#if buildResult.success}
            <CheckCircle2 size={16} class="shrink-0 mt-0.5" />
          {:else}
            <AlertCircle size={16} class="shrink-0 mt-0.5" />
          {/if}
          <div class="text-[11px] font-mono whitespace-pre-wrap leading-relaxed">
            {buildResult.message}
          </div>
        </div>
      {/if}
    </div>

    <!-- Footer Actions -->
    <div class="px-5 py-3 border-t border-slate-800 bg-slate-950/60 flex items-center justify-end gap-2">
      <button
        onclick={() => editorStore.showBuildModal = false}
        class="px-3.5 py-1.5 rounded-lg hover:bg-slate-800 text-slate-300 text-xs transition"
      >
        Close
      </button>
      <button
        onclick={startBuild}
        disabled={isBuilding}
        class="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 disabled:bg-slate-800 text-white font-medium text-xs transition shadow-sm"
      >
        {#if isBuilding}
          <Loader2 size={13} class="animate-spin" />
          <span>Building...</span>
        {:else}
          <Hammer size={13} />
          <span>Build Package</span>
        {/if}
      </button>
    </div>
  </div>
</div>
