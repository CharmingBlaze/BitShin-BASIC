; Particles 2D — fountain emitter plus Space burst
; Space = burst. Esc quits.

Graphics(800, 600)
SetBuffer(BackBuffer())
SetClsColor(12, 16, 28)
SetWindowTitle("BitShin BASIC — 2D particles")

; Continuous fountain
em = CreateEmitter2D()
PositionEmitter2D(em, 400, 420)
Particle2DRate(em, 70)
Particle2DMax(em, 160)
Particle2DLife(em, 1.6)
Particle2DSpeed(em, 220)
Particle2DSize(em, 7, 2)
Particle2DColor(em, 255, 180, 60, 1, 255, 40, 20, 0)
Particle2DVelocity(em, 0, -1, 0)
Particle2DGravity(em, 0, 280, 0)
Particle2DCone(em, 55)
Particle2DDrag(em, 0.4)

; One-shot burst (rate 0; Emit2D on Space)
burst = CreateEmitter2D()
PositionEmitter2D(burst, 400, 200)
Particle2DRate(burst, 0)
Particle2DMax(burst, 80)
Particle2DLife(burst, 0.8)
Particle2DSpeed(burst, 160)
Particle2DSize(burst, 5, 1)
Particle2DColor(burst, 80, 200, 255, 1, 20, 40, 80, 0)
Particle2DCone(burst, 360)
Particle2DGravity(burst, 0, 40, 0)

; Loop — Space emits burst particles
While Not KeyDown(KEY_ESCAPE)
    If KeyHit(KEY_SPACE) Then Emit2D(burst, 40)
    Cls
    SetColor(230, 230, 240)
    Text(16, 16, "2D particles  |  Space burst  |  Esc quit")
    Flip
Wend
End
