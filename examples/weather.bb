; Weather stack: atmosphere + clouds + particles + lightning + wind.
; 1 clear  2 rain  3 snow  4 fog  5 storm. T = long storm blend.
; Runtime ignores Escape / WindowShouldClose until the first Flip. While 1 is extra.

Graphics3D(1100, 700)
SetWindowTitle("BitShin BASIC — Weather")
SetCameraClsColor(70, 120, 190)
SetAmbientLight(70, 82, 100)
HidePointer()
MoveMouse(550, 350)

cam = CreateCamera()
SetPosition(cam, 0, 2.2, -8)
SetCameraRange(cam, 0.15, 4000)

CreateSkyBox("default")
CreateAtmosphere()
SetAtmosphere(-0.35, 0.62, 0.70, 1.0, 1.05, 1.15)
SetClouds(0.32, 0.9, 0.6)

sun = CreateDirectionalLight()
SetLightDirection(sun, 40, 35, 0)
SetLightColor(sun, 255, 236, 200)
SetLightShadow(sun, True)
EnableShadows(True)
ShadowMapSize(1024)
SetShadowQuality(0, 4)
SetShadowBias(0.0018)
SetAmbientColor(61, 71, 92)

ground = CreatePlane()
SetScale(ground, 80, 1, 80)
SetPosition(ground, 0, 0, 0)
SetEntityColor(ground, 62, 92, 58)
EntityShininess(ground, 0.12)

For i = 1 To 14
    g = CreateBox(0.12, 1.1 + Rnd(0.4), 0.12)
    SetPosition(g, Rnd(32) - 16, 0.55, Rnd(32) - 16)
    SetEntityColor(g, 40 + Rand(0, 40), 110 + Rand(0, 50), 40)
    SetWindSway(g, True, i * 0.4)
Next

For i = 1 To 6
    box = CreateCube()
    SetScale(box, 1.1, 1.1, 1.1)
    SetPosition(box, Rnd(20) - 10, 0.55, 2 + Rnd(12))
    SetEntityColor(box, 140 + Rand(0, 80), 90 + Rand(0, 40), 70)
    EntityShininess(box, 0.35)
    EntitySpecular(box, 80, 90, 100)
Next

tree = CreateCone()
SetScale(tree, 1.4, 3.2, 1.4)
SetPosition(tree, -4, 1.6, 6)
SetEntityColor(tree, 36, 92, 48)
SetWindSway(tree, True, 1.2)

SetWeatherDryingSpeed(0.02)
SetCameraRain(True)
SetWeatherTransition("rain", 0.85, 6)
SetWind(1.2, 0, 0.4, 1.1)

yaw# = 0
pitch# = 8
SetMousePosition(550, 350)
frames = 0

While 1
    frames = frames + 1
    dt# = DeltaTime()
    mxs# = MouseDeltaX()
    mys# = MouseDeltaY()
    yaw = yaw + mxs * 0.12
    pitch = pitch - mys * 0.12
    If pitch > 80 Then pitch = 80
    If pitch < -80 Then pitch = -80
    SetRotation(cam, pitch, yaw, 0)
    SetMousePosition(550, 350)

    spd# = 8 * dt
    If KeyDown(KEY_W) Then MoveEntity(cam, 0, 0, spd)
    If KeyDown(KEY_S) Then MoveEntity(cam, 0, 0, -spd)
    If KeyDown(KEY_A) Then MoveEntity(cam, -spd, 0, 0)
    If KeyDown(KEY_D) Then MoveEntity(cam, spd, 0, 0)

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
