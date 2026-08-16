; GPU Gems-style Gerstner ocean (swell + chop + wind). Esc quits.

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Gerstner ocean")
SetCameraClsColor(46, 88, 132)
SetAmbientLight(30, 40, 54)

cam = CreateCamera()
SetCameraRange(cam, 0.45, 4000)
CreateSkyBox("default")

sun = CreateDirectionalLight()
SetLightDirection(sun, 30, 34, 6)
SetLightColor(sun, 255, 232, 196)

water = CreateWater(500, 500, 96)
SetWaterStyle("gerstner")
SetWaterColor(5, 44, 68)
SetWaterWaves(4, 1.05)
SetGerstner(0, 0.90, 0.30, 0.34, 1.35, 58, 0.76)
SetGerstner(1, -0.20, 0.98, 0.28, 0.62, 24, 1.15)
SetGerstner(2, 0.58, -0.74, 0.22, 0.24, 10, 1.78)
SetGerstner(3, 0.16, 0.99, 0.15, 0.09, 3.5, 2.5)
SetWaterWind(0.92, 0.22, 0.78)
SetWaterFollow(True)
SetWaterLevel(0)
EnableWaterReflection(True)
EnableWaterRefraction(True)

buoy = CreateSphere(10)
SetScale(buoy, 0.7, 0.7, 0.7)
SetPosition(buoy, 3, 1, 4)
SetEntityColor(buoy, 220, 70, 50)
CreateBuoy(buoy)

crate = CreateCube()
SetScale(crate, 1.1, 1.1, 1.1)
SetPosition(crate, -6, 1.2, 7)
SetEntityColor(crate, 180, 140, 70)

pole = CreateCylinder()
SetScale(pole, 0.25, 3.2, 0.25)
SetPosition(pole, 8, 1.6, -4)
SetEntityColor(pole, 90, 90, 95)

yaw# = 32
pitch# = 11
dist# = 40
frames = 0
While 1
    frames = frames + 1
    dt# = DeltaTime()
    If KeyDown(KEY_A) Then yaw = yaw - 50 * dt
    If KeyDown(KEY_D) Then yaw = yaw + 50 * dt
    If KeyDown(KEY_W) Then dist = dist - 16 * dt
    If KeyDown(KEY_S) Then dist = dist + 16 * dt
    If KeyDown(KEY_Q) Then pitch = pitch + 24 * dt
    If KeyDown(KEY_E) Then pitch = pitch - 24 * dt
    If dist < 12 Then dist = 12
    If dist > 100 Then dist = 100
    If pitch < 5 Then pitch = 5
    If pitch > 52 Then pitch = 52
    cx# = Sin(yaw) * Cos(pitch) * dist
    cz# = Cos(yaw) * Cos(pitch) * dist
    cy# = 3.4 + Sin(pitch) * dist
    SetPosition(cam, cx, cy, cz)
    wh# = WaterHeight(0, 0)
    CameraLookAt(cam, 0, wh + 0.15, 22)
    Text(12, 12, "OCEAN Gerstner  WASD orbit  QE pitch  y=" + Str(wh))
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
