; Water — scenic DuDv + Fresnel with planar reflect/refract and shoreline props.
; WASD orbit, Q/E pitch, arrows move the boat. Esc quits (after a short startup delay).

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Pretty Water")
SetCameraClsColor(72, 128, 178)
SetAmbientLight(38, 48, 62)

; Camera / sky
cam = CreateCamera()
CreateSkyBox("default")

; Light
sun = CreateDirectionalLight()
SetLightDirection(sun, 28, 42, 8)
SetLightColor(sun, 255, 236, 200)

; Water
water = CreateWater(160, 160, 12)
SetWaterStyle("scenic")
SetWaterColor(18, 78, 108)
EnableWaterReflection(True)
EnableWaterRefraction(True)
SetWaterSpeed(0.04)
SetWaterWaveStrength(0.05)
SetWaterWaves(0)
SetWaterFollow(False)
SetWaterLevel(0)
SetUnderwaterFog(8, 35, 50, 0.09)

; Shore — so the reflection FBO has something obvious to grab
isle = CreateProcTerrain(5, 11, 4, 5, 22)
SetTerrainStreamRadius(isle, 1)

pier = CreateCube().Scale(8, 0.18, 1.2).Position([10, 0.35, 4]).Color(140, 92, 48)
red = CreateCube().Scale(1.2, 1.8, 1.2).Position([6, 1.1, 8]).Color(210, 70, 55)
blue = CreateCube().Scale(1.4, 1.4, 1.4).Position([-7, 0.9, 5]).Color(50, 110, 210)
ball = CreateSphere(12).Scale(1.3, 1.3, 1.3).Position([2, 1.4, -6]).Color(240, 210, 70)

; Underwater — visible when looking down through refraction
sub = CreateSphere(10).Scale(1.1, 1.1, 1.1).Position([-3, -2.2, 3]).Color(30, 180, 140)
cone = CreateCone().Scale(0.8, 2.2, 0.8).Position([12, 1.3, -3]).Color(80, 170, 90)

boat = CreateCube().Scale(1.5, 0.32, 0.7).Position([0, 0.4, 0]).Color(190, 95, 50)
CreateBuoy(boat)

; Loop
yaw# = 38
pitch# = 16
dist# = 26
frames = 0
While 1
    frames = frames + 1
    dt# = DeltaTime()
    If KeyDown(KEY_A) Then yaw = yaw - 55 * dt
    If KeyDown(KEY_D) Then yaw = yaw + 55 * dt
    If KeyDown(KEY_W) Then dist = dist - 14 * dt
    If KeyDown(KEY_S) Then dist = dist + 14 * dt
    If KeyDown(KEY_Q) Then pitch = pitch + 28 * dt
    If KeyDown(KEY_E) Then pitch = pitch - 28 * dt
    If dist < 8 Then dist = 8
    If dist > 70 Then dist = 70
    If pitch < 4 Then pitch = 4
    If pitch > 70 Then pitch = 70
    If KeyDown(KEY_LEFT) Then
        boat.Position(EntityX(boat) - 8 * dt, EntityY(boat), EntityZ(boat))
    EndIf
    If KeyDown(KEY_RIGHT) Then
        boat.Position(EntityX(boat) + 8 * dt, EntityY(boat), EntityZ(boat))
    EndIf
    cx# = Sin(yaw) * Cos(pitch) * dist
    cz# = Cos(yaw) * Cos(pitch) * dist
    cy# = 2.2 + Sin(pitch) * dist
    cam.Position(cx, cy, cz)
    CameraLookAt(cam, 0, 0.6, 2)
    Text(12, 12, "WASD orbit  QE pitch  arrows boat  Esc quit")
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
