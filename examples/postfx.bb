; Fullscreen post stack (GL 3.3): tonemap, exposure, cheap bloom, FXAA.
; Close the window to quit — no Escape.

Graphics3D(960, 600)
SetWindowTitle("PostFX")
cam = CreateCamera()
SetPosition(cam, 0, 2.2, -7)
sun = CreateLight()
SetRotation(sun, 50, 30, 0)

ground = CreatePlane()
SetPosition(ground, 0, -1, 8)
SetEntityColor(ground, 36, 40, 48)

For i = 0 To 7
    b = CreateSphere()
    SetPosition(b, (i - 3.5) * 1.4, 0.2, 6)
    SetEntityColor(b, 40 + i * 28, 80, 220 - i * 18)
Next

EnablePostFX(True)
SetExposure(1.15)
SetBloom(0.35)
SetFXAA(True)
SetColorGrade(1.05, 1.1, 255, 240, 230)

exp# = 1.15
bloom# = 0.35

While Not WindowShouldClose()
    TurnEntity(sun, 0, 0.15, 0)
    If KeyDown(KEY_UP) Then exp# = exp# + 0.01
    If KeyDown(KEY_DOWN) Then exp# = exp# - 0.01
    If KeyDown(KEY_RIGHT) Then bloom# = bloom# + 0.01
    If KeyDown(KEY_LEFT) Then bloom# = bloom# - 0.01
    SetExposure(exp#)
    SetBloom(bloom#)
    RenderWorld
    Text(12, 12, "PostFX  exposure=" + Str(GetExposure()) + "  bloom=" + Str(GetBloom()))
    Text(12, 30, "Arrows: exposure / bloom")
    Flip
Wend
End
