; Native Jolt contacts feed EntityCollided / CollisionX.

SetWindowTitle("BitShin BASIC — Physics contacts")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(18, 20, 28)

cam = CreateCamera()
SetPosition(cam, 0, 8, -14)
SetRotation(cam, 20, 0, 0)
CreateLight()

ground = CreateCube()
SetScale(ground, 10, 0.2, 10)
SetPosition(ground, 0, 0, 8)
SetEntityColor(ground, 50, 56, 70)
CreateBodyBox(ground, 10, 0.2, 10, 0)

hazard = CreateCube()
SetScale(hazard, 1, 1, 1)
SetPosition(hazard, 2, 1.2, 8)
SetEntityColor(hazard, 220, 70, 70)
SetEntityType(hazard, 2)
CreateBodyBox(hazard, 1, 1, 1, 0)

ball = CreateSphere(12)
SetPosition(ball, -2, 6, 8)
SetEntityColor(ball, 80, 190, 255)
SetEntityType(ball, 1)
CreateBodySphere(ball, 1, 1)

frames = 0
While 1
    frames = frames + 1
    UpdateWorld()
    other = EntityCollided(ball, hazard)
    RenderWorld()
    Text(16, 16, "Drop the ball on the red cube  |  Esc after frames")
    Text(16, 36, "EntityCollided " + other + "  x=" + Int(CollisionX(ball)))
    Flip()
    If frames > 8 And KeyHit(KEY_ESCAPE) Then End
Wend
End
