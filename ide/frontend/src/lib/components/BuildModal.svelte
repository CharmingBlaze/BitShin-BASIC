<script lang="ts">
  import { X, Hammer, Loader2 } from 'lucide-svelte';
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
      if (editorStore.activeTab.isTemporary) {
        await editorStore.saveCurrentFile();
      }
      const output = await AppAPI.buildExecutable(editorStore.activeTab.path, outputDir, targetOS);
      buildResult = {
        success: true,
        message: output || `Built standalone bundle to '${outputDir}'.`
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

<div class="ide-backdrop">
  <div class="ide-modal md">
    <div class="ide-modal-head">
      <div>
        <h2>Build package</h2>
        <p>Create a distribution folder with the executable and natives</p>
      </div>
      <button class="ide-iconbtn" onclick={() => editorStore.showBuildModal = false}><X size={15} /></button>
    </div>
    <div class="ide-modal-body space-y-4 text-xs">
      <div>
        <div style="margin-bottom:6px;color:var(--text-secondary);">Target platform</div>
        <div class="grid grid-cols-3 gap-2">
          {#each ['windows', 'linux', 'darwin'] as os}
            <button
              class="ide-btn"
              style={targetOS === os ? 'border-color:#6d5818;color:var(--accent-bright);background:var(--accent-dim);' : ''}
              onclick={() => targetOS = os}
            >
              {os === 'darwin' ? 'macOS' : os}
            </button>
          {/each}
        </div>
      </div>
      <div>
        <label for="out-dir" style="display:block;margin-bottom:6px;color:var(--text-secondary);">Output folder</label>
        <input id="out-dir" class="ide-field font-mono" bind:value={outputDir} placeholder="dist" />
      </div>
      {#if buildResult}
        <div class="ide-badge {buildResult.success ? 'ok' : 'err'}" style="height:auto;padding:8px;white-space:pre-wrap;font-size:11px;">
          {buildResult.message}
        </div>
      {/if}
    </div>
    <div class="ide-modal-foot">
      <button class="ide-btn" onclick={() => editorStore.showBuildModal = false}>Close</button>
      <button class="ide-btn primary" onclick={startBuild} disabled={isBuilding}>
        {#if isBuilding}
          <Loader2 size={13} class="animate-spin" /> Building…
        {:else}
          <Hammer size={13} /> Build
        {/if}
      </button>
    </div>
  </div>
</div>
