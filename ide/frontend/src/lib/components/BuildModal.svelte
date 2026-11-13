<script lang="ts">
  import { X, Hammer, Loader2 } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
  import { AppAPI } from '../wailsBridge';

  let targetOS = $state(editorStore.settings.targetOS || 'windows');
  // With a project open, Build packages its main script into moon.json's build.outputDir.
  const project = editorStore.project;
  let outputDir = $state(project ? project.outputPath : 'dist');
  let isBuilding = $state(false);
  let buildResult = $state<{ success: boolean; message: string } | null>(null);

  async function startBuild() {
    if (!project && !editorStore.activeTab) return;
    isBuilding = true;
    buildResult = null;

    try {
      let file = project?.mainPath ?? '';
      if (!project && editorStore.activeTab) {
        if (editorStore.activeTab.isTemporary) {
          await editorStore.saveCurrentFile();
        }
        file = editorStore.activeTab.path;
      }
      const output = await AppAPI.buildExecutable(file, outputDir, targetOS);
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
        <h2>Build standalone game</h2>
        <p>Makes a folder with your game's .exe and files. Zip it and share it: players need nothing else installed.</p>
      </div>
      <button class="ide-iconbtn" onclick={() => editorStore.showBuildModal = false}><X size={15} /></button>
    </div>
    <div class="ide-modal-body space-y-4 text-xs">
      {#if project}
        <div class="ide-badge ide-note">Project {project.project.name}: builds {project.project.main} (version {project.project.build.version}{project.project.build.company ? `, ${project.project.build.company}` : ''})</div>
      {/if}
      <div>
        <div class="ide-label">Target platform</div>
        <div class="grid grid-cols-3 gap-2">
          {#each ['windows', 'linux', 'darwin'] as os}
            <button
              class="ide-btn {targetOS === os ? 'ide-choice is-on' : ''}"
              onclick={() => targetOS = os}
            >
              {os === 'darwin' ? 'macOS' : (os === 'linux' ? 'Linux' : 'Windows')}
            </button>
          {/each}
        </div>
        {#if targetOS !== 'windows'}
          <p style="opacity: 0.7; margin-top: 6px">Build for Linux or macOS on that system, with moonBASIC for it.</p>
        {/if}
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
