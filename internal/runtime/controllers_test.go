package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestCharacterAndVehicleControllers(t *testing.T) {
	w := New(".")
	cmds := w.commandTable()

	// 1. FPS Controller
	player := w.createCubeMesh(nil)
	cam := w.spawnCamera(0)
	cmds["createfpscontroller"]([]value.Value{value.Num(float64(player)), value.Num(float64(cam)), value.Num(1.8), value.Num(0.4), value.Num(6.0), value.Num(10.0), value.Num(8.0)})
	if w.fpsControllers[player] == nil {
		t.Fatalf("createfpscontroller failed to create fpsCtrl")
	}

	w.delta = 0.016
	cmds["updatefps"]([]value.Value{value.Num(float64(player)), value.Num(0), value.Num(1.0), value.Num(0), value.Num(1), value.Num(0), value.Num(0.15)})
	res, _ := cmds["getfpsyaw"]([]value.Value{value.Num(float64(player))})
	if res.Num != 0 {
		t.Fatalf("initial fps yaw unexpected: %v", res)
	}

	// 2. TPS Controller
	tpsPlayer := w.createCubeMesh(nil)
	tpsCam := w.spawnCamera(0)
	cmds["createtpscontroller"]([]value.Value{value.Num(float64(tpsPlayer)), value.Num(float64(tpsCam)), value.Num(1.8), value.Num(0.45), value.Num(6.5), value.Num(11.0), value.Num(9.0)})
	if w.tpsControllers[tpsPlayer] == nil {
		t.Fatalf("createtpscontroller failed to create tpsCtrl")
	}
	cmds["updatetps"]([]value.Value{value.Num(float64(tpsPlayer)), value.Num(1.0), value.Num(0), value.Num(0), value.Num(0), value.Num(0), value.Num(5.0), value.Num(15.0), value.Num(45.0)})

	// 3. Top-Down Controller
	tdPlayer := w.createCubeMesh(nil)
	cmds["createtopdowncontroller"]([]value.Value{value.Num(float64(tdPlayer)), value.Num(8.0), value.Num(18.0)})
	if w.topDownControllers[tdPlayer] == nil {
		t.Fatalf("createtopdowncontroller failed to create topDownCtrl")
	}
	cmds["updatetopdown"]([]value.Value{value.Num(float64(tdPlayer)), value.Num(1.0), value.Num(0), value.Num(10.0), value.Num(0), value.Num(1)})
	res, _ = cmds["gettopdownaimyaw"]([]value.Value{value.Num(float64(tdPlayer))})
	if res.Num <= 0 {
		t.Fatalf("top down aim yaw failed: got %v", res)
	}

	// 4. Platformer Controller
	platPlayer := w.createCubeMesh(nil)
	cmds["createplatformercontroller"]([]value.Value{value.Num(float64(platPlayer)), value.Num(8.5), value.Num(12.0), value.Num(2)})
	if w.platformerControllers[platPlayer] == nil {
		t.Fatalf("createplatformercontroller failed to create platformerCtrl")
	}
	cmds["updateplatformer"]([]value.Value{value.Num(float64(platPlayer)), value.Num(1.0), value.Num(1), value.Num(1)})

	// 5. Mech Controller
	mech := w.createCubeMesh(nil)
	cmds["createmechcontroller"]([]value.Value{value.Num(float64(mech))})
	if w.vehCtrls[mech] == nil {
		t.Fatalf("createmechcontroller failed")
	}
	cmds["updatemech"]([]value.Value{value.Num(float64(mech)), value.Num(1.0), value.Num(0.5), value.Num(0)})

	// 6. Lunar Lander / Rocket Controller
	lander := w.createCubeMesh(nil)
	cmds["createlandercontroller"]([]value.Value{value.Num(float64(lander))})
	if w.vehCtrls[lander] == nil {
		t.Fatalf("createlandercontroller failed")
	}
	cmds["updatelander"]([]value.Value{value.Num(float64(lander)), value.Num(1.0), value.Num(0.2), value.Num(0), value.Num(0.1)})

	// 7. JetSki Controller
	jetski := w.createCubeMesh(nil)
	cmds["createjetskicontroller"]([]value.Value{value.Num(float64(jetski))})
	if w.vehCtrls[jetski] == nil {
		t.Fatalf("createjetskicontroller failed")
	}
	cmds["updatejetski"]([]value.Value{value.Num(float64(jetski)), value.Num(1.0), value.Num(0.7)})

	skier := w.createCubeMesh(nil)
	cmds["createwaterskicontroller"]([]value.Value{value.Num(float64(skier))})
	if w.vehCtrls[skier] == nil {
		t.Fatal("createwaterskicontroller failed")
	}
	cmds["updatewaterski"]([]value.Value{value.Num(float64(skier)), value.Num(1.0), value.Num(-0.4)})
}
