; 3D physics gameplay — Jolt by default. -tags nojolt uses the software fallback.
; Space kicks the ball. WASD moves the capsule. Esc / X quits.

SetWindowTitle("BitShin BASIC — Physics 3D")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(18, 20, 28)

cam = CreateCamera()
SetPosition(cam, 0, 8, -16)
SetRotation(cam, 22, 0, 0)
CreateLight()

ground = CreateCube()
SetScale(ground, 10, 0.2, 10)
SetPosition(ground, 0, 0, 8)
SetEntityColor(ground, 50, 56, 70)
CreateRigidBodyBox(ground, 10, 0.2, 10, 0)

ball = CreateSphere(12)
SetPosition(ball, -3, 8, 8)
SetEntityColor(ball, 80, 190, 255)
CreateRigidBodySphere(ball, 1, 1)

hero = CreateCapsule(0.4, 0.9, 8)
SetPosition(hero, 3, 3, 8)
SetEntityColor(hero, 255, 170, 80)
CreateCharacterController(hero, 1.8, 0.4, 50, 100)

Print "Physics backend:", GetPhysicsBackend()
Print "Character:", GetPhysicsCharacter()

While Not KeyDown(1) And Not KeyDown(KEY_X)
    vx# = 0
    vz# = 0
    If KeyDown(KEY_A) Then vx = -4
    If KeyDown(KEY_D) Then vx = 4
    If KeyDown(KEY_W) Then vz = 4
    If KeyDown(KEY_S) Then vz = -4
    MoveCharacter(hero, vx, vz)
    If KeyHit(KEY_SPACE) Then ApplyImpulse(ball, 0, 8, 0)

    UpdateWorld
    hit = Raycast(0, 12, 8, 0, -20, 0)

    RenderWorld
    Text(16, 16, "backend " + GetPhysicsBackend() + "  char " + GetPhysicsCharacter())
    Text(16, 36, "ray hit " + hit + "  at y=" + Int(PickedY()) + "  onGround " + CharacterOnGround(hero))
    Text(16, 56, "WASD walk  |  Space kick ball  |  Esc/X quit")
    Flip
Wend
End
