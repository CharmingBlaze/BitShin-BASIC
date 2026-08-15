; Streaming terrain + instanced trees + water + crowd-ish walkers + ImGui stats.

Graphics3D(1200, 720)
SetWindowTitle("Large world")
SetCameraClsColor(80, 130, 190)
SetAmbientLight(45, 55, 68)
CreateSkyBox("default")

cam = CreateCamera()
sun = CreateDirectionalLight()
SetLightDirection(sun, 48, 30, 0)
SetLightShadow(sun, True)
EnableShadows(True)
SetShadowResolution(1024)
SetShadowFilter("pcf")

land = CreateProcTerrain(11, 15, 5, 8, 36)
SetTerrainStreamRadius(land, 2)
CreateWorldStream(36, 1)
water = CreateWater(120, 120, 40)
SetWaterLevel(-1.2)
SetWaterColor(16, 64, 88)
EnableWaterReflection(True)
SetProbeGrid(-30, 8, -30, 2, 1, 2, 30)

proto = CreateCone()
HideEntity(proto)
trees = CreateInstancedMesh(proto, 80)
For i = 0 To 79
    ang# = i * 17
    rx# = Sin(ang#) * (12 + i * 0.35)
    rz# = Cos(ang#) * (12 + i * 0.35)
    SetInstanceTransform(trees, i, rx#, TerrainHeight(rx#, rz#) + 1.1, rz#, 0, ang#, 0, 0.5, 2.0, 0.5)
Next
BatchInstances(trees)

player = CreateCube()
SetScale(player, 0.55, 1.1, 0.55)
SetPosition(player, 8, 6, 8)
SetEntityColor(player, 255, 210, 70)
SetStreamFollow(player)

yaw# = 30
While Not KeyDown(1)
    dt# = DeltaTime()
    If KeyDown(KEY_A) Then yaw = yaw - 85 * dt
    If KeyDown(KEY_D) Then yaw = yaw + 85 * dt
    If KeyDown(KEY_W) Then SetPosition(player, EntityX(player) + Sin(yaw) * 11 * dt, EntityY(player), EntityZ(player) + Cos(yaw) * 11 * dt)
    If KeyDown(KEY_S) Then SetPosition(player, EntityX(player) - Sin(yaw) * 8 * dt, EntityY(player), EntityZ(player) - Cos(yaw) * 8 * dt)
    SetPosition(player, EntityX(player), TerrainHeight(EntityX(player), EntityZ(player)) + 0.75, EntityZ(player))
    CameraFollow(cam, player, 13, 5.5, 8, yaw, 14)

    GuiBegin("Scale")
    GuiText("fps " + Str(Int(StatsFPS())) + "  draws " + Str(StatsDraws()))
    GuiText("chunks " + Str(StatsChunks()) + "  jobs " + Str(StatsJobs()))
    GuiText("ents " + Str(EntityCount()) + "  water " + Str(WaterHeight(EntityX(player), EntityZ(player))))
    GuiEnd()

    RenderWorld
    Flip
Wend
End
