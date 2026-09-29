; Geo — projected GeoJSON path on a geo-bounded heightmap around Midtown.
; WASD walk. Esc quits.

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Geo")
SetCameraClsColor(90, 140, 200)
SetAmbientLight(50, 60, 70)

; Camera / sky
cam = CreateCamera().Position([0, 40, -70])
CreateSkyBox("default")

; Light / shadows
sun = CreateDirectionalLight()
SetLightDirection(sun, 50, 35, 0)
SetLightShadow(sun, True)
EnableShadows(True)
SetShadowResolution(1024)

; World — (0,0) is this WGS84 point; stream chunks stay around the origin
SetGeoOrigin(-73.9857, 40.7484)
CreateWorldStream(24, 2)

hm = GenerateHeightmap(64, 64, 11, 4, 90, 0.4, 2)
ground = CreateTerrainFromGeoDEM(hm, -73.9877, 40.7464, -73.9837, 40.7504, 8)
SetTerrainStreamRadius(ground, 2)
SetTerrainLOD(ground, 1)

path = LoadGeoJSON("assets/midtown_walk.geojson")

player = CreateCube().Scale(0.7, 1.4, 0.7).Position([0, 8, 0]).Color(230, 200, 80)

tx = GeoTileX(-73.9857, 40.7484, 15)
ty = GeoTileY()
tz = GeoTileZ()

; Loop
yaw# = 0
While Not KeyDown(1)
    dt# = DeltaTime()
    If KeyDown(KEY_A) Then yaw = yaw - 90 * dt
    If KeyDown(KEY_D) Then yaw = yaw + 90 * dt
    spd# = 12 * dt
    If KeyDown(KEY_W) Then
        player.Position(EntityX(player) + Sin(yaw) * spd, EntityY(player), EntityZ(player) + Cos(yaw) * spd)
    EndIf
    If KeyDown(KEY_S) Then
        player.Position(EntityX(player) - Sin(yaw) * spd, EntityY(player), EntityZ(player) - Cos(yaw) * spd)
    EndIf
    SetStreamOrigin(EntityX(player), 0, EntityZ(player))
    h# = TerrainHeight(EntityX(player), EntityZ(player))
    player.Position(EntityX(player), h + 0.9, EntityZ(player))
    lon# = GeoUnproject(EntityX(player), EntityZ(player))
    lat# = GeoUnprojectLat()
    cam.Follow(player, 18, 8, 10, yaw, 14)
    Text(12, 12, "WASD  lon=" + Str(lon) + " lat=" + Str(lat) + "  tile=" + Str(tx) + "/" + Str(ty) + "/" + Str(tz))
    Text(12, 32, "features=" + Str(GeoJSONCount()) + "  h=" + Str(h))
    RenderWorld
    Flip
Wend
End
