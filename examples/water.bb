; Scenic water (DuDv + Fresnel + planar reflect/refract + specular). Esc quits.

Graphics3D(1100, 700)
SetWindowTitle("Pretty Water")
SetCameraClsColor(72, 128, 178)
SetAmbientLight(38, 48, 62)

cam = CreateCamera()
CreateSkyBox("default")

sun = CreateDirectionalLight()
SetLightDirection(sun, 28, 42, 8)
SetLightColor(sun, 255, 236, 200)

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

; Shore strip so the reflection FBO has something obvious to grab.
isle = CreateProcTerrain(5, 11, 4, 5, 22)
SetTerrainStreamRadius(isle, 1)

pier = CreateCube()
SetScale(pier, 8, 0.18, 1.2)
SetPosition(pier, 10, 0.35, 4)
SetEntityColor(pier, 140, 92, 48)

red = CreateCube()
SetScale(red, 1.2, 1.8, 1.2)
SetPosition(red, 6, 1.1, 8)
SetEntityColor(red, 210, 70, 55)

blue = CreateCube()
SetScale(blue, 1.4, 1.4, 1.4)
SetPosition(blue, -7, 0.9, 5)
SetEntityColor(blue, 50, 110, 210)

ball = CreateSphere(12)
SetScale(ball, 1.3, 1.3, 1.3)
SetPosition(ball, 2, 1.4, -6)
SetEntityColor(ball, 240, 210, 70)

; Underwater so refraction is visible when looking down.
sub = CreateSphere(10)
SetScale(sub, 1.1, 1.1, 1.1)
SetPosition(sub, -3, -2.2, 3)
SetEntityColor(sub, 30, 180, 140)

cone = CreateCone()
SetScale(cone, 0.8, 2.2, 0.8)
SetPosition(cone, 12, 1.3, -3)
SetEntityColor(cone, 80, 170, 90)

boat = CreateCube()
SetScale(boat, 1.5, 0.32, 0.7)
SetPosition(boat, 0, 0.4, 0)
SetEntityColor(boat, 190, 95, 50)
CreateBuoy(boat)

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
        SetPosition(boat, EntityX(boat) - 8 * dt, EntityY(boat), EntityZ(boat))
    EndIf
    If KeyDown(KEY_RIGHT) Then
        SetPosition(boat, EntityX(boat) + 8 * dt, EntityY(boat), EntityZ(boat))
    EndIf
    cx# = Sin(yaw) * Cos(pitch) * dist
    cz# = Cos(yaw) * Cos(pitch) * dist
    cy# = 2.2 + Sin(pitch) * dist
    SetPosition(cam, cx, cy, cz)
    CameraLookAt(cam, 0, 0.6, 2)
    Text(12, 12, "WASD orbit  QE pitch  arrows boat  Esc quit")
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
