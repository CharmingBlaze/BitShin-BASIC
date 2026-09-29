; 2D platform — Chipmunk circle on a floor; arrows move, Space jumps.
; Esc or X quits.

SetWindowTitle("BitShin BASIC — 2D platform")
Graphics2D(800, 480)
SetClsColor(18, 22, 34)

; Chipmunk world
Physics2D()
Gravity2D(0, 900)

; Floor
SetColor(60, 70, 90)
floorImg = CreateImage(800, 40)
floor = CreateSprite(floorImg)
SetSpritePosition(floor, 400, 450)
CreateBox2D(floor, 800, 40, 0, 0)

; Player ball
SetColor(80, 190, 255)
ballImg = CreateImage(28, 28)
player = CreateSprite(ballImg)
SetSpritePosition(player, 200, 200)
CreateCircle2D(player, 14, 1, 1)

While Not KeyDown(1) And Not KeyDown(KEY_X)
    dt# = DeltaTime() * 60
    If KeyDown(KEY_LEFT) Then ApplyImpulse2D(player, -25 * dt, 0)
    If KeyDown(KEY_RIGHT) Then ApplyImpulse2D(player, 25 * dt, 0)
    If KeyHit(KEY_SPACE) Then ApplyImpulse2D(player, 0, -420)
    UpdateWorld
    Cls
    SetColor(230, 230, 240)
    Text(16, 16, "Arrows move  |  Space jump  |  Esc/X quit")
    Flip
Wend
End
