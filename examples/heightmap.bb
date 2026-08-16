; Generate a heightmap (FBM + thermal + hydraulic), save PNG, mesh it.
; Leave running (no Escape). WASD walk.

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Heightmap")
SetCameraClsColor(88, 138, 198)
SetAmbientLight(48, 58, 68)

cam = CreateCamera()
SetPosition(cam, 0, 16, -20)
CreateSkyBox("default")

sun = CreateDirectionalLight()
SetLightDirection(sun, 48, 32, 0)
SetLightShadow(sun, True)
EnableShadows(True)
SetShadowResolution(1024)

; GenerateHeightmap(w, h, seed [, octaves, scale, persist, lacunarity])
hm = GenerateHeightmap(128, 128, 42, 6, 70, 0.48, 2.15)
ErodeHeightmap(hm, 6, 0.018, 0.22)
HydraulicErodeHeightmap(hm, 280, 20)
SaveHeightmap("assets/heightmap.png", hm)

ground = CreateTerrainFromHeightmap(hm, 80, 80, 14)
SetTerrainStreamRadius(ground, 2)
SetTerrainLOD(ground, 1)

player = CreateCube()
SetScale(player, 0.55, 1.1, 0.55)
SetPosition(player, 0, 10, 0)
SetEntityColor(player, 230, 200, 80)

Print "Heightmap: seed=42  saved assets/heightmap.png  w=" + Str(HeightmapWidth(hm))

yaw# = 0
While 1
    dt# = DeltaTime()
    If KeyDown(KEY_A) Then yaw = yaw - 90 * dt
    If KeyDown(KEY_D) Then yaw = yaw + 90 * dt
    spd# = 10 * dt
    If KeyDown(KEY_W) Then
        SetPosition(player, EntityX(player) + Sin(yaw) * spd, EntityY(player), EntityZ(player) + Cos(yaw) * spd)
    EndIf
    If KeyDown(KEY_S) Then
        SetPosition(player, EntityX(player) - Sin(yaw) * spd, EntityY(player), EntityZ(player) - Cos(yaw) * spd)
    EndIf
    SetStreamOrigin(EntityX(player), 0, EntityZ(player))
    h# = TerrainHeight(EntityX(player), EntityZ(player))
    SetPosition(player, EntityX(player), h + 0.8, EntityZ(player))
    CameraFollow(cam, player, 12, 5, 8, yaw, 12)
    Text(12, 12, "WASD  chunks=" + Str(TerrainChunkCount()) + "  h=" + Str(h))
    RenderWorld
    Flip
Wend
