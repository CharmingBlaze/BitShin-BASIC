<script lang="ts">
  import { onMount } from 'svelte';
  import {
    X, FolderPlus, FolderOpen, History, Box, Square, Crosshair, PersonStanding,
    Footprints, Map as MapIcon, Trash2, Loader2, ArrowLeft
  } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';
  import { AppAPI } from '../wailsBridge';
  import type { ProjectTemplate, RecentProject } from '../types';

  const icons: Record<string, typeof Box> = {
    empty3d: Box,
    empty2d: Square,
    fps: Crosshair,
    tps: PersonStanding,
    platformer2d: Footprints,
    topdown2d: MapIcon
  };

  let recent = $state<RecentProject[]>([]);
  let templates = $state<ProjectTemplate[]>([]);
  let template = $state('empty3d');
  let name = $state('MyGame');
  let parent = $state('');
  let busy = $state(false);
  let error = $state('');
  // A folder without moon.json that Open Project can turn into a project.
  let initRoot = $state('');

  onMount(async () => {
    try {
      [recent, templates, parent] = await Promise.all([
        AppAPI.getRecentProjects(),
        AppAPI.listProjectTemplates(),
        AppAPI.defaultProjectsDir()
      ]);
    } catch (err: any) {
      error = err.message || String(err);
    }
  });

  function close() {
    editorStore.showStartScreen = false;
  }

  async function run(fn: () => Promise<void>) {
    busy = true;
    error = '';
    try {
      await fn();
    } catch (err: any) {
      error = err.message || String(err);
    } finally {
      busy = false;
    }
  }

  function openRecent(r: RecentProject) {
    run(() => editorStore.openProject(r.root));
  }

  async function removeRecent(r: RecentProject) {
    await AppAPI.removeRecentProject(r.root).catch(() => {});
    recent = recent.filter((x) => x.root !== r.root);
  }

  async function openFolder() {
    initRoot = '';
    const dir = await AppAPI.selectProjectDirectory().catch(() => '');
    if (!dir) return;
    await run(async () => {
      try {
        await editorStore.openProject(dir);
      } catch (err: any) {
        if (String(err.message || err).startsWith('no moon.json')) initRoot = dir;
        throw err;
      }
    });
  }

  async function browseParent() {
    const dir = await AppAPI.selectProjectDirectory().catch(() => '');
    if (dir) parent = dir;
  }

  function create() {
    run(() => editorStore.createProject(parent, name, template));
  }
</script>

<div class="ide-backdrop">
  <div class="ide-modal lg">
    <div class="ide-modal-head">
      <div>
        {#if editorStore.startView === 'new'}
          <h2>New project</h2>
          <p>Pick a template. It writes moon.json, main.mb, assets/, and a starter scene.</p>
        {:else}
          <h2>Moon Studio</h2>
          <p>Open a project or start a new one</p>
        {/if}
      </div>
      <button class="ide-iconbtn" onclick={close} title="Close"><X size={15} /></button>
    </div>

    <div class="ide-modal-body text-xs">
      {#if editorStore.startView === 'new'}
        <div class="pm-templates">
          {#each templates as t (t.id)}
            {@const Icon = icons[t.id] ?? Box}
            <button
              class="ide-card {template === t.id ? 'ide-choice is-on' : ''}"
              onclick={() => (template = t.id)}
              ondblclick={create}
            >
              <div class="ide-card-icon flex items-center justify-center"><Icon size={16} /></div>
              <div>
                <strong>{t.title} <em>{t.is2D ? '2D' : '3D'}</em></strong>
                <span>{t.description}</span>
              </div>
            </button>
          {/each}
        </div>
        <div class="grid grid-cols-2 gap-4" style="margin-top: 6px;">
          <div>
            <label for="pm-name" class="ide-label">Project name</label>
            <input id="pm-name" class="ide-field" bind:value={name} placeholder="MyGame" />
          </div>
          <div>
            <label for="pm-parent" class="ide-label">Location</label>
            <div class="flex items-center gap-1">
              <input id="pm-parent" class="ide-field font-mono" bind:value={parent} />
              <button class="ide-btn" onclick={browseParent} title="Choose folder">…</button>
            </div>
          </div>
        </div>
        <p class="pm-path">Creates <span class="font-mono">{parent}{parent.includes('\\') ? '\\' : '/'}{name}</span></p>
      {:else}
        <div class="grid grid-cols-2 gap-4">
          <button class="ide-card" onclick={() => { error = ''; editorStore.startView = 'new'; }}>
            <div class="ide-card-icon flex items-center justify-center"><FolderPlus size={16} /></div>
            <div>
              <strong>New project</strong>
              <span>From a 2D or 3D template</span>
            </div>
          </button>
          <button class="ide-card" onclick={openFolder} disabled={busy}>
            <div class="ide-card-icon flex items-center justify-center"><FolderOpen size={16} /></div>
            <div>
              <strong>Open project</strong>
              <span>A folder with a moon.json</span>
            </div>
          </button>
        </div>

        <div class="ide-label pm-recent-head"><History size={12} /> Recent projects</div>
        {#if recent.length === 0}
          <p class="pm-empty">No recent projects yet.</p>
        {:else}
          <ul class="pm-recent">
            {#each recent as r (r.root)}
              <li>
                <button class="pm-recent-open" onclick={() => openRecent(r)} disabled={busy || !r.exists} title={r.root}>
                  <strong>{r.name}</strong>
                  <span class="font-mono">{r.root}</span>
                  {#if !r.exists}<em>missing</em>{/if}
                </button>
                <button class="ide-iconbtn" onclick={() => removeRecent(r)} title="Remove from list"><Trash2 size={12} /></button>
              </li>
            {/each}
          </ul>
        {/if}
      {/if}

      {#if error}
        <div class="ide-badge ide-note err" style="margin-top: 10px;">{error}</div>
      {/if}
      {#if initRoot}
        <button class="ide-btn" style="margin-top: 8px;" onclick={() => run(() => editorStore.initProject(initRoot))}>
          Create moon.json in this folder
        </button>
      {/if}
    </div>

    <div class="ide-modal-foot">
      {#if editorStore.startView === 'new'}
        <button class="ide-btn" onclick={() => { error = ''; editorStore.startView = 'home'; }}><ArrowLeft size={13} /> Back</button>
        <button class="ide-btn primary" onclick={create} disabled={busy || !name.trim()}>
          {#if busy}<Loader2 size={13} class="animate-spin" /> Creating…{:else}<FolderPlus size={13} /> Create project{/if}
        </button>
      {:else}
        <button class="ide-btn" onclick={close}>Continue without a project</button>
      {/if}
    </div>
  </div>
</div>

<style>
  .pm-templates {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: 8px;
  }
  .pm-templates em {
    margin-left: 4px;
    font-style: normal;
    font-size: 10px;
    color: var(--text-dim);
  }
  .pm-path {
    margin-top: 8px;
    color: var(--text-muted);
    word-break: break-all;
  }
  .pm-recent-head {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 6px;
  }
  .pm-empty {
    color: var(--text-muted);
  }
  .pm-recent {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .pm-recent li {
    display: flex;
    align-items: center;
    gap: 4px;
    border: 1px solid var(--border);
    background: var(--bg-input);
    margin-bottom: 6px;
  }
  .pm-recent-open {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    padding: 8px 10px;
    text-align: left;
  }
  .pm-recent-open:hover:not(:disabled) {
    background: var(--accent-dim);
  }
  .pm-recent-open span {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 10px;
    color: var(--text-muted);
  }
  .pm-recent-open em {
    font-style: normal;
    font-size: 10px;
    color: var(--warn);
  }
</style>
