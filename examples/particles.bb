; 3D emitter — WASD move, Esc quits

Graphics3D(800, 600)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — particles")

cam = CreateCamera()
SetPosition(cam, 0, 2, -6)
light = CreateLight()

cube = CreateCube()
SetPosition(cube, 0, 0, 4)
SetEntityColor(cube, 90, 160, 255)

em = CreateEmitter(cube)
SetEmitterRate(em, 40)
SetEmitterMax(em, 80)
SetEmitterLife(em, 1.4)
SetEmitterSpeed(em, 3)
SetEmitterSize(em, 0.22, 0.04)
SetEmitterColor(em, 255, 180, 60, 1, 255, 40, 10, 0)
SetEmitterVelocity(em, 0, 1, 0)
SetEmitterCone(em, 50)
SetEmitterGravity(em, 0, -8, 0)

While Not KeyDown(KEY_ESCAPE)
    dt# = DeltaTime() * 60
    If KeyDown(KEY_W) Then MoveEntity(cam, 0, 0, 0.12 * dt)
    If KeyDown(KEY_S) Then MoveEntity(cam, 0, 0, -0.12 * dt)
    If KeyDown(KEY_A) Then MoveEntity(cam, -0.12 * dt, 0, 0)
    If KeyDown(KEY_D) Then MoveEntity(cam, 0.12 * dt, 0, 0)
    TurnEntity(cube, 0, 0.5 * dt, 0)
    UpdateWorld
    RenderWorld
    Text(12, 12, "Particles follow the cube  |  WASD  |  Esc")
    Flip
Wend
End
