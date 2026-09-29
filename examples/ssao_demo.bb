; SSAO — colonnade and steps showing contact crevices and ambient occlusion.
; Space toggles SSAO on/off (KeyHit 57). Esc quits.

Graphics3D(1024, 640)
SetWindowTitle("BitShin BASIC — Screen-Space Ambient Occlusion (SSAO)")

; Camera / light
camera = CreateCamera().Position([0, 4, -9])
camera.Point(0, 1.2, 0)

light = CreateDirectionalLight()
SetLightDirection(light, 45, 45, 0)
SetAmbientLight(60, 65, 75)

; PostFX / SSAO
EnablePostFX(1)
EnableSSAO(1)
SetSSAORadius(0.8)
SetSSAOIntensity(2.0)

; Floor
floorPlane = CreatePlane(30, 30).Position([0, 0, 0]).Color(180, 185, 195)

; Colonnade — pillars, capitals, pedestals
For i = -3 To 3
    col = CreateCylinder().Scale(0.45, 2.5, 0.45).Position([Float(i) * 2.2, 2.5, 2.5]).Color(215, 215, 220)
    cap = CreateCube().Scale(0.6, 0.2, 0.6).Position([Float(i) * 2.2, 5.1, 2.5]).Color(230, 230, 235)
    base = CreateCube().Scale(0.65, 0.2, 0.65).Position([Float(i) * 2.2, 0.1, 2.5]).Color(200, 200, 205)
Next

beam = CreateCube().Scale(7.5, 0.35, 0.6).Position([0, 5.45, 2.5]).Color(225, 225, 230)

; Tiered steps
For s = 1 To 4
    stepW# = 7.0 - Float(s) * 1.2
    st = CreateCube().Scale(stepW, 0.25, 2.8 - Float(s) * 0.4).Position([0, Float(s) * 0.25 - 0.125, -0.5])
    st.Color(190 + s * 10, 195 + s * 10, 205 + s * 10)
Next

; Props on the top step
cube = CreateCube().Scale(0.6, 0.6, 0.6).Position([-1.2, 1.3, -0.5]).Color(80, 160, 240)
sphere = CreateSphere().Scale(0.6, 0.6, 0.6).Position([1.2, 1.3, -0.5]).Color(240, 140, 60)

camAngle# = 0.0

; Loop
While Not KeyDown(1)
    dt# = DeltaTime() * 60.0
    camAngle = camAngle + 0.35 * dt

    camX# = Sin(camAngle) * 9.5
    camZ# = -Cos(camAngle) * 9.5
    camera.Position(camX, 4.2 + Sin(camAngle * 1.5) * 0.8, camZ)
    camera.Point(0, 1.6, 1.0)

    If KeyHit(57) Then
        isSSAO = GetSSAO()
        If isSSAO Then
            SetSSAO(0)
        Else
            SetSSAO(1)
        EndIf
    EndIf

    RenderWorld

    Color 20, 25, 35, 210
    Rect 16, 16, 380, 72, 1
    Color 70, 140, 230, 255
    Rect 16, 16, 380, 72, 0

    Color 255, 255, 255
    ssaoText$ = "OFF"
    If GetSSAO() Then ssaoText = "ON (Screen-Space Ambient Occlusion)"
    Text 28, 28, "SSAO: " + ssaoText
    Color 180, 210, 255
    Text 28, 48, "SPACE: Toggle SSAO ON / OFF"
    Text 28, 64, "ESC: Quit"

    Flip
Wend
End
