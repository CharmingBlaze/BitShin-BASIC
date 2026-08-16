<script lang="ts">
  import { X } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  const themes = [
    { id: 'bitshin-dark', name: 'BitShin Dark' },
    { id: 'blitz-classic', name: 'Blitz3D Classic' },
    { id: 'cyberpunk', name: 'Cyberpunk' },
    { id: 'vs-dark', name: 'VS Dark' }
  ];

  function save() {
    editorStore.saveSettings();
    editorStore.showSettingsModal = false;
  }
</script>

<div class="ide-backdrop">
  <div class="ide-modal sm">
    <div class="ide-modal-head">
      <div>
        <h2>Settings</h2>
        <p>Theme, font, and editor layout</p>
      </div>
      <button class="ide-iconbtn" onclick={() => editorStore.showSettingsModal = false}><X size={15} /></button>
    </div>
    <div class="ide-modal-body space-y-4 text-xs">
      <div>
        <label for="theme-select" style="display:block;margin-bottom:6px;color:var(--text-secondary);">Editor theme</label>
        <select id="theme-select" class="ide-field" bind:value={editorStore.settings.theme}>
          {#each themes as t}
            <option value={t.id}>{t.name}</option>
          {/each}
        </select>
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label for="font-size-input" style="display:block;margin-bottom:6px;color:var(--text-secondary);">Font size</label>
          <input id="font-size-input" class="ide-field" type="number" min="10" max="32" bind:value={editorStore.settings.fontSize} />
        </div>
        <div>
          <label for="tab-size-select" style="display:block;margin-bottom:6px;color:var(--text-secondary);">Tab size</label>
          <select id="tab-size-select" class="ide-field" bind:value={editorStore.settings.tabSize}>
            <option value={2}>2</option>
            <option value={4}>4</option>
            <option value={8}>8</option>
          </select>
        </div>
      </div>
      <label class="flex items-center justify-between" style="padding:8px;border:1px solid var(--border);">
        <span>Show minimap</span>
        <input type="checkbox" bind:checked={editorStore.settings.minimap} />
      </label>
      <div>
        <label for="wrap-select" style="display:block;margin-bottom:6px;color:var(--text-secondary);">Word wrap</label>
        <select id="wrap-select" class="ide-field" bind:value={editorStore.settings.wordWrap}>
          <option value="off">Off</option>
          <option value="on">On</option>
          <option value="wordWrapColumn">Column</option>
        </select>
      </div>
    </div>
    <div class="ide-modal-foot">
      <button class="ide-btn primary" onclick={save}>Save</button>
    </div>
  </div>
</div>
