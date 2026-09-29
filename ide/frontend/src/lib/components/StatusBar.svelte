<script lang="ts">
  import { Terminal } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
</script>

<footer class="ide-status">
  <div class="ide-status-group">
    <span class="ide-lang">BitShin BASIC</span>
    {#if editorStore.activeTab}
      <span class="ide-status-sep"></span>
      <span title={editorStore.activeTab.path}>{editorStore.activeTab.name}</span>
    {/if}
    <span class="ide-status-sep"></span>
    <span>Ln {editorStore.cursorPos.line}, Col {editorStore.cursorPos.col}</span>
    <button onclick={() => editorStore.toggleOutputPanel()} title="Toggle output (Ctrl+`)">
      <Terminal size={11} />
      Output
    </button>
  </div>

  <div class="ide-status-group">
    {#if editorStore.isRunning}
      <span class="is-warn">Running</span>
      <span class="ide-status-sep"></span>
    {:else if editorStore.exitCode === 0}
      <span class="is-ok">Ready</span>
      <span class="ide-status-sep"></span>
    {/if}
    <span class="ide-lsp" title={editorStore.lspConnected ? 'Language server connected' : 'Language server disconnected'}>
      <span class="ide-lsp-dot {editorStore.lspConnected ? 'on' : ''}"></span>
      {editorStore.lspConnected ? 'LSP connected' : 'LSP disconnected'}
    </span>
    <span class="ide-status-sep"></span>
    <span>UTF-8</span>
    <span>Spaces: {editorStore.settings.tabSize}</span>
  </div>
</footer>
