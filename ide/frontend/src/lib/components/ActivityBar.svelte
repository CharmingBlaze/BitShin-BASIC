<script lang="ts">
  import { FolderTree, BookOpen, ListTree, Library, Settings } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  const items = [
    { id: 'files' as const, icon: FolderTree, title: 'Explorer' },
    { id: 'commands' as const, icon: BookOpen, title: 'Command reference' },
    { id: 'outline' as const, icon: ListTree, title: 'Outline' },
    { id: 'examples' as const, icon: Library, title: 'Examples' }
  ];

  function activate(id: typeof items[number]['id']) {
    if (editorStore.activeSidebarTab === id && !editorStore.sidebarCollapsed) {
      editorStore.sidebarCollapsed = true;
      return;
    }
    editorStore.activeSidebarTab = id;
    editorStore.sidebarCollapsed = false;
  }
</script>

<nav class="ide-activity" aria-label="Activity">
  {#each items as item}
    {@const Icon = item.icon}
    <button
      class="ide-activity-btn {editorStore.activeSidebarTab === item.id && !editorStore.sidebarCollapsed ? 'active' : ''}"
      title={item.title}
      onclick={() => activate(item.id)}
    >
      <Icon size={18} />
    </button>
  {/each}
  <div class="ide-activity-spacer"></div>
  <button
    class="ide-activity-btn"
    title="Settings"
    onclick={() => editorStore.showSettingsModal = true}
  >
    <Settings size={18} />
  </button>
</nav>
