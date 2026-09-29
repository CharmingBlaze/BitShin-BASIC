; Contact demo — ball drops onto a red hazard cube.
; Esc quits after a few frames.

SetWindowTitle("BitShin BASIC — Physics contacts")
Graphics3D(960, 600, 0, 2)
SetCameraClsColor(18, 20, 28)

; Camera
cam = CreateCamera()
cam.Position([0, 8, -14])
cam.Rotate(20, 0, 0)
CreateLight()

; Static ground
ground = CreateCube().Scale(10, 0.2, 10).Position([0, 0, 8]).Color(50, 56, 70)
CreateBodyBox(ground, 10, 0.2, 10, 0)

; Hazard (collision type 2)
hazard = CreateCube().Scale(1, 1, 1).Position([2, 1.2, 8]).Color(220, 70, 70)
SetEntityType(hazard, 2)
CreateBodyBox(hazard, 1, 1, 1, 0)

; Ball (collision type 1)
ball = CreateSphere(12).Position([-2, 6, 8]).Color(80, 190, 255)
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
