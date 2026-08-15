<script lang="ts">
  import { X, Settings, Sliders, Type, Check } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  const themes = [
    { id: 'bitshin-dark', name: 'BitShin Dark (Default OLED)' },
    { id: 'blitz-classic', name: 'Blitz3D Classic Blue' },
    { id: 'cyberpunk', name: 'Cyberpunk Neon' },
    { id: 'vs-dark', name: 'VS Dark' }
  ];

  function save() {
    editorStore.saveSettings();
    editorStore.showSettingsModal = false;
  }
</script>

<div class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50 select-none animate-in fade-in duration-150">
  <div class="bg-slate-900 border border-slate-700/80 rounded-xl shadow-2xl w-full max-w-lg overflow-hidden flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-slate-800 bg-slate-950/60">
      <div class="flex items-center gap-2">
        <div class="p-1.5 rounded-lg bg-sky-500/20 text-sky-400">
          <Settings size={18} />
        </div>
        <div>
          <h2 class="text-sm font-bold text-slate-100">IDE Preferences</h2>
          <p class="text-xs text-slate-400">Configure editor theme, typography, and behavior</p>
        </div>
      </div>
      <button
        onclick={() => editorStore.showSettingsModal = false}
        class="p-1.5 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
      >
        <X size={16} />
      </button>
    </div>

    <!-- Body -->
    <div class="p-5 space-y-4 text-xs">
      <!-- Theme -->
      <div>
        <label class="block font-medium text-slate-300 mb-1">Editor Theme</label>
        <select
          bind:value={editorStore.settings.theme}
          class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-slate-200 focus:outline-none focus:border-sky-500 cursor-pointer"
        >
          {#each themes as t}
            <option value={t.id}>{t.name}</option>
          {/each}
        </select>
      </div>

      <!-- Font Size & Tab Size -->
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="block font-medium text-slate-300 mb-1">Font Size (px)</label>
          <input
            type="number"
            min="10"
            max="32"
            bind:value={editorStore.settings.fontSize}
            class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-slate-200 focus:outline-none focus:border-sky-500"
          />
        </div>

        <div>
          <label class="block font-medium text-slate-300 mb-1">Tab Size</label>
          <select
            bind:value={editorStore.settings.tabSize}
            class="w-full bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-slate-200 focus:outline-none focus:border-sky-500 cursor-pointer"
          >
            <option value={2}>2 Spaces</option>
            <option value={4}>4 Spaces</option>
            <option value={8}>8 Spaces</option>
          </select>
        </div>
      </div>

      <!-- Toggles -->
      <div class="space-y-2 pt-2 border-t border-slate-800">
        <label class="flex items-center justify-between p-2 rounded-lg bg-slate-950/60 border border-slate-800 cursor-pointer hover:border-slate-700">
          <span class="text-slate-300 font-medium">Show Minimap</span>
          <input
            type="checkbox"
            bind:checked={editorStore.settings.minimap}
            class="w-4 h-4 rounded text-sky-600 focus:ring-sky-500 border-slate-700 bg-slate-900 cursor-pointer"
          />
        </label>

        <label class="flex items-center justify-between p-2 rounded-lg bg-slate-950/60 border border-slate-800 cursor-pointer hover:border-slate-700">
          <span class="text-slate-300 font-medium">Word Wrap</span>
          <select
            bind:value={editorStore.settings.wordWrap}
            class="bg-slate-900 border border-slate-700 rounded px-2 py-1 text-slate-200 text-xs cursor-pointer"
          >
            <option value="off">Off</option>
            <option value="on">On</option>
            <option value="wordWrapColumn">Column Wrap</option>
          </select>
        </label>
      </div>
    </div>

    <!-- Footer -->
    <div class="px-5 py-3 border-t border-slate-800 bg-slate-950/60 flex items-center justify-end gap-2">
      <button
        onclick={save}
        class="flex items-center gap-1.5 px-4 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white font-medium text-xs transition shadow-sm"
      >
        <Check size={13} />
        <span>Save Settings</span>
      </button>
    </div>
  </div>
</div>
