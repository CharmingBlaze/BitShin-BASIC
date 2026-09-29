; Grab / beam — convex hull, cylinder collider, grab/throw, and PlaceAtRay laser.
; Space grab, T throw, G drop. Esc quits (after a short startup delay).

Graphics3D(960, 540, 0, 2)
SetWindowTitle("BitShin BASIC — Hull / Grab / Beam")

; Camera / light
cam = CreateCamera().Position([0, 6, -16])
CameraLookAt(cam, 0, 2, 6)
CreateLight()

; World
ground = CreateCube().Scale(16, 0.25, 16).Position([0, 0, 8])
CreateBodyBox(ground, 16, 0.25, 16, 0)

barrel = CreateCylinder(0.45, 1.4, 12).Position([-3, 1.2, 8])
CreateBodyCylinder(barrel, 0.7, 0.45, 1)

wedge = CreateCone(1.2, 1.6, 10).Position([3, 1.2, 8])
CreateBodyConvex(wedge, 1)

hand = CreateCube().Scale(0.2, 0.2, 0.5).Position([0, 2.2, 5])
CreateBodyBox(hand, 0.2, 0.2, 0.5, 1)

crate = CreateCube().Scale(0.45, 0.45, 0.45).Position([0.8, 1.2, 6])
CreateBodyBox(crate, 0.45, 0.45, 0.45, 1)

dot = CreateSphere(0.12, 8)
laser = CreateBeam(hand, dot, 0.04)

; Loop
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
