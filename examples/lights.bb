; Lights — outdoor sun/sky look, then Space toggles an indoor room lighting set.
; Space (KeyHit 57) switches outdoor ↔ indoor. Esc quits.

Graphics3D(640, 480)
SetBuffer(BackBuffer())
SetWindowTitle("BitShin BASIC — Lights")

; Camera
cam = CreateCamera().Position([0, 2, -6])

; Shadows
EnableShadows True
ShadowCascades 2
SetShadowFilter "pcf"
EnableShadowAtlas True

; Lights
sun = CreateLight(1).Rotate(50, 30, 0)
SetLightColor(sun, 255, 230, 180)
SetLightSpecular(sun, 255, 244, 220)
SetLightShadow sun, True

lamp = CreateLight(2).Position([2, 2, 4])
SetLightColor(lamp, 80, 160, 255)
SetLightRange(lamp, 12)
SetLightSpecular(lamp, 140, 190, 255)
SetLightShadow lamp, True

spot = CreateSpotLight().Position([-2.2, 3.2, 2])
SetLightDirection(spot, -50, 24, 0)
SetLightCone(spot, 16, 34)
SetLightColor(spot, 255, 170, 70)
SetLightRange(spot, 12)
SetLightAmbient(spot, 24, 12, 4)

; World
cube = CreateCube().Position([0, 0, 5]).Color(230, 226, 218)
SetEntityAmbient(cube, 36, 34, 32)
cube.Specular(255, 255, 255)
cube.Shininess(48)

box = CreateCube().Position([-1.7, 0, 6.2]).Color(40, 90, 190)
box.Specular(180, 210, 255)
box.Shininess(64)

bulb = CreatePointLight().Position([-1.7, 1.5, 6.2])
SetLightColor(bulb, 255, 214, 150)
SetLightAttenuation(bulb, 1, 0.22, 0.20)

; Loop — Space toggles outdoor / indoor
SetLighting("outdoor")
env = 1
While Not KeyDown(1)
    dt# = DeltaTime() * 60
    cube.Turn(0.3 * dt, 0.5 * dt, 0)
    If KeyHit(57)
        If env = 1
            IndoorLighting()
            env = 2
        Else
            OutdoorLighting()
            env = 1
        EndIf
    EndIf
    RenderWorld
    Flip
Wend
End
