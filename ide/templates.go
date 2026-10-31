package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectTemplate is one New Project starting point.
type ProjectTemplate struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Is2D        bool   `json:"is2D"`
	main        string
	scene       []sceneEntity
}

// sceneEntity is one entity of a version-2 scene file.
type sceneEntity struct {
	ID       int        `json:"id"`
	Name     string     `json:"name"`
	Kind     string     `json:"kind"`
	Parent   int        `json:"parent"`
	Position [3]float64 `json:"position"`
	Rotation [3]float64 `json:"rotation"`
	Scale    [3]float64 `json:"scale"`
	Color    [3]int     `json:"color"`
	Light    string     `json:"light,omitempty"`
	Body     string     `json:"body,omitempty"`
}

// starterScene is the ground, sun, and two props every 3D template's
// scenes/main.scene.json holds. main.mb builds the same objects in code.
var starterScene = []sceneEntity{
	{ID: 1, Name: "Ground", Kind: "cube", Position: [3]float64{0, 0, 0}, Scale: [3]float64{40, 0.25, 40}, Color: [3]int{45, 80, 60}, Body: "static"},
	{ID: 2, Name: "Sun", Kind: "light", Light: "directional", Position: [3]float64{0, 20, 0}, Rotation: [3]float64{45, 35, 0}, Scale: [3]float64{1, 1, 1}, Color: [3]int{255, 240, 210}},
	{ID: 3, Name: "Crate", Kind: "cube", Position: [3]float64{3, 1.25, 4}, Scale: [3]float64{1, 1, 1}, Color: [3]int{180, 120, 70}, Body: "static"},
	{ID: 4, Name: "Pillar", Kind: "cylinder", Position: [3]float64{-4, 2, 6}, Scale: [3]float64{1, 2, 1}, Color: [3]int{150, 160, 180}, Body: "static"},
}

// sceneComment is shared by the 3D templates.
const sceneComment = `; The ground, sun, crate, and pillar below are also in scenes/main.scene.json.
; Double-click that file in the Explorer to edit it in the scene editor.
; When your moon.exe's LoadScene reads version-2 scene files, you can replace
; this block with:  LoadScene("scenes/main.scene.json")
`

const worldBlock = `sun = CreateDirectionalLight()
SetLightDirection(sun, 45, 35, 0)
SetLightColor(sun, 255, 240, 210)

ground = CreateCube().Scale(40, 0.25, 40).Position([0, 0, 0]).Color(45, 80, 60)
CreateRigidBodyBox(ground, 40, 0.25, 40, 0)

crate = CreateCube().Scale(1, 1, 1).Position([3, 1.25, 4]).Color(180, 120, 70)
CreateRigidBodyBox(crate, 1, 1, 1, 0)

pillar = CreateCylinder(12).Scale(1, 2, 1).Position([-4, 2, 6]).Color(150, 160, 180)
CreateRigidBodyBox(pillar, 1, 2, 1, 0)
`

var projectTemplates = []ProjectTemplate{
	{
		ID:          "empty3d",
		Title:       "Empty 3D",
		Description: "Camera, sun, ground, and two props. Arrow keys orbit.",
		scene:       starterScene,
		main: `; {{NAME}} — empty 3D project.
; F5 runs this file: it is "main" in moon.json. Arrow keys orbit, Esc quits.

SetWindowTitle("{{TITLE}}")
Graphics3D({{W}}, {{H}})
SetGraphicsQuality("{{QUALITY}}")
SetCameraClsColor(40, 50, 70)
SetAmbientLight(70, 75, 90)

cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

` + sceneComment + worldBlock + `
; Orbit the camera around the middle of the scene.
yaw# = 0
While Not Input.KeyDown(KEY_ESCAPE)
    yaw = yaw + GetAxis(KEY_LEFT, KEY_RIGHT) * 90 * Time.Delta()
    PositionEntity(cam, Math.Sin(yaw) * 14, 6, -Math.Cos(yaw) * 14)
    PointEntity(cam, crate)

    UpdateWorld
    RenderWorld
    Text(16, 16, "{{TITLE}} — arrows orbit, Esc quits")
    Flip
Wend
End
`,
	},
	{
		ID:          "empty2d",
		Title:       "Empty 2D",
		Description: "A 2D window with a moving shape and HUD text.",
		Is2D:        true,
		main: `; {{NAME}} — empty 2D project.
; F5 runs this file: it is "main" in moon.json. Esc quits.

SetWindowTitle("{{TITLE}}")
Graphics2D({{W}}, {{H}})
SetClsColor(18, 22, 34)

; A ball that bounces between the window edges.
x# = 100
vx# = 180

While Not Input.KeyDown(KEY_ESCAPE)
    x = x + vx * Time.Delta()
    If x > {{W}} - 40 Or x < 0 Then vx = -vx

    Cls
    SetColor(80, 180, 255)
    Oval(x, {{H}} / 2 - 20, 40, 40)
    SetColor(230, 230, 240)
    Text(16, 16, "{{TITLE}} — Esc quits")
    Flip
Wend
End
`,
	},
	{
		ID:          "fps",
		Title:       "First-person",
		Description: "FPS controller: WASD, mouse look, jump, sprint, crouch.",
		scene:       starterScene,
		main: `; {{NAME}} — first-person project.
; WASD move, mouse look, Space jump, Shift sprint, Ctrl crouch, Esc quits.
; The controller commands are in docs/MODERN_GAME_HELPERS.md.

SetWindowTitle("{{TITLE}}")
Graphics3D({{W}}, {{H}})
SetGraphicsQuality("{{QUALITY}}")
SetCameraClsColor(40, 50, 70)
SetAmbientLight(70, 75, 90)
HidePointer()

cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

` + sceneComment + worldBlock + `
; The player is a pivot; the FPS controller moves it and aims the camera.
player = CreatePivot()
player.Position([0, 2, 0])
CreateFPSController(player, cam, 1.8, 0.4, 6.0, 10.5, 9.5)

frames = 0
While 1
    frames = frames + 1
    If frames > 8 And Input.KeyHit(KEY_ESCAPE) Then Exit

    walkX# = GetAxis(KEY_A, KEY_D)
    walkZ# = GetAxis(KEY_S, KEY_W)
    jump = Input.KeyHit(KEY_SPACE)
    sprint = Input.KeyDown(KEY_LEFT_SHIFT)
    crouch = Input.KeyDown(KEY_LEFT_CONTROL)
    UpdateFPS(player, walkX#, walkZ#, jump, sprint, crouch, 0.15)

    UpdateWorld
    RenderWorld
    Text(16, 16, "WASD move, mouse look, Space jump, Esc quits")
    Flip
Wend
End
`,
	},
	{
		ID:          "tps",
		Title:       "Third-person",
		Description: "TPS controller with a spring-arm camera that orbits with the mouse.",
		scene:       starterScene,
		main: `; {{NAME}} — third-person project.
; WASD move, mouse orbits the camera, Space jump, Shift sprint, Esc quits.
; The controller commands are in docs/MODERN_GAME_HELPERS.md.

SetWindowTitle("{{TITLE}}")
Graphics3D({{W}}, {{H}})
SetGraphicsQuality("{{QUALITY}}")
SetCameraClsColor(50, 70, 95)
SetAmbientLight(70, 75, 90)
HidePointer()

cam = CreateCamera()
SetCameraRange(cam, 0.1, 1000)

` + sceneComment + worldBlock + `
; The player is a pivot with a body and a head parented to it.
player = CreatePivot()
player.Position([0, 2, 0])
body = CreateCylinder(12, player).Scale(0.5, 0.65, 0.5).Position([0, 0.8, 0]).Color(60, 150, 220)
head = CreateSphere(10, player).Scale(0.35, 0.35, 0.35).Position([0, 1.6, 0]).Color(255, 210, 160)
CreateTPSController(player, cam, 1.8, 0.45, 6.5, 11.0, 10.0)

camPitch# = 15
camYaw# = 0
frames = 0
While 1
    frames = frames + 1
    If frames > 8 And Input.KeyHit(KEY_ESCAPE) Then Exit

    camYaw = camYaw + MouseDeltaX() * 0.18
    camPitch = Math.Clamp(camPitch + MouseDeltaY() * 0.14, -20, 65)
    moveX# = GetAxis(KEY_A, KEY_D)
    moveZ# = GetAxis(KEY_S, KEY_W)
    jump = Input.KeyHit(KEY_SPACE)
    sprint = Input.KeyDown(KEY_LEFT_SHIFT)
    UpdateTPS(player, moveX#, moveZ#, jump, sprint, 0, 4.5, camPitch, camYaw)

    UpdateWorld
    RenderWorld
    Text(16, 16, "WASD move, mouse orbits, Space jump, Esc quits")
    Flip
Wend
End
`,
	},
	{
		ID:          "platformer2d",
		Title:       "Platformer 2D",
		Description: "Chipmunk physics character with ledges and a one-way platform.",
		Is2D:        true,
		main: `; {{NAME}} — 2D platformer project.
; Arrows move, Space or Up jumps, Esc quits.
; CreateCharacter2D and friends are in docs/PHYSICS.md.

SetWindowTitle("{{TITLE}}")
Graphics2D({{W}}, {{H}})
SetClsColor(18, 22, 34)

Physics2D()
Gravity2D(0, 900)
SetColor(255, 255, 255)

; Floor along the bottom of the window
floor = CreateSprite(CreateImage({{W}}, 40))
SetSpritePosition(floor, {{W}} / 2, {{H}} - 20)
SpriteColor(floor, 60, 70, 90)
CreateBox2D(floor, {{W}}, 40, 0, 0)

; A solid ledge and a one-way platform you can jump up through
ledge = CreateSprite(CreateImage(160, 16))
SetSpritePosition(ledge, 160, {{H}} - 180)
SpriteColor(ledge, 60, 70, 90)
CreateBox2D(ledge, 160, 16, 0, 0)

oneway = CreateSprite(CreateImage(160, 12))
SetSpritePosition(oneway, 380, {{H}} - 110)
SpriteColor(oneway, 230, 200, 90)
CreateBox2D(oneway, 160, 12, 0, 0)
SetOneWay2D(oneway)

; The player: a physics character controller on a sprite
player = CreateSprite(CreateImage(24, 48))
SetSpritePosition(player, 80, {{H}} - 80)
SpriteColor(player, 80, 190, 255)
CreateCharacter2D(player, 24, 48)
SetCharacterSpeed2D(player, 220)
SetCharacterJump2D(player, 110)
SetCharacterCoyote2D(player, 0.1)
SetCharacterJumpBuffer2D(player, 0.12)

While Not Input.KeyDown(KEY_ESCAPE)
    dx# = 0
    If Input.KeyDown(KEY_LEFT) Then dx = dx - 1
    If Input.KeyDown(KEY_RIGHT) Then dx = dx + 1
    MoveCharacter2D(player, dx)
    If Input.KeyHit(KEY_SPACE) Or Input.KeyHit(KEY_UP) Then JumpCharacter2D(player)

    UpdateWorld

    Cls
    SetColor(230, 230, 240)
    Text(16, 16, "Arrows move, Space jumps, Esc quits")
    Flip
Wend
End
`,
	},
	{
		ID:          "topdown2d",
		Title:       "Top-down 2D",
		Description: "WASD movement over a world larger than the screen, with a following camera.",
		Is2D:        true,
		main: `; {{NAME}} — top-down 2D project.
; WASD or arrows move, Esc quits. The camera follows the player.

SetWindowTitle("{{TITLE}}")
Graphics2D({{W}}, {{H}})
SetClsColor(24, 30, 26)

worldW = 2000
worldH = 2000
speed# = 260

; Sprites are in the world and move with the camera; Text stays on screen.
SetColor(255, 255, 255)
For i = 0 To 39
    rock = CreateSprite(CreateImage(30, 30))
    SetSpritePosition(rock, Rnd(0, worldW), Rnd(0, worldH))
    SpriteColor(rock, 70, 80, 72)
Next

player = CreateSprite(CreateImage(28, 28))
SetSpritePosition(player, worldW / 2, worldH / 2)
SpriteColor(player, 80, 190, 255)
SpriteLayer(player, 1)

; Camera 0 follows the player smoothly and stays inside the world.
CameraTarget2D(0, player)
CameraSmooth2D(0, 0.25)
CameraBounds2D(0, 0, 0, worldW, worldH)

While Not Input.KeyDown(KEY_ESCAPE)
    dt# = Time.Delta()
    mx# = GetAxis(KEY_A, KEY_D) + GetAxis(KEY_LEFT, KEY_RIGHT)
    my# = GetAxis(KEY_W, KEY_S) + GetAxis(KEY_UP, KEY_DOWN)
    px# = Math.Clamp(SpriteX(player) + mx * speed * dt, 0, worldW)
    py# = Math.Clamp(SpriteY(player) + my * speed * dt, 0, worldH)
    SetSpritePosition(player, px, py)

    Cls
    SetColor(230, 230, 240)
    Text(16, 16, "WASD move, Esc quits")
    Flip
Wend
End
`,
	},
}

func findTemplate(id string) (ProjectTemplate, bool) {
	if id == "" {
		id = "empty3d"
	}
	for _, t := range projectTemplates {
		if t.ID == id {
			return t, true
		}
	}
	return ProjectTemplate{}, false
}

// ListProjectTemplates returns the New Project choices.
func (a *App) ListProjectTemplates() []ProjectTemplate {
	return projectTemplates
}

// writeTemplate writes moon.json, main.mb, assets/, and for 3D templates
// scenes/main.scene.json into root.
func writeTemplate(root string, p Project, t ProjectTemplate) error {
	for _, dir := range []string{root, filepath.Join(root, filepath.FromSlash(p.Assets)), filepath.Join(root, filepath.FromSlash(p.Scenes))} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if err := writeProjectFile(root, p); err != nil {
		return err
	}
	title := strings.ReplaceAll(p.Window.Title, `"`, "'")
	main := strings.NewReplacer(
		"{{NAME}}", p.Name,
		"{{TITLE}}", title,
		"{{W}}", fmt.Sprint(p.Window.Width),
		"{{H}}", fmt.Sprint(p.Window.Height),
		"{{QUALITY}}", p.Quality,
	).Replace(t.main)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p.Main)), []byte(main), 0o644); err != nil {
		return err
	}
	if len(t.scene) > 0 {
		data, err := sceneFileJSON(t.scene)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(p.Scenes), "main.scene.json"), data, 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(root, filepath.FromSlash(p.Assets), "README.txt"),
		[]byte("Put models, textures, and sounds here. Load them with paths like \"assets/player.glb\".\n"), 0o644)
}

// sceneFileJSON writes a version-2 scene with one entity per line, so the
// file stays readable and diffs well.
func sceneFileJSON(ents []sceneEntity) ([]byte, error) {
	var b strings.Builder
	b.WriteString("{\n  \"version\": 2,\n  \"environment\": {\"ambient\": [70, 75, 90], \"clearColor\": [40, 50, 70]},\n  \"entities\": [\n")
	for i, e := range ents {
		line, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		b.WriteString("    ")
		b.Write(line)
		if i < len(ents)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("  ]\n}\n")
	return []byte(b.String()), nil
}
