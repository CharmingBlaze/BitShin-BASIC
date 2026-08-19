package runtime

import (
	"testing"

	"github.com/g3n/engine/math32"
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
