package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestCameraHelpers(t *testing.T) {
	w := New(".")
	cmds := w.commandTable()

	cam := w.spawnCamera(0)
	target := w.createCubeMesh(nil)

	// 1. CameraSpringArm
	_, err := cmds["cameraspringarm"]([]value.Value{
		value.Num(float64(cam)),
		value.Num(float64(target)),
		value.Num(1.5),
		value.Num(5.0),
		value.Num(0.25),
		value.Num(12.0),
		value.Num(30.0),
		value.Num(15.0),
	})
	if err != nil {
		t.Fatalf("cameraspringarm failed: %v", err)
	}

	// 2. CameraOrbit
	_, err = cmds["cameraorbit"]([]value.Value{
		value.Num(float64(cam)),
		value.Num(float64(target)),
		value.Num(8.0),
		value.Num(2.5),
		value.Num(45.0),
		value.Num(20.0),
	})
	if err != nil {
		t.Fatalf("cameraorbit failed: %v", err)
	}

	// 3. CameraShake
	cmds["camerashake"]([]value.Value{value.Num(0.8), value.Num(0.5), value.Num(24.0)})
	if w.shakeTrauma < 0.79 {
		t.Fatalf("camerashake trauma not set: %v", w.shakeTrauma)
	}
	w.delta = 0.016
	cmds["updatecamerashake"]([]value.Value{value.Num(float64(cam))})

	// 4. CameraSmoothLook
	_, err = cmds["camerasmoothlook"]([]value.Value{
		value.Num(float64(cam)),
		value.Num(10),
		value.Num(5),
		value.Num(20),
		value.Num(8.0),
	})
	if err != nil {
		t.Fatalf("camerasmoothlook failed: %v", err)
	}

	// 5. CameraRTS
	_, err = cmds["camerarts"]([]value.Value{
		value.Num(float64(cam)),
		value.Num(50),
		value.Num(50),
		value.Num(25.0),
		value.Num(60.0),
		value.Num(45.0),
	})
	if err != nil {
		t.Fatalf("camerarts failed: %v", err)
	}
}
