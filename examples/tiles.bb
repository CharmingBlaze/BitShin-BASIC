; Tiles — CreateTileMap / SetTile / DrawTileMap
; Esc quits.

Graphics2D(640, 400)
SetClsColor(18, 22, 34)
SetWindowTitle("BitShin BASIC — tiles")

; Atlas placeholder (DrawTileMap without atlas uses solid colors; ids 1..4)
SetColor(70, 120, 70)
atlas = CreateImage(64, 16)

; Fill a 20x12 map: grass, props, and a ground row
map = CreateTileMap(16, 16, 20, 12)
For y = 0 To 11
    For x = 0 To 19
        t = 1
        If y = 11 Then t = 3
        If y = 10 And (x Mod 5) = 0 Then t = 2
        SetTile(map, x, y, t)
    Next
Next

; Loop — draw the tilemap
While Not KeyDown(KEY_ESCAPE)
    Cls
    DrawTileMap(map, 0, 80)
    SetColor(230, 230, 240)
    Text(12, 12, "CreateTileMap / SetTile / DrawTileMap  |  Esc")
    Flip
Wend
End
