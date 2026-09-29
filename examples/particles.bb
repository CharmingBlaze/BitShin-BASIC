; Particles — soft sprites and tumbling cubes from a spinning emitter.
; WASD move the camera. Esc quits.

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — particles")

; Camera / light
cam = CreateCamera().Position([0, 2, -6])
light = CreateLight()

; World
ground = CreatePlane(14, 14).Position([0, 0, 4]).Color(36, 42, 52)
cube = CreateCube().Position([0, 1, 4]).Color(90, 160, 255)

; Soft sprite emitter
em = CreateEmitter(cube)
SetEmitterShape(em, "soft")
SetEmitterRate(em, 40)
SetEmitterMax(em, 80)
SetEmitterLife(em, 1.4)
SetEmitterSpeed(em, 3)
SetEmitterSize(em, 0.22, 0.04)
SetEmitterColor(em, 255, 180, 60, 1, 255, 40, 10, 0)
SetEmitterVelocity(em, 0, 1, 0)
SetEmitterCone(em, 50)
SetEmitterGravity(em, 0, -8, 0)

; Cube bits emitter
bits = CreateEmitter(cube)
SetEmitterShape(bits, "cube")
SetEmitterRate(bits, 8)
SetEmitterMax(bits, 24)
SetEmitterLife(bits, 1.8)
SetEmitterSpeed(bits, 2.2)
SetEmitterSize(bits, 0.12, 0.04)
SetEmitterColor(bits, 140, 210, 255, 1, 40, 80, 140, 0.2)
SetEmitterVelocity(bits, 0, 1, 0)
SetEmitterCone(bits, 70)
SetEmitterGravity(bits, 0, -9, 0)

; Loop
While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    If KeyDown(KEY_W) Then cam.Move(0, 0, 0.12 * dt)
    If KeyDown(KEY_S) Then cam.Move(0, 0, -0.12 * dt)
    If KeyDown(KEY_A) Then cam.Move(-0.12 * dt, 0, 0)
    If KeyDown(KEY_D) Then cam.Move(0.12 * dt, 0, 0)
    cube.Turn(0, 0.5 * dt, 0)
    UpdateWorld
    RenderWorld
    Text(12, 12, "Soft sprites and tumbling cubes  |  WASD  |  Esc")
    Flip
Wend
End
