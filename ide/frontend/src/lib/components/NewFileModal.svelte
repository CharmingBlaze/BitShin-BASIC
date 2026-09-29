<script lang="ts">
  import { X, Box, Gamepad2, FileCode, Layers, Compass } from 'lucide-svelte';
  import { editorStore } from '../stores/editorState.svelte';

  const templates = [
    {
      id: 'blank',
      title: 'Blank script',
      desc: 'Empty file ready for your code.',
      icon: FileCode,
      code: `; BitShin BASIC Script\n\nPrint("Hello BitShin BASIC")\n`
    },
    {
      id: '3d_starter',
      title: '3D scene starter',
      desc: 'Viewport with camera, light, cube, and ground.',
      icon: Box,
      code: `; BitShin BASIC — 3D Scene Starter\nGraphics3D(1280, 720, 0, 2)\nSetWindowTitle("BitShin 3D Starter")\n\ncam = CreateCamera()\nPositionEntity(cam, 0, 3, -8)\nRotateEntity(cam, 15, 0, 0)\n\nsun = CreateLight(1)\nSetLightDirection(sun, 50, 35, 0)\n\nbox = CreateCube()\nEntityColor(box, 56, 189, 248)\n\nground = CreatePlane(50, 50)\nPositionEntity(ground, 0, -1, 0)\nEntityColor(ground, 30, 41, 59)\n\nWhile Not KeyDown(1)\n    dt# = DeltaTime() * 60\n    TurnEntity(box, 0.5 * dt, 0.8 * dt, 0)\n    RenderWorld\n    Flip\nWend\nEnd\n`
    },
    {
      id: '2d_starter',
      title: '2D game starter',
      desc: '2D setup with input, motion, and HUD text.',
      icon: Gamepad2,
      code: `; BitShin BASIC — 2D Game Starter\nGraphics2D(800, 600)\nSetWindowTitle("BitShin 2D Starter")\n\nx# = 400 : y# = 300\nvx# = 4 : vy# = 3\nradius# = 24\n\nWhile Not KeyDown(1)\n    x = x + vx\n    y = y + vy\n    \n    If x <= radius Or x >= 800 - radius Then vx = -vx\n    If y <= radius Or y >= 600 - radius Then vy = -vy\n    \n    Cls\n    Color(56, 189, 248)\n    DrawOval(x - radius, y - radius, radius * 2, radius * 2)\n    \n    Color(255, 255, 255)\n    Text(20, 20, "BitShin 2D — Press ESC to exit")\n    Flip\nWend\nEnd\n`
    },
    {
      id: 'physics_jolt',
      title: '3D physics (Jolt)',
      desc: 'Rigid bodies falling onto a static floor.',
      icon: Layers,
      code: `; BitShin BASIC — 3D Jolt Physics\nGraphics3D(1280, 720, 0, 2)\nSetWindowTitle("BitShin Jolt Physics")\n\ncam = CreateCamera()\nPositionEntity(cam, 0, 6, -15)\nRotateEntity(cam, 20, 0, 0)\n\nsun = CreateLight(1)\nSetLightDirection(sun, -50, 40, 0)\n\n; Static floor\nfloor = CreateBox(30, 1, 30)\nPositionEntity(floor, 0, -0.5, 0)\nEntityColor(floor, 51, 65, 85)\nPhysicsStaticBody(floor)\n\n; Dynamic bodies\nDim cubes(10)\nFor i = 0 To 9\n    cubes(i) = CreateCube()\n    PositionEntity(cubes(i), Rnd(-3, 3), 4 + i * 2.5, Rnd(-3, 3))\n    EntityColor(cubes(i), Rnd(100, 255), Rnd(100, 255), Rnd(100, 255))\n    PhysicsBody(cubes(i), 1.0)\nNext\n\nWhile Not KeyDown(1)\n    UpdatePhysics(DeltaTime())\n    RenderWorld\n    Flip\nWend\nEnd\n`
    },
    {
      id: 'freelook_cam',
      title: 'First-person free look',
      desc: 'WASD + mouse navigation.',
      icon: Compass,
      code: `; BitShin BASIC — Free Look Camera\nGraphics3D(1280, 720, 0, 2)\nSetWindowTitle("BitShin FreeLook Demo")\n\ncam = CreateFreeCamera()\nPositionEntity(cam, 0, 2, -10)\n\nsun = CreateLight(1)\nSetLightDirection(sun, 50, 35, 0)\n\n; Scatter demo columns\nFor x = -15 To 15 Step 5\n    For z = -15 To 15 Step 5\n        col = CreateCylinder(0.8, 4, 16)\n        PositionEntity(col, x, 2, z)\n        EntityColor(col, 100, 116, 139)\n    Next\nNext\n\nWhile Not KeyDown(1)\n    UpdateFreeLook(cam, 10)\n    RenderWorld\n    Flip\nWend\nEnd\n`
    }
  ];

  function chooseTemplate(t: typeof templates[0]) {
    editorStore.newTab(t.title, t.code);
    editorStore.showNewModal = false;
  }
</script>

<div class="ide-backdrop">
  <div class="ide-modal lg">
    <div class="ide-modal-head">
      <div>
        <h2>New file</h2>
        <p>Choose a starter template or a blank script</p>
      </div>
      <button class="ide-iconbtn" onclick={() => editorStore.showNewModal = false}><X size={15} /></button>
    </div>
    <div class="ide-modal-body">
      {#each templates as t}
        {@const Icon = t.icon}
        <button class="ide-card" onclick={() => chooseTemplate(t)}>
          <div class="ide-card-icon flex items-center justify-center"><Icon size={16} /></div>
          <div>
            <strong>{t.title}</strong>
            <span>{t.desc}</span>
          </div>
        </button>
      {/each}
    </div>
  </div>
</div>
