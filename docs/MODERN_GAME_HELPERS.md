# Modern Game Helpers & Character Controllers

BitShin BASIC provides a high-level suite of modern character controllers, camera helpers, math utilities, spatial audio, and time controls.

---

## 1. Character Controllers

### First-Person Controller (FPS)
Creates an FPS character with smooth mouse look ($-89^\circ$ to $+89^\circ$), camera-relative movement, sprinting, smooth crouch transitions, coyote time, and step head-bobbing.

```basic
player = CreatePivot()
cam = CreateCamera()
CreateFPSController(player, cam, 1.8, 0.4, 6.0, 10.5, 9.5)

While 1
    walkX# = GetAxis(KEY_A, KEY_D)
    walkZ# = GetAxis(KEY_S, KEY_W)
    jump = KeyHit(KEY_SPACE)
    sprint = KeyDown(KEY_LEFT_SHIFT)
    crouch = KeyDown(KEY_LEFT_CONTROL)

    UpdateFPS(player, walkX#, walkZ#, jump, sprint, crouch, 0.15)
    UpdateWorld
    RenderWorld
    Flip
Wend
```

**Commands:**
- `CreateFPSController(entity, cam [, height, radius, walkSpeed, sprintSpeed, jumpForce])`
- `UpdateFPS(entity, walkX, walkZ, isJump, isSprint, isCrouch, mouseSensitivity)`
- `GetFPSPitch#(entity)` / `GetFPSYaw#(entity)`
- `IsFPSGrounded(entity)`

---

### Third-Person Action Controller (TPS)
Provides orbital character control with smooth rotation toward movement direction, precision aim mode, and an obstacle-avoiding **Spring-Arm camera**.

```basic
player = CreatePivot()
cam = CreateCamera()
CreateTPSController(player, cam, 1.8, 0.45, 6.5, 11.0, 10.0)

While 1
    moveX# = GetAxis(KEY_A, KEY_D)
    moveZ# = GetAxis(KEY_S, KEY_W)
    jump = KeyHit(KEY_SPACE)
    sprint = KeyDown(KEY_LEFT_SHIFT)
    aim = MouseDown(2)

    camYaw# = camYaw# + MouseDeltaX() * 0.2
    camPitch# = Clamp(camPitch# + MouseDeltaY() * 0.15, -15, 60)

    UpdateTPS(player, moveX#, moveZ#, jump, sprint, aim, 4.5, camPitch#, camYaw#)
    UpdateWorld
    RenderWorld
    Flip
Wend
```

**Commands:**
- `CreateTPSController(entity, cam [, height, radius, moveSpeed, sprintSpeed, jumpForce])`
- `UpdateTPS(entity, moveX, moveZ, isJump, isSprint, isAimMode, camDist, camPitch, camYaw)`
- `GetTPSYaw#(entity)`
- `IsTPSGrounded(entity)`

---

### Top-Down / Twin-Stick Controller
Ideal for ARPGs, top-down shooters, and roguelikes with instant mouse cursor aiming and dash evasion.

```basic
player = CreatePivot()
cam = CreateCamera()
CreateTopDownController(player, 8.0, 18.0)

While 1
    moveX# = GetAxis(KEY_A, KEY_D)
    moveZ# = GetAxis(KEY_S, KEY_W)
    aimX# = GetMouseWorldX(cam, 0)
    aimZ# = GetMouseWorldZ(cam, 0)
    dash = KeyHit(KEY_SPACE)

    UpdateTopDown(player, moveX#, moveZ#, aimX#, aimZ#, dash)
    CameraFollow(cam, player, 16, 20, 10, 0, 55)
    UpdateWorld
    RenderWorld
    Flip
Wend
```

**Commands:**
- `CreateTopDownController(entity [, speed, rotSpeed])`
- `UpdateTopDown(entity, moveX, moveZ, aimTargetX, aimTargetZ, isDash)`
- `GetTopDownAimYaw#(entity)`

---

### 2.5D / Platformer Controller
Equipped with variable jump height (cuts jump on early key release), double/multi-jump, coyote time, and jump buffering.

```basic
player = CreatePivot()
CreatePlatformerController(player, 8.5, 12.0, 2)

While 1
    moveX# = GetAxis(KEY_A, KEY_D)
    jumpPressed = KeyHit(KEY_SPACE)
    jumpHeld = KeyDown(KEY_SPACE)

    UpdatePlatformer(player, moveX#, jumpPressed, jumpHeld)
    UpdateWorld
    RenderWorld
    Flip
Wend
```

**Commands:**
- `CreatePlatformerController(entity [, speed, jumpForce, maxJumps])`
- `UpdatePlatformer(entity, moveX, jumpPressed, jumpHeld)`
- `IsPlatformerGrounded(entity)`

---

## 2. Camera Helpers & Systems

### Spring-Arm Collision Avoidance
Raycasts against static physics geometry to smoothly pull the camera forward when backed against walls or obstacles, preventing clipping inside level geometry.

```basic
CameraSpringArm(cam, target, offsetY#, dist#, radius#, damp#, yaw#, pitch#)
```

### Orbital Camera
Smooth spherical orbit around target entity:

```basic
CameraOrbit(cam, target, dist#, height#, yaw#, pitch#)
```

### Screen / Camera Shake
Procedural trauma-decay shake for explosions, impacts, gun recoil, and earthquakes:

```basic
CameraShake(trauma#, duration#, frequency#) ; Set trauma (0.0 to 1.0)
AddCameraShake(trauma#)                      ; Stack additional trauma
UpdateCameraShake(cam)                       ; Call inside render loop
```

### Smooth Look Tracking
Damped rotation tracking towards any 3D coordinate:

```basic
CameraSmoothLook(cam, targetX#, targetY#, targetZ#, speed#)
```

### RTS / Strategy Camera
Isometric / top-down camera with panning, scroll-wheel zooming, and rotation:

```basic
CameraRTS(cam, panX#, panZ#, zoom#, pitch#, yaw#)
```

---

## 3. Math & Smoothing Utilities

| Command | Description |
| --- | --- |
| `Lerp#(a#, b#, t#)` | Linear interpolation between `a` and `b`. |
| `LerpAngle#(a#, b#, t#)` | Interpolates angles across the shortest $360^\circ$ wrap. |
| `SmoothDamp#(current#, target#, smoothTime#, maxSpeed#)` | Critically damped spring smoothing (Unity-style). |
| `MoveTowards#(current#, target#, maxDelta#)` | Moves `current` toward `target` by at most `maxDelta`. |
| `Clamp#(val#, min#, max#)` | Clamps value between `min` and `max`. |
| `WrapAngle#(angle#)` | Normalizes angle into $[-180^\circ, +180^\circ]$. |
| `AngleDiff#(a#, b#)` | Calculates shortest angular difference between two angles. |
| `Distance3D#(x1#, y1#, z1#, x2#, y2#, z2#)` | 3D Euclidean distance. |
| `Distance2D#(x1#, y1#, x2#, y2#)` | 2D distance. |

---

## 4. Input & Raycasting Helpers

- `GetAxis#(negativeKey, positiveKey)`: Returns `-1.0`, `0.0`, or `+1.0`.
- `GetMouseWorldX#(cam, targetPlaneY#)`: Computes world X position on plane under cursor.
- `GetMouseWorldZ#(cam, targetPlaneY#)`: Computes world Z position on plane under cursor.
- `IsGrounded(entity [, rayLength#])`: Returns `1` if physics ray detects floor beneath entity.

---

## 5. Audio & Time Scale

- `PlaySound3D(sound, x#, y#, z#, maxDist# [, baseVolume#])`: Plays sound with distance falloff relative to the active camera.
- `SetTimeScale(scale#)` / `TimeScale#([scale#])`: Sets simulation speed (`0.5` = half-speed bullet time, `1.0` = normal, `2.0` = fast-forward, `0.0` = pause).

---

## 6. Tween Engine

Smoothly interpolates position, rotation, or material color over time without writing manual timers:

```basic
TweenPosition(entity, targetX#, targetY#, targetZ#, duration#, "out")
TweenRotation(entity, pitch#, yaw#, roll#, duration#, "inout")
TweenColor(entity, r, g, b, duration#, "smooth")
```
Supported ease modes: `"linear"`, `"in"`, `"out"`, `"inout"`, `"smooth"`, `"smoother"`.
