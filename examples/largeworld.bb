; Large world — streaming terrain, instanced trees, water, and ImGui stats.
; WASD walk. Esc quits.

Graphics3D(1200, 720)
SetWindowTitle("BitShin BASIC — Large world")
SetCameraClsColor(80, 130, 190)
SetAmbientLight(45, 55, 68)
CreateSkyBox("default")

; Camera / light
cam = CreateCamera()
sun = CreateDirectionalLight()
SetLightDirection(sun, 48, 30, 0)
SetLightShadow(sun, True)
EnableShadows(True)
SetShadowResolution(1024)
SetShadowFilter("pcf")

; Terrain / water
land = CreateProcTerrain(11, 15, 5, 8, 36)
SetTerrainStreamRadius(land, 2)
CreateWorldStream(36, 1)
water = CreateWater(120, 120, 40)
SetWaterLevel(-1.2)
SetWaterColor(16, 64, 88)
EnableWaterReflection(True)
SetProbeGrid(-30, 8, -30, 2, 1, 2, 30)

; Instanced trees
proto = CreateCone().Hide()
trees = CreateInstancedMesh(proto, 80)
For i = 0 To 79
    ang# = i * 17
    rx# = Sin(ang#) * (12 + i * 0.35)
    rz# = Cos(ang#) * (12 + i * 0.35)
    SetInstanceTransform(trees, i, rx#, TerrainHeight(rx#, rz#) + 1.1, rz#, 0, ang#, 0, 0.5, 2.0, 0.5)
Next
BatchInstances(trees)

player = CreateCube().Scale(0.55, 1.1, 0.55).Position([8, 6, 8]).Color(255, 210, 70)
SetStreamFollow(player)

; Loop
yaw# = 30
While Not KeyDown(1)
    dt# = DeltaTime()
    If KeyDown(KEY_A) Then yaw = yaw - 85 * dt
    If KeyDown(KEY_D) Then yaw = yaw + 85 * dt
    If KeyDown(KEY_W) Then player.Position(EntityX(player) + Sin(yaw) * 11 * dt, EntityY(player), EntityZ(player) + Cos(yaw) * 11 * dt)
    If KeyDown(KEY_S) Then player.Position(EntityX(player) - Sin(yaw) * 8 * dt, EntityY(player), EntityZ(player) - Cos(yaw) * 8 * dt)
    player.Position(EntityX(player), TerrainHeight(EntityX(player), EntityZ(player)) + 0.75, EntityZ(player))
    cam.Follow(player, 13, 5.5, 8, yaw, 14)

    GuiBegin("Scale")
    GuiText("fps " + Str(Int(StatsFPS())) + "  draws " + Str(StatsDraws()))
    GuiText("chunks " + Str(StatsChunks()) + "  jobs " + Str(StatsJobs()))
    GuiText("ents " + Str(EntityCount()) + "  water " + Str(WaterHeight(EntityX(player), EntityZ(player))))
    GuiEnd()

    RenderWorld
    Flip
Wend
End
