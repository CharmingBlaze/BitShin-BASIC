; PostFX — tonemap, exposure, bloom, and FXAA on a row of spheres.
; Arrow keys tweak exposure / bloom. Close the window to quit — no Escape.

Graphics3D(960, 600)
SetWindowTitle("BitShin BASIC — PostFX")

; Camera / light
cam = CreateCamera().Position([0, 2.2, -7])
sun = CreateLight().Rotate(50, 30, 0)

; World
ground = CreatePlane().Position([0, -1, 8]).Color(36, 40, 48)

For i = 0 To 7
    b = CreateSphere().Position([(i - 3.5) * 1.4, 0.2, 6]).Color(40 + i * 28, 80, 220 - i * 18)
Next

; Post stack
EnablePostFX(True)
SetExposure(1.15)
SetBloom(0.35)
SetFXAA(True)
SetColorGrade(1.05, 1.1, 255, 240, 230)

exp# = 1.15
bloom# = 0.35

; Loop
While Not WindowShouldClose()
    sun.Turn(0, 0.15, 0)
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
