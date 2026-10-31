<script lang="ts">
  import { X, Clapperboard } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  let name = $state('level1');
  let openEditor = $state(true);
  let error = $state('');
  let busy = $state(false);
  const dir = editorStore.newSceneDir();

  function close() {
    editorStore.showNewScene = false;
  }

  async function create() {
    busy = true;
    error = '';
    try {
      await editorStore.createScene(name.trim(), openEditor);
      close();
    } catch (err: any) {
      error = err.message || String(err);
    } finally {
      busy = false;
    }
  }
</script>

<div class="ide-backdrop">
  <div class="ide-modal sm">
    <div class="ide-modal-head">
      <div>
        <h2>New scene</h2>
        <p>An empty version-2 .scene.json file</p>
      </div>
      <button class="ide-iconbtn" onclick={close}><X size={15} /></button>
    </div>
    <div class="ide-modal-body space-y-4 text-xs">
      <div>
        <label for="scene-name" class="ide-label">Scene name</label>
        <input
          id="scene-name"
          class="ide-field font-mono"
          bind:value={name}
          onkeydown={(e) => { if (e.key === 'Enter' && name.trim()) create(); }}
        />
        <p class="ns-path font-mono">{dir || '(workspace)'}{dir ? (dir.includes('\\') ? '\\' : '/') : ''}{name.replace(/\.scene\.json$|\.json$/i, '')}.scene.json</p>
      </div>
      <label class="ide-check">
        <span>Open it in the scene editor</span>
        <input type="checkbox" bind:checked={openEditor} />
      </label>
      {#if error}
        <div class="ide-badge ide-note err">{error}</div>
      {/if}
    </div>
    <div class="ide-modal-foot">
      <button class="ide-btn" onclick={close}>Cancel</button>
      <button class="ide-btn primary" onclick={create} disabled={busy || !name.trim()}><Clapperboard size={13} /> Create</button>
    </div>
  </div>
</div>

<style>
  .ns-path {
    margin-top: 6px;
    color: var(--text-muted);
    word-break: break-all;
  }
</style>
