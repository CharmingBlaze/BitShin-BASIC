; Weather — atmosphere, clouds, particles, lightning, and wind on a small field.
; 1 clear  2 rain  3 snow  4 fog  5 storm. T long storm blend. L lightning. -/= intensity.
; Esc quits after a few frames (runtime ignores Escape until the first Flip).

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Weather")
SetCameraClsColor(70, 120, 190)
SetAmbientLight(70, 82, 100)
HidePointer()
MoveMouse(550, 350)

; Camera
cam = CreateCamera().Position([0, 2.2, -8])
SetCameraRange(cam, 0.15, 4000)

; Sky / atmosphere
CreateSkyBox("default")
CreateAtmosphere()
SetAtmosphere(-0.35, 0.62, 0.70, 1.0, 1.05, 1.15)
SetClouds(0.32, 0.9, 0.6)

; Light / shadows
sun = CreateDirectionalLight()
SetLightDirection(sun, 40, 35, 0)
SetLightColor(sun, 255, 236, 200)
SetLightShadow(sun, True)
EnableShadows(True)
ShadowMapSize(1024)
SetShadowQuality(0, 4)
SetShadowBias(0.0018)
SetAmbientColor(61, 71, 92)

; Ground / props
ground = CreatePlane().Scale(80, 1, 80).Position([0, 0, 0]).Color(62, 92, 58)
ground.Shininess(0.12)

For i = 1 To 14
    g = CreateBox(0.12, 1.1 + Rnd(0.4), 0.12).Position([Rnd(32) - 16, 0.55, Rnd(32) - 16])
    g.Color(40 + Rand(0, 40), 110 + Rand(0, 50), 40)
    SetWindSway(g, True, i * 0.4)
Next

For i = 1 To 6
    box = CreateCube().Scale(1.1, 1.1, 1.1).Position([Rnd(20) - 10, 0.55, 2 + Rnd(12)])
    box.Color(140 + Rand(0, 80), 90 + Rand(0, 40), 70)
    box.Shininess(0.35)
    box.Specular(80, 90, 100)
Next

tree = CreateCone().Scale(1.4, 3.2, 1.4).Position([-4, 1.6, 6]).Color(36, 92, 48)
SetWindSway(tree, True, 1.2)

; Weather defaults
SetWeatherDryingSpeed(0.02)
SetCameraRain(True)
SetWeatherTransition("rain", 0.85, 6)
SetWind(1.2, 0, 0.4, 1.1)

yaw# = 0
pitch# = 8
SetMousePosition(550, 350)
frames = 0

; Loop
While 1
    frames = frames + 1
    dt# = DeltaTime()
    mxs# = MouseDeltaX()
    mys# = MouseDeltaY()
    yaw = yaw + mxs * 0.12
    pitch = pitch - mys * 0.12
    If pitch > 80 Then pitch = 80
    If pitch < -80 Then pitch = -80
    cam.Rotate(pitch, yaw, 0)
    SetMousePosition(550, 350)

    spd# = 8 * dt
    If KeyDown(KEY_W) Then cam.Move(0, 0, spd)
    If KeyDown(KEY_S) Then cam.Move(0, 0, -spd)
    If KeyDown(KEY_A) Then cam.Move(-spd, 0, 0)
    If KeyDown(KEY_D) Then cam.Move(spd, 0, 0)

    If KeyHit(KEY_1) Then SetWeather("clear")
    If KeyHit(KEY_2) Then SetWeather("rain")
    If KeyHit(KEY_3) Then SetWeather("snow")
    If KeyHit(KEY_4) Then SetWeather("fog")
    If KeyHit(KEY_5) Then SetWeather("storm")
    If KeyHit(KEY_T) Then SetWeatherTransition("storm", 0.85, 10)
    If KeyHit(KEY_L) Then StrikeLightning()
    If KeyHit(KEY_MINUS) Then SetWeatherIntensity(WeatherIntensity() - 0.1)
    If KeyHit(KEY_EQUALS) Then SetWeatherIntensity(WeatherIntensity() + 0.1)

    Text(12, 12, "1 clear  2 rain  3 snow  4 fog  5 storm   T 10s storm   L lightning  -/= intensity")
    Text(12, 32, "Weather=" + Weather$() + "  I=" + Str(WeatherIntensity()) + "  wet=" + Str(WeatherWetness()) + "  Esc quit")
    RenderWorld
    Flip
    If frames > 8
        If KeyHit(KEY_ESCAPE) Then End
    EndIf
Wend
End
