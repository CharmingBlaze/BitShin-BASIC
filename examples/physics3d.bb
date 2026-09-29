; Physics 3D — capsule walks with WASD; Space kicks the ball.
; Esc or X quits. (Jolt by default; -tags nojolt uses the software fallback.)

SetWindowTitle("BitShin BASIC — Physics 3D")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(18, 20, 28)

; Camera
cam = CreateCamera()
cam.Position([0, 8, -16])
cam.Rotate(22, 0, 0)
CreateLight()

; Static ground
ground = CreateCube().Scale(10, 0.2, 10).Position([0, 0, 8]).Color(50, 56, 70)
CreateRigidBodyBox(ground, 10, 0.2, 10, 0)

; Kick ball
ball = CreateSphere(12).Position([-3, 8, 8]).Color(80, 190, 255)
CreateRigidBodySphere(ball, 1, 1)

; Character controller
hero = CreateCapsule(0.4, 0.9, 8).Position([3, 3, 8]).Color(255, 170, 80)
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
