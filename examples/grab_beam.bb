; Convex hull, cylinder collider, grab/throw, projectile, beam, PlaceAtRay.

Graphics3D(960, 540, 0, 2)
SetWindowTitle("BitShin BASIC — Hull / Grab / Beam")
cam = CreateCamera()
SetPosition(cam, 0, 6, -16)
CameraLookAt(cam, 0, 2, 6)
CreateLight()

ground = CreateCube()
SetScale(ground, 16, 0.25, 16)
SetPosition(ground, 0, 0, 8)
CreateBodyBox(ground, 16, 0.25, 16, 0)

barrel = CreateCylinder(0.45, 1.4, 12)
SetPosition(barrel, -3, 1.2, 8)
CreateBodyCylinder(barrel, 0.7, 0.45, 1)

wedge = CreateCone(1.2, 1.6, 10)
SetPosition(wedge, 3, 1.2, 8)
CreateBodyConvex(wedge, 1)

hand = CreateCube()
SetScale(hand, 0.2, 0.2, 0.5)
SetPosition(hand, 0, 2.2, 5)
CreateBodyBox(hand, 0.2, 0.2, 0.5, 1)

crate = CreateCube()
SetScale(crate, 0.45, 0.45, 0.45)
SetPosition(crate, 0.8, 1.2, 6)
CreateBodyBox(crate, 0.45, 0.45, 0.45, 1)

dot = CreateSphere(0.12, 8)
laser = CreateBeam(hand, dot, 0.04)

frames = 0
While 1
    frames = frames + 1
    PlaceAtRay(hand, dot, 18)
    If KeyHit(KEY_SPACE) Then Grab(hand, crate)
    If KeyHit(KEY_T) Then Throw(hand, 16)
    If KeyHit(KEY_G) Then DropGrab(hand)
    UpdateWorld
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
