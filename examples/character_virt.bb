; CharacterVirtual controller. Esc after a few frames.

SetWindowTitle("BitShin BASIC — Character")
Graphics3D(960, 540, 0, 2)
SetCameraClsColor(18, 22, 30)
CreateLight()

cam = CreateCamera()
SetPosition(cam, 0, 6, -14)
SetRotation(cam, 18, 0, 0)

ground = CreateCube()
SetScale(ground, 12, 0.2, 12)
SetPosition(ground, 0, 0, 8)
SetEntityColor(ground, 50, 58, 70)
CreateRigidBodyBox(ground, 12, 0.2, 12, 0)

hero = CreateCapsule(0.4, 0.9, 8)
SetPosition(hero, 0, 2.2, 8)
SetEntityColor(hero, 255, 170, 80)
CreateCharacterController(hero, 1.8, 0.4, 50, 100)

frames = 0
While 1
    frames = frames + 1
    vx# = 0
    vz# = 0
    If KeyDown(KEY_A) Then vx = -5
    If KeyDown(KEY_D) Then vx = 5
    If KeyDown(KEY_W) Then vz = 5
    If KeyDown(KEY_S) Then vz = -5
    MoveCharacter(hero, vx, vz)
    ExtendedUpdate(hero)
    UpdateWorld
    RenderWorld
    Text(12, 12, "Character  WASD  ground=" + GetCharacterGroundState(hero) + "  " + GetPhysicsCharacter())
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
