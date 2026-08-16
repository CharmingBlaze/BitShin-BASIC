package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/g3n/engine/camera"
	"github.com/g3n/engine/math32"

	"bitshinbasic/internal/ast"
	"bitshinbasic/internal/lex"
	"bitshinbasic/internal/parse"
)

func TestShowcaseCameraProjectsOrigin(t *testing.T) {
	cam := camera.New(16.0 / 9.0)
	cam.SetFar(4000)
	gx, gy, gz := toG3N(0, 2, -6)
	cam.SetPosition(gx, gy, gz)
	origin := math32.Vector3{0, 0, 0}
	cam.Project(&origin)
	t.Logf("identity at Blitz (0,2,-6) → G3N (%.2f,%.2f,%.2f) project origin NDC=(%.3f,%.3f,%.3f)",
		gx, gy, gz, origin.X, origin.Y, origin.Z)

	cam2 := camera.New(16.0 / 9.0)
	cam2.SetFar(4000)
	cam2.SetPosition(gx, gy, gz)
	up := math32.Vector3{0, 1, 0}
	cam2.LookAt(&math32.Vector3{0, 0, 0}, &up)
	hit := math32.Vector3{0, 0, 0}
	cam2.Project(&hit)
	t.Logf("LookAt origin project NDC=(%.3f,%.3f,%.3f)", hit.X, hit.Y, hit.Z)

	// spinning_cube: camera at origin, cube at Blitz (0,0,5)
	cam3 := camera.New(16.0 / 9.0)
	cx, cy, cz := toG3N(0, 0, 5)
	cube := math32.Vector3{cx, cy, cz}
	cam3.Project(&cube)
	t.Logf("identity at origin project Blitz (0,0,5) NDC=(%.3f,%.3f,%.3f)", cube.X, cube.Y, cube.Z)
}

func TestAimDefaultCameraSeesOrigin(t *testing.T) {
	cam := camera.New(16.0 / 9.0)
	cam.SetFar(4000)
	gx, gy, gz := toG3N(0, 2, -6)
	cam.SetPosition(gx, gy, gz)
	e := &Entity{node: cam, cam: cam}
	New(".").aimDefaultCamera(e)
	origin := math32.Vector3{0, 0, 0}
	cam.Project(&origin)
	if origin.X < -0.35 || origin.X > 0.35 || origin.Y < -0.35 || origin.Y > 0.35 || origin.Z <= 0 || origin.Z >= 1 {
		t.Fatalf("PositionEntity(0,2,-6) must frame origin near center, NDC=%v", origin)
	}
}

func TestClawPointEntityStillSeesTarget(t *testing.T) {
	cam := camera.New(16.0 / 9.0)
	cam.SetFar(4000)
	gx, gy, gz := toG3N(0, 7.2, -13)
	cam.SetPosition(gx, gy, gz)
	e := &Entity{node: cam, cam: cam}
	New(".").aimDefaultCamera(e)
	tx, ty, tz := toG3N(0, 3.2, 0)
	up := math32.Vector3{0, 1, 0}
	cam.LookAt(&math32.Vector3{tx, ty, tz}, &up)
	tgt := math32.Vector3{tx, ty, tz}
	cam.Project(&tgt)
	if tgt.X < -0.95 || tgt.X > 0.95 || tgt.Y < -0.95 || tgt.Y > 0.95 || tgt.Z <= 0 || tgt.Z >= 1 {
		t.Fatalf("claw PointEntity must still see (0,3.2,0), NDC=%v", tgt)
	}
}

func TestCommandTableHasShowcaseNames(t *testing.T) {
	m := New(".").commandTable()
	need := []string{
		"graphics3d", "setwindowtitle", "createcamera", "positionentity",
		"cameraclscolor", "createlight", "setlightdirection", "setlightcolor",
		"createcube", "entitycolor", "createplane", "keydown", "deltatime",
		"turnentity", "renderworld", "flip", "hudprint",
	}
	for _, name := range need {
		if m[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}
}

func TestShowcaseExampleCommandsExist(t *testing.T) {
	path := filepath.Join("..", "..", "examples", "showcase.bb")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	prog, err := parse.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	table := New(".").commandTable()
	known := map[string]bool{"print": true}
	for k := range table {
		known[k] = true
	}
	walkShowcaseStmts(t, prog.Stmts, known)
}

func walkShowcaseStmts(t *testing.T, stmts []ast.Stmt, known map[string]bool) {
	t.Helper()
	for _, s := range stmts {
		switch n := s.(type) {
		case *ast.CallStmt:
			name := lex.IdentKey(n.Name)
			if !known[name] {
				t.Errorf("showcase.bb unknown command %s", n.Name)
			}
		case *ast.AssignStmt:
			if c, ok := n.Value.(*ast.CallExpr); ok {
				name := lex.IdentKey(c.Name)
				if !known[name] {
					t.Errorf("showcase.bb unknown command %s", c.Name)
				}
			}
		case *ast.WhileStmt:
			walkShowcaseStmts(t, n.Body, known)
		}
	}
}
