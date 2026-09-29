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
        <h2>Compile native executable</h2>
        <p>Compile Blitz source code directly to a native machine-code binary (AOT) or Web export</p>
      </div>
      <button class="ide-iconbtn" onclick={() => editorStore.showBuildModal = false}><X size={15} /></button>
    </div>
    <div class="ide-modal-body space-y-4 text-xs">
      <div>
        <div class="ide-label">Target platform</div>
        <div class="grid grid-cols-4 gap-2">
          {#each ['windows', 'linux', 'darwin', 'wasm'] as os}
            <button
              class="ide-btn {targetOS === os ? 'ide-choice is-on' : ''}"
              onclick={() => targetOS = os}
            >
              {os === 'darwin' ? 'macOS' : (os === 'wasm' ? 'Web (WASM)' : os)}
            </button>
          {/each}
        </div>
      </div>
      <div>
        <label for="out-dir" class="ide-label">Output folder</label>
        <input id="out-dir" class="ide-field font-mono" bind:value={outputDir} placeholder="dist" />
      </div>
      {#if buildResult}
        <div class="ide-badge ide-note {buildResult.success ? 'ok' : 'err'}">
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
