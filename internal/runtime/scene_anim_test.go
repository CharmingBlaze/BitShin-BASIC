package runtime

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/g3n/engine/core"
	"github.com/g3n/engine/math32"

	"moonbasic/internal/value"
)

func TestPhysicsThreadsSizesJobPool(t *testing.T) {
	w := New(".")
	w.applyPhysThreads(8)
	if w.physThreads != 8 {
		t.Fatalf("physThreads %d", w.physThreads)
	}
	if w.jobWorkers != 8 {
		t.Fatalf("jobWorkers %d", w.jobWorkers)
	}
	p := w.ensureJobs()
	if p == nil {
		t.Fatal("job pool")
	}
	w.applyPhysThreads(4)
	if w.jobs != nil {
		t.Fatal("pool should rebuild after PhysicsThreads")
	}
	p2 := w.ensureJobs()
	if p2 == nil || p2 == p {
		t.Fatal("expected a new job pool")
	}
	p2.close()
}

func TestNlerpQuatHalfTurn(t *testing.T) {
	a := math32.Quaternion{W: 1}
	b := math32.Quaternion{Y: 1}
	q := nlerpQuat(a, b, 0.5)
	if q.W < 0.6 || q.Y < 0.6 {
		t.Fatalf("nlerp %v %v %v %v", q.X, q.Y, q.Z, q.W)
	}
}

func TestAnimBlendDefaults(t *testing.T) {
	st := &animState{dir: 1, speed: 1, prev: -1, blend: 1}
	if st.prev != -1 || st.blend != 1 {
		t.Fatal("idle blend must not mix clip 0")
	}
}

func TestSceneKindFallbackCube(t *testing.T) {
	if spawnKindName("") != "cube" {
		t.Fatalf("empty kind %q", spawnKindName(""))
	}
	if spawnKindName("sphere") != "sphere" {
		t.Fatal("sphere")
	}
	if spawnKindName("MESH") != "mesh" {
		t.Fatal("mesh")
	}
	if spawnKindName("light") != "light" {
		t.Fatal("light")
	}
	if spawnKindName("camera") != "camera" {
		t.Fatal("camera")
	}
}

func TestSceneJSONWritesLightsAndVisible(t *testing.T) {
	w := New(".")
	id := w.makeLight(2, 0)
	if id == 0 {
		t.Fatal("light")
	}
	w.ents[id].node.GetNode().SetVisible(false)
	root := w.sceneJSON()
	list, _ := root["entities"].([]any)
	found := false
	for _, raw := range list {
		em, _ := raw.(map[string]any)
		if em["kind"] != "light" {
			continue
		}
		found = true
		if em["visible"] != false {
			t.Fatalf("visible %v", em["visible"])
		}
		if int(anyToValue(em["light"]).Number()) != 2 {
			t.Fatalf("light kind %v", em["light"])
		}
	}
	if !found {
		t.Fatal("scene dump missing light")
	}
}

func TestSceneJSONCameraRoundTrip(t *testing.T) {
	w := New(".")
	id := w.spawnCamera(0)
	if id == 0 || w.ents[id] == nil || w.ents[id].cam == nil {
		t.Fatal("camera")
	}
	w.ents[id].cam.SetFov(75)
	w.ents[id].cam.SetNear(0.25)
	w.ents[id].cam.SetFar(800)
	root := w.sceneJSON()
	list, _ := root["entities"].([]any)
	found := false
	for _, raw := range list {
		em, _ := raw.(map[string]any)
		if em["kind"] != "camera" {
			continue
		}
		found = true
		if int(anyToValue(em["fov"]).Number()) != 75 {
			t.Fatalf("fov %v", em["fov"])
		}
		if anyToValue(em["near"]).Number() < 0.2 || anyToValue(em["near"]).Number() > 0.3 {
			t.Fatalf("near %v", em["near"])
		}
		if int(anyToValue(em["far"]).Number()) != 800 {
			t.Fatalf("far %v", em["far"])
		}
	}
	if !found {
		t.Fatal("scene dump missing camera")
	}
	w2 := New(".")
	w2.applySceneJSON(root)
	ok := false
	for _, e := range w2.ents {
		if e == nil || e.cam == nil {
			continue
		}
		ok = true
		if e.kind != "camera" {
			t.Fatalf("kind %q", e.kind)
		}
		if e.cam.Fov() < 74 || e.cam.Fov() > 76 {
			t.Fatalf("restored fov %v", e.cam.Fov())
		}
		if e.cam.Near() < 0.2 || e.cam.Near() > 0.3 {
			t.Fatalf("restored near %v", e.cam.Near())
		}
		if e.cam.Far() < 799 || e.cam.Far() > 801 {
			t.Fatalf("restored far %v", e.cam.Far())
		}
	}
	if !ok {
		t.Fatal("SceneLoad did not rebuild a camera")
	}
}

func TestSceneJSONRestoresRigidBox(t *testing.T) {
	w := New(".")
	id := w.createCubeMesh(nil)
	w.ents[id].node.GetNode().SetPosition(toG3N(0, 1, 4))
	w.ensurePhys3()
	w.phys3.AddBoxEx(id, 0, 1, 4, 2, 0.25, 2, 2)
	w.phys3.SetMass(id, 40)
	e := w.ents[id]
	e.bodyType = 1
	e.collKind = 1
	e.boxX, e.boxY, e.boxZ = 2, 0.25, 2
	root := w.sceneJSON()
	w2 := New(".")
	w2.applySceneJSON(root)
	var found int
	for id2, e2 := range w2.ents {
		if e2 == nil || e2.collKind != 1 {
			continue
		}
		found = id2
	}
	if found == 0 {
		t.Fatal("scene load dropped the rigid box")
	}
	px, py, pz, ok := w2.phys3.GetPosition(found)
	if !ok {
		t.Fatal("restored entity has no physics body")
	}
	if py < 0.9 || py > 1.1 || px > 0.1 || pz < 3.9 || pz > 4.1 {
		t.Fatalf("restored body pose (%v,%v,%v)", px, py, pz)
	}
	if w2.phys3.GetMass(found) < 39 || w2.phys3.GetMass(found) > 41 {
		t.Fatalf("restored mass %v", w2.phys3.GetMass(found))
	}
	_, _, _, _, hit := w2.phys3.Raycast(0, 5, 4, 0, -8, 0)
	if !hit {
		t.Fatal("ray should hit the restored box")
	}
}

func TestSceneJSONRestoresPlay(t *testing.T) {
	w := New(".")
	road := w.createCubeMesh(nil)
	w.ents[road].node.GetNode().SetPosition(toG3N(0, 0, 10))
	w.ents[road].node.GetNode().SetScale(4, 0.2, 6)
	w.ensurePhys3()
	if !w.addEntityMeshBody(road, w.ents[road]) {
		t.Fatal("mesh collider was not created")
	}

	left := w.createCubeMesh(nil)
	right := w.createCubeMesh(nil)
	w.ents[left].node.GetNode().SetPosition(toG3N(0, 2, 0))
	w.ents[right].node.GetNode().SetPosition(toG3N(2, 2, 0))
	w.ensurePhys3()
	w.phys3.AddBoxEx(left, 0, 2, 0, 0.5, 0.5, 0.5, 2)
	w.phys3.AddBoxEx(right, 2, 2, 0, 0.5, 0.5, 0.5, 2)
	w.phys3.SetMass(left, 1)
	w.phys3.SetMass(right, 1)
	for _, id := range []int{left, right} {
		e := w.ents[id]
		e.bodyType = 1
		e.collKind = 1
		e.boxX, e.boxY, e.boxZ = 0.5, 0.5, 0.5
	}
	if _, err := w.Call("createfixedjoint", []value.Value{
		value.Num(float64(left)), value.Num(float64(right)),
		value.Num(1), value.Num(2), value.Num(0),
	}); err != nil {
		t.Fatal(err)
	}

	car := w.createCubeMesh(nil)
	w.ents[car].node.GetNode().SetPosition(toG3N(0, 1, -8))
	if _, err := w.Call("createcarcontroller", []value.Value{
		value.Num(float64(car)), value.Num(1.1), value.Num(0.4), value.Num(2), value.Num(800),
	}); err != nil {
		t.Fatal(err)
	}

	person := w.createCubeMesh(nil)
	cam := w.createCubeMesh(nil)
	w.ents[person].node.GetNode().SetPosition(toG3N(6, 1, 0))
	if _, err := w.Call("createfpscontroller", []value.Value{
		value.Num(float64(person)), value.Num(float64(cam)),
		value.Num(1.7), value.Num(0.35), value.Num(5), value.Num(9), value.Num(8),
	}); err != nil {
		t.Fatal(err)
	}
	flag := w.createCloth(2.6, 3.4, 12, 14, 4)
	if c := w.cloths[flag]; c != nil {
		c.wind = 7
	}

	raw, err := json.Marshal(w.sceneJSON())
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	w2 := New(".")
	w2.applySceneJSON(root)

	var meshID int
	for id, e := range w2.ents {
		if e != nil && e.collKind == 7 {
			meshID = id
		}
	}
	if meshID == 0 {
		t.Fatal("scene load dropped the mesh collider")
	}
	if _, _, _, _, hit := w2.phys3.Raycast(0, 3, 10, 0, -6, 0); !hit {
		t.Fatal("ray should hit the restored mesh collider")
	}

	var carCtrl *vehCtrl
	for _, c := range w2.vehCtrls {
		if c != nil && c.kind == "car" {
			carCtrl = c
		}
	}
	if carCtrl == nil {
		t.Fatal("scene load dropped the car controller")
	}
	if carCtrl.mass < 700 || carCtrl.mass > 900 {
		t.Fatalf("car mass %v", carCtrl.mass)
	}
	if _, _, _, ok := w2.phys3.GetPosition(carCtrl.id); !ok {
		t.Fatal("reloaded car has no body")
	}
	if runtime.GOOS == "windows" && carCtrl.native == 0 {
		t.Fatal("reloaded car lost its vehicle constraint")
	}

	var fps *fpsCtrl
	for _, c := range w2.fpsControllers {
		fps = c
	}
	if fps == nil {
		t.Fatal("scene load dropped the fps controller")
	}
	if fps.camID == 0 || w2.ents[fps.camID] == nil {
		t.Fatalf("fps camera was not remapped: %d", fps.camID)
	}
	if fps.walkSpeed < 4.5 || fps.walkSpeed > 5.5 || fps.height < 1.6 || fps.height > 1.8 {
		t.Fatalf("fps create args walk=%v height=%v", fps.walkSpeed, fps.height)
	}
	if _, _, _, ok := w2.phys3.GetPosition(fps.id); !ok {
		t.Fatal("reloaded character has no body")
	}
	var sheet *clothSheet
	for _, c := range w2.cloths {
		if c != nil && c.nx == 12 {
			sheet = c
		}
	}
	if sheet == nil || sheet.ny != 14 || sheet.pin != 4 || sheet.width < 2.5 || sheet.width > 2.7 || sheet.wind < 6.5 || sheet.wind > 7.5 {
		t.Fatalf("cloth sheet was not restored: %+v", sheet)
	}

	var bx float32
	var aID, bID int
	for id := range w2.ents {
		x, y, z, ok := w2.phys3.GetPosition(id)
		if !ok || y < 1.5 || y > 2.5 || z < -0.5 || z > 0.5 {
			continue
		}
		if x < 1 {
			aID = id
		} else {
			bID, bx = id, x
		}
	}
	if aID == 0 || bID == 0 {
		t.Fatal("scene load dropped a joint body")
	}
	w2.phys3.SetGravity(0, 0, 0)
	w2.phys3.Wake(aID)
	w2.phys3.Wake(bID)
	w2.phys3.SetVelocity(aID, 6, 0, 0)
	for i := 0; i < 45; i++ {
		w2.phys3.Step(1.0 / 60)
	}
	nx, _, _, ok := w2.phys3.GetPosition(bID)
	if !ok || nx < bx+0.4 {
		t.Fatalf("fixed joint did not pull the other body: start %v now %v", bx, nx)
	}
}

func TestSceneJSONRestoresLook(t *testing.T) {
	w := New(".")
	id := w.createCubeMesh(nil)
	p := w.convertEntityPBR(w.ents[id])
	if p == nil {
		t.Fatal("no pbr material")
	}
	p.metallic = 0.8
	p.roughness = 0.25
	p.ao = 0.9
	p.albedo = math32.Color4{0.2, 0.4, 0.6, 1}
	p.emissive = math32.Color{0.1, 0.0, 0.0}
	p.applyFactors()
	w.ents[id].albedoPath = "assets/missing.png"

	flag := w.createCloth(2, 2, 4, 4, 1)
	n := w.phys3.ClothVertexCount(flag)
	buf := make([]float32, n*3)
	if got := w.phys3.ClothVertices(flag, buf); got < 4 {
		t.Fatalf("cloth verts %d", got)
	}
	buf[3] += 0.35
	w.phys3.SetClothVertices(flag, buf)

	hip := core.NewNode()
	hip.SetPosition(toG3N(0.4, 1.2, 0))
	w.ents[id].skin = &meshSkin{joints: []skinJoint{{name: "hip", node: hip, bindX: 0, bindY: 1, bindZ: 0}}}

	raw, err := json.Marshal(w.sceneJSON())
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	w2 := New(".")
	w2.applySceneJSON(root)

	var found *Entity
	for _, e := range w2.ents {
		if e != nil && e.pbrWrap != nil && e.pbrWrap.metallic > 0.7 {
			found = e
		}
	}
	if found == nil || found.pbrWrap.roughness < 0.2 || found.pbrWrap.roughness > 0.3 || found.albedoPath != "assets/missing.png" {
		t.Fatalf("material was not restored: %+v", found)
	}

	var posed bool
	for id2, c := range w2.cloths {
		if c == nil || c.nx != 4 {
			continue
		}
		got := make([]float32, c.nx*c.ny*3)
		if n := w2.phys3.ClothVertices(id2, got); n < 4 {
			t.Fatalf("reloaded cloth verts %d", n)
		}
		if got[3] > buf[3]-0.05 && got[3] < buf[3]+0.05 {
			posed = true
		}
	}
	if !posed {
		t.Fatal("cloth pose was flattened")
	}

	again := &Entity{skin: &meshSkin{joints: []skinJoint{{name: "hip", node: core.NewNode()}}}}
	w2.restoreSceneSkin(again, map[string]any{"joints": sceneSkinJoints(&Entity{skin: w.ents[id].skin})})
	p2 := again.skin.joints[0].node.Position()
	x, y, z := fromG3N(p2.X, p2.Y, p2.Z)
	if x < 0.3 || x > 0.5 || y < 1.1 || y > 1.3 {
		t.Fatalf("skin joint (%v,%v,%v)", x, y, z)
	}
}
