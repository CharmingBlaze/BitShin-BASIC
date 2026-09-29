; Canal — low camera on dark water, haze, and a sun path between buildings.
; WASD move, mouse look, Q/E height. Esc quits (after a short startup delay).

Graphics3D(1280, 720)
SetWindowTitle("BitShin BASIC — Canal")
SetCameraClsColor(186, 198, 210)
SetAmbientLight(64, 72, 80)
HidePointer()

; Camera
cam = CreateCamera()
SetCameraRange(cam, 0.12, 900)
cam.Position([0, 1.25, -24])

CreateSkyBox("default")

; Light / shadows
sun = CreateDirectionalLight()
SetLightDirection(sun, 52, 180, 0)
SetLightColor(sun, 255, 246, 224)
SetLightShadow(sun, True)
EnableShadows(True)
SetShadowResolution(1024)

; Fog / post
CameraFogMode(2)
CameraFogColor(176, 190, 204)
CameraFogDensity(0.015)

EnablePostFX(True)
SetTonemap("neutral")
SetExposure(1.08)
SetBloom(0.38)

; Water
water = CreateWater(34, 240, 64)
SetPosition(water, 0, 0, 72)
SetWaterStyle("scenic")
SetWaterColor(9, 46, 36)
SetWaterWaves(0)
SetWaterFollow(False)
SetWaterLevel(0)
SetWaterSpeed(0.022)
SetWaterWaveStrength(0.016)
EnableWaterReflection(True)
EnableWaterRefraction(True)

; Canal sides
For i = 0 To 18
    h# = 4.4 + (i Mod 3) * 0.85
    left = CreateCube().Scale(3.2, h, 7.4).Position([-12.5, h * 0.5 - 0.15, i * 12])
    shade = 104 + (i Mod 4) * 9
    left.Color(shade, shade - 1, shade - 8)

    right = CreateCube()
    rh# = 3.6 + ((i + 1) Mod 4) * 0.7
    right.Scale(2.8, rh, 7.4)
    right.Position([12.2, rh * 0.5 - 0.15, i * 12 + 3])
    rshade = 118 + (i Mod 3) * 7
    right.Color(rshade, rshade - 3, rshade - 10)
Next

; Far cliffs
For i = 0 To 5
    cliff = CreateCube()
    ch# = 16 + (i Mod 3) * 3
    cliff.Scale(9, ch, 12)
    cliff.Position([-22 + i * 9, ch * 0.42, 168])
    cliff.Color(92, 98, 104)
Next

; Loop
yaw# = 0
pitch# = -3
frames = 0
SetMousePosition(640, 360)

While 1
    frames = frames + 1
    dt# = DeltaTime()
    yaw = yaw + MouseDeltaX() * 0.12
    pitch = pitch - MouseDeltaY() * 0.12
    If pitch > 70 Then pitch = 70
    If pitch < -70 Then pitch = -70
    cam.Rotate(pitch, yaw, 0)
    SetMousePosition(640, 360)

    spd# = 9 * dt
    If KeyDown(KEY_W) Then cam.Move(0, 0, spd)
    If KeyDown(KEY_S) Then cam.Move(0, 0, -spd)
    If KeyDown(KEY_A) Then cam.Move(-spd, 0, 0)
    If KeyDown(KEY_D) Then cam.Move(spd, 0, 0)
    If KeyDown(KEY_Q) Then cam.Position(EntityX(cam), EntityY(cam) + 6 * dt, EntityZ(cam))
    If KeyDown(KEY_E) Then cam.Position(EntityX(cam), EntityY(cam) - 6 * dt, EntityZ(cam))

    Text(12, 12, "WASD  mouse look  QE height  Esc quit")
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
