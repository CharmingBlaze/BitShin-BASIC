; Combined ocean: Gerstner swell + chop + Fresnel / planar reflect. Esc quits.

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Ocean")
SetCameraClsColor(48, 92, 138)
SetAmbientLight(32, 42, 55)

cam = CreateCamera()
SetCameraRange(cam, 0.45, 4000)
CreateSkyBox("default")

sun = CreateDirectionalLight()
SetLightDirection(sun, 28, 36, 8)
SetLightColor(sun, 255, 236, 200)

water = CreateWater(520, 520, 96)
SetWaterStyle("ocean")
SetWaterColor(6, 48, 72)
SetWaterWaves(4, 0.95)
SetGerstner(0, 0.88, 0.32, 0.36, 1.25, 56, 0.78)
SetGerstner(1, -0.22, 0.97, 0.28, 0.58, 26, 1.12)
SetGerstner(2, 0.62, -0.70, 0.22, 0.22, 11, 1.72)
SetGerstner(3, 0.18, 0.98, 0.16, 0.08, 3.8, 2.45)
SetWaterWind(0.85, 0.28, 0.72)
SetWaterSpeed(0.04)
SetWaterWaveStrength(0.04)
EnableWaterReflection(True)
EnableWaterRefraction(True)
SetWaterFollow(True)
SetWaterLevel(0)
SetWaterCaustics(water, True)
SetWaterSSR(water, True)
SetWaterAmbientSound(water, "examples/assets/ocean.ogg", GetWeatherIntensity())

isle = CreateProcTerrain(4, 11, 4, 4.5, 24)
SetTerrainStreamRadius(isle, 1)

red = CreateCube()
SetScale(red, 1.3, 1.8, 1.3)
SetPosition(red, 7, 1.2, 6)
SetEntityColor(red, 210, 65, 50)
CreateBuoy(red)

ball = CreateSphere(12)
SetScale(ball, 1.2, 1.2, 1.2)
SetPosition(ball, -5, 1.3, 4)
SetEntityColor(ball, 240, 200, 60)
CreateBuoy(ball)

boat = CreateCube()
SetScale(boat, 1.6, 0.3, 0.7)
SetPosition(boat, 1, 0.5, -2)
SetEntityColor(boat, 185, 95, 48)
CreateBuoy(boat)

yaw# = 28
pitch# = 10
dist# = 38
frames = 0
While 1
    frames = frames + 1
    dt# = DeltaTime()
    If KeyDown(KEY_A) Then yaw = yaw - 52 * dt
    If KeyDown(KEY_D) Then yaw = yaw + 52 * dt
    If KeyDown(KEY_W) Then dist = dist - 14 * dt
    If KeyDown(KEY_S) Then dist = dist + 14 * dt
    If KeyDown(KEY_Q) Then pitch = pitch + 26 * dt
    If KeyDown(KEY_E) Then pitch = pitch - 26 * dt
    If dist < 12 Then dist = 12
    If dist > 90 Then dist = 90
    If pitch < 4 Then pitch = 4
    If pitch > 55 Then pitch = 55
    cx# = Sin(yaw) * Cos(pitch) * dist
    cz# = Cos(yaw) * Cos(pitch) * dist
    cy# = 3.2 + Sin(pitch) * dist
    SetPosition(cam, cx, cy, cz)
    CameraLookAt(cam, 0, WaterHeight(0, 0) + 0.2, 18)
    Text(12, 12, "OCEAN  WASD orbit  QE pitch  Esc quit")
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
