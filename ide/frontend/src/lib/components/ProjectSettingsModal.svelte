<script lang="ts">
  import { X, Loader2 } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
  import type { Project } from '../types';

  // Edit a copy; Save writes moon.json and the backend fills defaults.
  let p = $state<Project>(JSON.parse(JSON.stringify(editorStore.project!.project)));
  let busy = $state(false);
  let error = $state('');

  function close() {
    editorStore.showProjectSettings = false;
  }

  async function save() {
    busy = true;
    error = '';
    try {
      p.window.width = Number(p.window.width) || 0;
      p.window.height = Number(p.window.height) || 0;
      await editorStore.saveProject(p);
      close();
    } catch (err: any) {
      error = err.message || String(err);
    } finally {
      busy = false;
    }
  }
</script>

<div class="ide-backdrop">
  <div class="ide-modal md">
    <div class="ide-modal-head">
      <div>
        <h2>Project settings</h2>
        <p class="font-mono">{editorStore.project?.file}</p>
      </div>
      <button class="ide-iconbtn" onclick={close}><X size={15} /></button>
    </div>
    <div class="ide-modal-body space-y-4 text-xs">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="ps-name" class="ide-label">Name</label>
          <input id="ps-name" class="ide-field" bind:value={p.name} />
        </div>
        <div>
          <label for="ps-main" class="ide-label">Main script (F5 runs it)</label>
          <input id="ps-main" class="ide-field font-mono" bind:value={p.main} placeholder="main.mb" />
        </div>
      </div>

      <div>
        <div class="ide-label">Window</div>
        <div class="grid grid-cols-3 gap-4">
          <input class="ide-field" type="number" min="160" max="7680" bind:value={p.window.width} title="Width" aria-label="Window width" />
          <input class="ide-field" type="number" min="120" max="4320" bind:value={p.window.height} title="Height" aria-label="Window height" />
          <select class="ide-field" bind:value={p.quality} title="SetGraphicsQuality tier" aria-label="Graphics quality">
            {#each ['auto', 'low', 'medium', 'high', 'ultra'] as q}
              <option value={q}>Quality: {q}</option>
            {/each}
          </select>
        </div>
        <input class="ide-field" style="margin-top: 8px;" bind:value={p.window.title} placeholder="Window title" aria-label="Window title" />
      </div>

      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="ps-scenes" class="ide-label">Scenes folder</label>
          <input id="ps-scenes" class="ide-field font-mono" bind:value={p.scenes} placeholder="scenes" />
        </div>
        <div>
          <label for="ps-assets" class="ide-label">Assets folder</label>
          <input id="ps-assets" class="ide-field font-mono" bind:value={p.assets} placeholder="assets" />
        </div>
      </div>

      <div>
        <div class="ide-label">Build</div>
        <div class="grid grid-cols-2 gap-4">
          <input class="ide-field font-mono" bind:value={p.build.outputDir} placeholder="dist" title="Output folder" aria-label="Build output folder" />
          <input class="ide-field font-mono" bind:value={p.build.icon} placeholder="assets/icon.png" title="Icon" aria-label="Build icon" />
          <input class="ide-field" bind:value={p.build.version} placeholder="1.0.0" title="Version" aria-label="Build version" />
          <input class="ide-field" bind:value={p.build.company} placeholder="Company" title="Company" aria-label="Build company" />
        </div>
        <p class="ps-note">The Build dialog uses the output folder. Icon, version, and company are saved for packaging; moon build does not stamp them into the binary yet. The window values are what new templates write into main.mb; the engine does not read moon.json.</p>
      </div>

      {#if error}
        <div class="ide-badge ide-note err">{error}</div>
      {/if}
    </div>
    <div class="ide-modal-foot">
      <button class="ide-btn" onclick={close}>Cancel</button>
      <button class="ide-btn primary" onclick={save} disabled={busy}>
        {#if busy}<Loader2 size={13} class="animate-spin" />{/if} Save
      </button>
    </div>
  </div>
</div>

<style>
  .ps-note {
    margin-top: 6px;
    color: var(--text-muted);
    line-height: 1.45;
  }
</style>
