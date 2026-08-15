; Capsule + WASD impulse + jump raycast. Esc after Flip.

SetWindowTitle("BitShin BASIC — Physics body")
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
CreateBodyBox(ground, 10, 0.2, 10, 0)

playerEntity = CreateCapsule(0.4, 0.9, 8)
SetPosition(playerEntity, 0, 3, 8)
SetEntityColor(playerEntity, 80, 190, 255)
playerBody = CreateBodyCapsule(playerEntity, 0.9, 0.4, 2)
ActivateBody(playerBody)

frames = 0
While 1
    frames = frames + 1
    moveDirX# = 0.0
    moveDirZ# = 0.0
    If KeyDown(KEY_A) Then moveDirX = -5.0
    If KeyDown(KEY_D) Then moveDirX = 5.0
    If KeyDown(KEY_W) Then moveDirZ = 5.0
    If KeyDown(KEY_S) Then moveDirZ = -5.0
    ApplyImpulse(playerBody, moveDirX, 0, moveDirZ)
    hit = Raycast(EntityX(playerEntity), EntityY(playerEntity), EntityZ(playerEntity), 0, -2.5, 0)
    If hit And KeyHit(KEY_SPACE) Then ApplyImpulse(playerBody, 0, 15.0, 0)
    UpdateWorld()
    RenderWorld()
    Text(16, 16, "WASD impulse  |  Space jump if grounded  |  Esc after frames")
    Text(16, 36, "backend " + GetPhysicsBackend() + "  ray " + hit)
    Flip()
    If frames > 8 And KeyHit(KEY_ESCAPE) Then End
Wend
End
