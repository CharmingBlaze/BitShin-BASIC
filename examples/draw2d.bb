; Draw 2D — bouncing oval, RectsOverlap, and DeltaTime
; Esc quits.

Graphics(640, 480)
SetBuffer(BackBuffer())
SetClsColor(18, 22, 34)
SetWindowTitle("BitShin BASIC — 2D")

; Moving oval state
x# = 40
vx# = 3.2

; Loop — bounce, draw Rect + Oval, highlight on overlap
While Not KeyDown(1)
    dt# = DeltaTime() * 60
    x = x + vx * dt
    If x > 580 Or x < 20 Then vx = -vx

    Cls
    SetColor(255, 90, 90)
    Rect(280, 180, 80, 80, 1)
    SetColor(80, 180, 255)
    If RectsOverlap(x, 200, 40, 40, 280, 180, 80, 80) Then SetColor(255, 230, 80)
    Oval(x, 200, 40, 40)
    SetColor(230, 230, 240)
    Text(16, 16, "DeltaTime  |  RectsOverlap  |  Esc quit")
    Flip
Wend
End
