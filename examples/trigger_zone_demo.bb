; Trigger zone — walk the cube into the translucent sensor; it turns gold inside.
; WASD / arrows move. Esc quits.

Graphics3D 960, 600
SetWindowTitle("BitShin BASIC — Trigger Zones & 3D Audio")

; Camera
camera = CreateCamera()
camera.Position([0, 7, -12])
camera.Point(0, 0, 0)

; Light
light = CreateDirectionalLight()
SetLightDirection(light, 50, 40, 0)
SetAmbientLight(60, 70, 85)

; Ground floor
ground = CreatePlane(30, 30).Position([0, 0, 0]).Color(40, 50, 65)
CreateBodyBox(ground, 15, 0.1, 15, 0)

; Player cube (WASD)
player = CreateCube().Scale(0.6, 0.6, 0.6).Position([-4, 0.6, 0]).Color(70, 180, 255)
CreateBodyBox(player, 0.6, 0.6, 0.6, 1)

; Trigger zone (transparent box + kinematic sensor)
zone = CreateCube().Scale(2.0, 1.5, 2.0).Position([3.0, 1.5, 0]).Color(100, 150, 255).Alpha(0.4)
CreateSensor(zone, 3.0, 1.5, 0, 2.0, 1.5, 2.0, 2)

px# = -4.0
pz# = 0.0
zoneActive = 0
activeTimer# = 0.0

While Not KeyDown(1)
    dt# = DeltaTime() * 60.0
    spd# = 0.12 * dt

    ; Move player with WASD / arrows
    If KeyDown(30) Or KeyDown(203) Then px = px - spd
    If KeyDown(32) Or KeyDown(205) Then px = px + spd
    If KeyDown(17) Or KeyDown(200) Then pz = pz + spd
    If KeyDown(31) Or KeyDown(208) Then pz = pz - spd

    SetPosition(player, px, 0.6, pz)

    UpdateWorld

    ; Check if player is inside the trigger zone
    inTrigger = CheckTrigger(zone, player)

    If inTrigger Then
        SetEntityColor(zone, 255, 200, 50)
        SetEntityAlpha(zone, 0.65)
        zoneActive = 1
        activeTimer = activeTimer + dt
    Else
        SetEntityColor(zone, 80, 130, 240)
        SetEntityAlpha(zone, 0.35)
        zoneActive = 0
    EndIf

    RenderWorld

    ; HUD overlay
    Color 15, 20, 30, 210
    Rect 20, 20, 420, 85, 1
    If zoneActive Then
        Color 255, 200, 50, 255
    Else
        Color 70, 140, 230, 255
    EndIf
    Rect 20, 20, 420, 85, 0

    Color 255, 255, 255
    Text 35, 32, "PHYSICS TRIGGER ZONE DEMO"
    If zoneActive Then
        Color 255, 220, 80
        Text 35, 52, "STATUS: [ INSIDE TRIGGER ZONE! ]"
    Else
        Color 180, 200, 220
        Text 35, 52, "STATUS: Outside zone"
    EndIf
    Color 140, 170, 200
    Text 35, 72, "Use WASD to move into the glowing box  |  ESC to quit"

    Flip
Wend
End
