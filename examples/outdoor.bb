; Outdoor — walkable hills, cabin rooms, dialogue, key item, and a wolf actor.
; WASD walk, mouse / arrows look, E use, [ ] time of day, F5 save, F9 load.
; Esc quits after the window has presented a few frames.

Graphics3D(1280, 720)
SetWindowTitle("BitShin BASIC — Outdoor")
SetCameraClsColor(110, 160, 210)
SetAmbientLight(88, 92, 100)
EnableFog(True)
SetFog(145, 168, 188, 40, 280)
HidePointer()
SetMousePosition(640, 360)
Color(255, 245, 220)

; Camera / sky / light
cam = CreateCamera()
SetCameraRange(cam, 0.25, 2000)
CreateSkyBox("default")
sun = CreateDirectionalLight()
SetLightColor(sun, 255, 220, 175)
SetLightShadow(sun, True)
EnableShadows(True)
SetShadowResolution(1024)
SetTimeOfDay(15)
SetPlayer(cam)

; Terrain
hm = GenerateHeightmapPreset("rolling-hills", 96, 96, 7)
ground = CreateTerrainFromHeightmap(hm, 140, 140, 18)
ApplyTerrainSplat(ground)
SetTerrainGrassCoverage(ground, 0.7)
SetTerrainWaterHeight(ground, 2)
SetTerrainSnow(True, 16)
SetTerrainStreamRadius(ground, 2)
SetTerrainLOD(ground, 1)
SetTerrainDetail(ground, 8)
SetStreamOrigin(0, 0, 0)

; Scatter templates (hidden) — copies land on loaded chunks
tree = CreateCone(0.55, 2.4, 8).Color(36, 110, 48).Hide()
TerrainTrees(ground, tree, 0.7)

rock = CreateCube().Scale(0.45, 0.28, 0.45).Color(120, 112, 100).Hide()
TerrainProps(ground, rock, 0.45)
TerrainFoliage(ground, 1)

; Cabin — shell stays visible outdoors; furniture is room-tagged
hx# = 16
hz# = 22
hy# = TerrainHeight(hx, hz)

hall = CreateRoom()
kitchen = CreateRoom()
RoomLink(hall, kitchen)
RoomAmbient(hall, 42, 36, 28)
RoomAmbient(kitchen, 28, 36, 48)

floorH = CreateCube().Scale(2.2, 0.08, 1.6).Position([hx, hy + 0.08, hz]).Color(92, 70, 48)
wall = CreateCube().Scale(2.2, 1.3, 0.08).Position([hx, hy + 1.3, hz + 1.6]).Color(128, 96, 64)

door = CreateCube().Scale(0.08, 1.1, 0.55).Position([hx + 2.1, hy + 1.1, hz]).Color(150, 96, 52)
CreateDoor(door, hall, kitchen)

table = CreateCube().Scale(0.5, 0.35, 0.5).Position([hx - 0.4, hy + 0.4, hz]).Color(96, 64, 40)
SetRoom(table, hall)

lamp = CreatePointLight().Position([hx, hy + 2.2, hz])
SetLightColor(lamp, 255, 196, 120)
SetRoom(lamp, hall)

pot = CreateSphere(10).Scale(0.22, 0.18, 0.22).Position([hx + 3.2, hy + 0.3, hz + 0.4]).Color(70, 110, 140)
SetRoom(pot, kitchen)

npc = CreateCube().Scale(0.35, 0.9, 0.35).Position([hx + 0.6, hy + 0.95, hz - 0.4]).Color(220, 180, 120)
SetRoom(npc, hall)
SetDialogue(npc, "The key is outside.|Bring it back and the wolf will still chase you.")

; Items / actors
key = CreateCube().Scale(0.12, 0.12, 0.28).Position([8, TerrainHeight(8, 10) + 0.2, 10]).Color(230, 190, 40)
SetItem(key, "key")

wolf = CreateCube().Scale(0.55, 0.4, 0.9).Position([28, TerrainHeight(28, 24) + 0.4, 24]).Color(70, 68, 74)
CreateActor(wolf, 11, 4)

; Start pose
yaw# = 40
pitch# = -8
frames = 0
cam.Position(6, TerrainHeight(6, 8) + 1.75, 8)
cam.Rotate(pitch, yaw, 0)

; Loop
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
    cam.Rotate(pitch, yaw, 0)
    SetMousePosition(640, 360)

    spd# = 8 * dt
    If KeyDown(KEY_W) Then cam.Move(0, 0, spd)
    If KeyDown(KEY_S) Then cam.Move(0, 0, -spd)
    If KeyDown(KEY_A) Then cam.Move(-spd, 0, 0)
    If KeyDown(KEY_D) Then cam.Move(spd, 0, 0)
    SetStreamOrigin(EntityX(cam), 0, EntityZ(cam))
    h# = TerrainHeight(EntityX(cam), EntityZ(cam))
    cam.Position(EntityX(cam), h + 1.75, EntityZ(cam))

    hour# = GetTimeOfDay()
    If KeyDown(KEY_LBRACKET) Then hour = hour - 4 * dt
    If KeyDown(KEY_RBRACKET) Then hour = hour + 4 * dt
    If hour < 0 Then hour = hour + 24
    If hour >= 24 Then hour = hour - 24
    SetTimeOfDay(hour)

    If KeyHit(KEY_E) Then
        If DialogueOn() Then
            DialogueAdvance()
        Else
            CameraPick(cam, 640, 360)
            hit = PickedEntity()
            Use(hit)
            If hit = door Then
                If CurrentRoom() = 0 Then
                    EnterRoom(hall)
                Else
                    EnterRoom(0)
                EndIf
            EndIf
        EndIf
    EndIf
    If KeyHit(KEY_F5) Then SaveGame("outdoor.json")
    If KeyHit(KEY_F9) Then LoadGame("outdoor.json")

    line$ = ""
    If DialogueOn() Then line$ = DialogueLine$()
    have = 0
    If InventoryHas("key") Then have = 1

    Cls
    Text(12, 12, "WASD walk   E use   [ ] time   F5 save   F9 load   Esc quit")
    Text(12, 32, "hour " + Str(hour) + "   room " + Str(CurrentRoom()) + "   wolf " + Str(ActorState(wolf)) + "   key " + Str(have))
    Text(12, 52, line$)
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
