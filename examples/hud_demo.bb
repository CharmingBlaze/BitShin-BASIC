; HUD demo — immediate-mode 2D overlay (Rect, Oval, Line, Text) on a 3D scene.
; Space damages HP (KeyHit 57). Esc quits.

Graphics3D(960, 600)
SetWindowTitle("BitShin BASIC — 2D HUD on 3D")

; Camera / light
camera = CreateCamera().Position([0, 2, -6])
camera.Point(0, 0, 0)

light = CreateLight().Rotate(45, 30, 0)

; World
ground = CreatePlane(20, 20).Position([0, -1, 0]).Color(35, 45, 60)
cube = CreateCube().Position([0, 0.5, 0]).Color(80, 160, 255)
sphere = CreateSphere().Scale(0.4, 0.4, 0.4).Color(255, 180, 40)

hp# = 85.0
maxHp# = 100.0
shield# = 60.0
score = 1250
angle# = 0.0

; Loop
While Not KeyDown(1)
    dt# = DeltaTime() * 60.0
    angle = angle + 1.2 * dt

    cube.Turn(0.6 * dt, 0.9 * dt, 0.3 * dt)
    sphere.Position(Sin(angle) * 2.5, 0.5 + Cos(angle * 2) * 0.4, Cos(angle) * 2.5)

    If KeyHit(57) Then
        hp = hp - 15.0
        If hp < 0.0 Then hp = maxHp
    EndIf

    RenderWorld

    ; Top bar
    Color 15, 20, 30, 200
    Rect 0, 0, 960, 48, 1

    Color 80, 160, 255, 255
    Line 0, 48, 960, 48

    Color 255, 255, 255
    Text 20, 15, "SCORE: " + Str(score) + "   |   SPACE: Damage HP   |   ESC: Quit"

    ; Health bar
    barX = 20
    barY = 65
    barW = 200
    barH = 18

    Color 60, 15, 15, 220
    Rect barX, barY, barW, barH, 1

    fillW = (hp / maxHp) * barW
    If fillW > 0 Then
        Color 50, 220, 90, 230
        Rect barX, barY, fillW, barH, 1
    EndIf

    Color 200, 220, 240, 255
    Rect barX, barY, barW, barH, 0
    Text barX + barW + 12, barY + 2, "HP " + Str(Int(hp)) + " / 100"

    ; Shield bar
    shieldY = 90
    Color 15, 30, 60, 220
    Rect barX, shieldY, barW, 12, 1
    Color 60, 170, 255, 230
    Rect barX, shieldY, (shield / 100.0) * barW, 12, 1
    Color 180, 210, 255, 255
    Rect barX, shieldY, barW, 12, 0
    Text barX + barW + 12, shieldY - 1, "SHIELD 60%"

    ; Radar
    radarX = 860
    radarY = 70
    radarR = 36

    Color 10, 30, 40, 190
    Oval radarX - radarR, radarY - radarR, radarR * 2, radarR * 2

    dotX = radarX + Sin(angle) * (radarR * 0.65)
    dotY = radarY + Cos(angle) * (radarR * 0.65)
    Color 255, 80, 80, 255
    Oval dotX - 4, dotY - 4, 8, 8

    Color 100, 255, 150, 255
    Oval radarX - 3, radarY - 3, 6, 6

    ; Crosshair
    cx = 960 / 2
    cy = 600 / 2
    Color 255, 255, 255, 180
    Line cx - 12, cy, cx - 4, cy
    Line cx + 4, cy, cx + 12, cy
    Line cx, cy - 12, cx, cy - 4
    Line cx, cy + 4, cx, cy + 12
    Oval cx - 1, cy - 1, 2, 2

    Flip
Wend
End
