<script lang="ts">
  import { X, Box, Gamepad2, Sparkles, FileCode, Layers, Compass } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  const templates = [
    {
      id: 'blank',
      title: 'Blank BASIC Script',
      desc: 'Empty file ready for your custom code.',
      icon: FileCode,
      code: `; BitShin BASIC Script\n\nPrint("Hello BitShin BASIC")\n`
    },
    {
      id: '3d_starter',
      title: '3D Scene Starter',
      desc: 'Complete OpenGL 3D viewport with camera, directional light, rotating cube, and ground plane.',
      icon: Box,
      code: `; BitShin BASIC — 3D Scene Starter\nGraphics3D(1280, 720, 0, 2)\nSetWindowTitle("BitShin 3D Starter")\n\ncam = CreateCamera()\nPositionEntity(cam, 0, 3, -8)\nRotateEntity(cam, 15, 0, 0)\n\nsun = CreateLight(1)\nSetLightDirection(sun, -45, 30, 0)\n\nbox = CreateCube()\nEntityColor(box, 56, 189, 248)\n\nground = CreatePlane(50, 50)\nPositionEntity(ground, 0, -1, 0)\nEntityColor(ground, 30, 41, 59)\n\nWhile Not KeyDown(1)\n    dt# = DeltaTime() * 60\n    TurnEntity(box, 0.5 * dt, 0.8 * dt, 0)\n    RenderWorld\n    Flip\nWend\nEnd\n`
    },
    {
      id: '2d_starter',
      title: '2D Game Starter',
      desc: 'Ebiten 2D engine setup with keyboard input, bouncing sprites, and HUD text.',
      icon: Gamepad2,
      code: `; BitShin BASIC — 2D Game Starter\nGraphics2D(800, 600)\nSetWindowTitle("BitShin 2D Starter")\n\nx# = 400 : y# = 300\nvx# = 4 : vy# = 3\nradius# = 24\n\nWhile Not KeyDown(1)\n    x = x + vx\n    y = y + vy\n    \n    If x <= radius Or x >= 800 - radius Then vx = -vx\n    If y <= radius Or y >= 600 - radius Then vy = -vy\n    \n    Cls\n    Color(56, 189, 248)\n    DrawOval(x - radius, y - radius, radius * 2, radius * 2)\n    \n    Color(255, 255, 255)\n    Text(20, 20, "BitShin 2D — Press ESC to exit")\n    Flip\nWend\nEnd\n`
    },
    {
      id: 'physics_jolt',
      title: '3D Physics (Jolt)',
      desc: 'Real-time rigid body dynamics simulation with falling tumbling shapes.',
      icon: Layers,
      code: `; BitShin BASIC — 3D Jolt Physics\nGraphics3D(1280, 720, 0, 2)\nSetWindowTitle("BitShin Jolt Physics")\n\ncam = CreateCamera()\nPositionEntity(cam, 0, 6, -15)\nRotateEntity(cam, 20, 0, 0)\n\nsun = CreateLight(1)\nSetLightDirection(sun, -50, 40, 0)\n\n; Static floor\nfloor = CreateBox(30, 1, 30)\nPositionEntity(floor, 0, -0.5, 0)\nEntityColor(floor, 51, 65, 85)\nPhysicsStaticBody(floor)\n\n; Dynamic bodies\nDim cubes(10)\nFor i = 0 To 9\n    cubes(i) = CreateCube()\n    PositionEntity(cubes(i), Rnd(-3, 3), 4 + i * 2.5, Rnd(-3, 3))\n    EntityColor(cubes(i), Rnd(100, 255), Rnd(100, 255), Rnd(100, 255))\n    PhysicsBody(cubes(i), 1.0)\nNext\n\nWhile Not KeyDown(1)\n    UpdatePhysics(DeltaTime())\n    RenderWorld\n    Flip\nWend\nEnd\n`
    },
    {
      id: 'freelook_cam',
      title: 'First-Person Free Look',
      desc: 'WASD + Mouse 3D navigation in an environment with lighting.',
      icon: Compass,
      code: `; BitShin BASIC — Free Look Camera\nGraphics3D(1280, 720, 0, 2)\nSetWindowTitle("BitShin FreeLook Demo")\n\ncam = CreateFreeCamera()\nPositionEntity(cam, 0, 2, -10)\n\nsun = CreateLight(1)\nSetLightDirection(sun, -45, 30, 0)\n\n; Scatter demo columns\nFor x = -15 To 15 Step 5\n    For z = -15 To 15 Step 5\n        col = CreateCylinder(0.8, 4, 16)\n        PositionEntity(col, x, 2, z)\n        EntityColor(col, 100, 116, 139)\n    Next\nNext\n\nWhile Not KeyDown(1)\n    UpdateFreeLook(cam, 10)\n    RenderWorld\n    Flip\nWend\nEnd\n`
    }
  ];

  function chooseTemplate(t: typeof templates[0]) {
    editorStore.newTab(t.title, t.code);
    editorStore.showNewModal = false;
  }
</script>

<div class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center p-4 z-50 select-none animate-in fade-in duration-150">
  <div class="bg-slate-900 border border-slate-700/80 rounded-xl shadow-2xl w-full max-w-2xl overflow-hidden flex flex-col max-h-[85vh]">
    <!-- Modal Header -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-slate-800 bg-slate-950/60">
      <div class="flex items-center gap-2">
        <div class="p-1.5 rounded-lg bg-sky-500/20 text-sky-400">
          <Sparkles size={18} />
        </div>
        <div>
          <h2 class="text-sm font-bold text-slate-100">Create New BitShin Script</h2>
          <p class="text-xs text-slate-400">Select a game template or start with a blank file</p>
        </div>
      </div>
      <button
        onclick={() => editorStore.showNewModal = false}
        class="p-1.5 rounded-lg hover:bg-slate-800 text-slate-400 hover:text-slate-200 transition"
      >
        <X size={16} />
      </button>
    </div>

    <!-- Templates Grid -->
    <div class="p-5 overflow-y-auto space-y-2.5">
      {#each templates as t}
        {@const Icon = t.icon}
        <button
          onclick={() => chooseTemplate(t)}
          class="w-full text-left p-3.5 rounded-xl bg-slate-950/60 hover:bg-slate-800/80 border border-slate-800 hover:border-sky-500/50 transition-all flex items-start gap-3.5 group"
        >
          <div class="p-2.5 rounded-lg bg-slate-900 text-sky-400 group-hover:bg-sky-500 group-hover:text-white transition shadow-sm">
            <Icon size={20} />
          </div>
          <div class="flex-1">
            <div class="text-xs font-bold text-slate-200 group-hover:text-sky-300 transition">{t.title}</div>
            <div class="text-[11px] text-slate-400 leading-relaxed mt-0.5">{t.desc}</div>
          </div>
        </button>
      {/each}
    </div>
  </div>
</div>
