; Real XZ heightfield: alpine FBM hills, splat, walk. Esc after a few frames.

Graphics3D(1280, 720)
SetWindowTitle("TERRAIN")
SetCameraClsColor(110, 160, 210)
SetAmbientLight(88, 92, 100)
EnableFog(True)
SetFog(145, 168, 188, 150, 800)
EnableHeightFog(False)
SetFogHeight(0, 0)
HidePointer()
SetMousePosition(640, 360)
Color(255, 245, 220)

cam = CreateCamera()
SetCameraRange(cam, 0.25, 4000)
CreateSkyBox("default")
SetSkyPreset("default")

sun = CreateDirectionalLight()
SetLightDirection(sun, 42, 48, 12)
SetLightColor(sun, 255, 220, 175)
SetLightShadow(sun, True)
EnableShadows(True)
SetShadowResolution(1024)

; CPU heightmap → chunked XZ mesh (G3N NewGeometry). Strong height scale.
hm = GenerateHeightmapPreset("alpine", 160, 160, 19)
ground = CreateTerrainFromHeightmap(hm, 220, 220, 28)
ApplyTerrainSplat(ground)
SetTerrainGrassCoverage(ground, 0.68)
SetTerrainWaterHeight(ground, 5.5)
SetTerrainSnow(True, 19)
SetTerrainFogFalloff(0)
SetTerrainStreamRadius(ground, 3)
SetTerrainLOD(ground, 1)
SetStreamOrigin(0, 0, 0)

; Small lake in the low bowl — do not touch ocean / mbwater shaders.
water = CreateWater(36, 36, 16)
SetWaterLevel(5.5)
SetWaterColor(22, 78, 102)
EnableWaterReflection(True)
SetWaterFollow(False)
SetWaterWaves(2, 0.05)

yaw# = 18
pitch# = -12
fly = 0
frames = 0
SetPosition(cam, 12, TerrainHeight(12, 18) + 1.8, 18)
SetRotation(cam, pitch, yaw, 0)

While 1
    frames = frames + 1
    dt# = DeltaTime()
    yaw = yaw + MouseDeltaX() * 0.14
    pitch = pitch - MouseDeltaY() * 0.14
    If KeyDown(KEY_LEFT) Then yaw = yaw - 80 * dt
    If KeyDown(KEY_RIGHT) Then yaw = yaw + 80 * dt
    If KeyDown(KEY_UP) Then pitch = pitch + 50 * dt
    If KeyDown(KEY_DOWN) Then pitch = pitch - 50 * dt
    If pitch > 80 Then pitch = 80
    If pitch < -80 Then pitch = -80
    If KeyHit(KEY_SPACE) Then fly = 1 - fly
    SetRotation(cam, pitch, yaw, 0)
    SetMousePosition(640, 360)

    spd# = 14 * dt
    If KeyDown(KEY_W) Then MoveEntity(cam, 0, 0, spd)
    If KeyDown(KEY_S) Then MoveEntity(cam, 0, 0, -spd)
    If KeyDown(KEY_A) Then MoveEntity(cam, -spd, 0, 0)
    If KeyDown(KEY_D) Then MoveEntity(cam, spd, 0, 0)

    SetStreamOrigin(EntityX(cam), 0, EntityZ(cam))
    h# = TerrainHeight(EntityX(cam), EntityZ(cam))
    If fly Then
        If KeyDown(KEY_Q) Then SetPosition(cam, EntityX(cam), EntityY(cam) + 12 * dt, EntityZ(cam))
        If KeyDown(KEY_E) Then SetPosition(cam, EntityX(cam), EntityY(cam) - 12 * dt, EntityZ(cam))
    Else
        SetPosition(cam, EntityX(cam), h + 1.75, EntityZ(cam))
    EndIf

    Cls
    Text(12, 12, "TERRAIN  WASD walk  mouse/arrows look  Space fly  Q/E  Esc quit")
    Text(12, 32, "chunks=" + Str(TerrainChunkCount()) + "  h=" + Str(h) + "  fps=" + Str(StatsFPS()))
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
