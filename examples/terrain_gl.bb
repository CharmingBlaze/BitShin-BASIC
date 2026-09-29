; Terrain-GL — value-noise hills with splat and streamed LOD (OpenGL port).
; WASD walk, mouse/arrows look, Space fly, Q/E height in fly. Esc after a few frames.

Graphics3D(1280, 720)
SetWindowTitle("Terrain-OpenGL (BitShin BASIC)")
SetCameraClsColor(128, 153, 179)
SetAmbientLight(60, 68, 78)
EnableFog(True)
SetFog(128, 153, 179, 80, 280)
HidePointer()
SetMousePosition(640, 360)
Color(255, 245, 220)

; Sky
CreateSkyBox("default")
SetSkyPreset("default")

; Light / shadows
sun = CreateDirectionalLight()
SetLightColor(sun, 255, 255, 230)
SetLightDirection(sun, 40, 45, 8)
SetLightShadow(sun, True)
EnableShadows(True)
SetShadowResolution(1024)

; Terrain
ground = CreateTerrainGL(3, 25, 3, 18, 0.042, 6, 48)
ApplyTerrainSplat(ground)
SetTerrainLOD(ground, 1)
SetTerrainGrassCoverage(ground, 0.65)
SetTerrainWaterHeight(ground, 4.5)
SetTerrainSnow(True, 14)
SetTerrainStreamRadius(ground, 3)

; Water
water = CreateWater(48, 48, 20)
SetWaterLevel(4.5)
SetWaterColor(40, 110, 140)
EnableWaterReflection(True)
SetWaterFollow(False)
SetWaterWaves(2, 0.08)

; Camera
cam = CreateCamera()
SetCameraRange(cam, 0.25, 4000)
cam.Position(8, TerrainHeight(8, -16) + 2.0, -16)
CameraLookAt(cam, 0, TerrainHeight(0, 0) + 2, 8)

yaw# = 0
pitch# = -10
fly = 0
frames = 0

; Loop
While 1
    frames = frames + 1
    dt# = DeltaTime()
    yaw = yaw + MouseDeltaX() * 0.14
    pitch = pitch - MouseDeltaY() * 0.14
    If KeyDown(KEY_LEFT) Then yaw = yaw - 70 * dt
    If KeyDown(KEY_RIGHT) Then yaw = yaw + 70 * dt
    If KeyDown(KEY_UP) Then pitch = pitch + 40 * dt
    If KeyDown(KEY_DOWN) Then pitch = pitch - 40 * dt
    If pitch > 80 Then pitch = 80
    If pitch < -80 Then pitch = -80
    If KeyHit(KEY_SPACE) Then fly = 1 - fly
    cam.Rotate(pitch, yaw, 0)
    SetMousePosition(640, 360)

    spd# = 16 * dt
    If KeyDown(KEY_W) Then cam.Move(0, 0, spd)
    If KeyDown(KEY_S) Then cam.Move(0, 0, -spd)
    If KeyDown(KEY_A) Then cam.Move(-spd, 0, 0)
    If KeyDown(KEY_D) Then cam.Move(spd, 0, 0)
    If fly Then
        If KeyDown(KEY_Q) Then cam.Position(EntityX(cam), EntityY(cam) + 12 * dt, EntityZ(cam))
        If KeyDown(KEY_E) Then cam.Position(EntityX(cam), EntityY(cam) - 12 * dt, EntityZ(cam))
    Else
        h# = TerrainHeight(EntityX(cam), EntityZ(cam))
        cam.Position(EntityX(cam), h + 1.7, EntityZ(cam))
    EndIf
    SetStreamOrigin(EntityX(cam), 0, EntityZ(cam))
    Cls
    Text(12, 12, "TERRAIN-GL  WASD walk  mouse/arrows look  Space fly  Esc quit")
    Text(12, 32, "chunks=" + Str(TerrainChunkCount()) + "  h=" + Str(TerrainHeight(EntityX(cam), EntityZ(cam))) + "  fps=" + Str(StatsFPS()))
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
