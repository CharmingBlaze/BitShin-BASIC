package runtime

import (
	"moonbasic/internal/value"
)

// Far cascades take turns. The nearest redraws every frame; the second every
// other frame; the rest once in four. A resting cascade keeps both its picture
// and the view it was drawn with, so what it shows stays correct, only slightly
// old. In a mostly still landscape that is most of the shadow cost saved, and a
// moving object far off lags by at most a few frames (CRYENGINE and Cyberpunk
// refresh distant cascades less often in the same way).

// shadowCascadeDue reports whether cascade i redraws on this frame.
func shadowCascadeDue(i, frame int) bool {
	switch i {
	case 0:
		return true
	case 1:
		return frame%2 == 0
	case 2:
		return frame%4 == 1
	default:
		return frame%4 == 3
	}
}

// planShadowTurns decides which cascades rest this frame. Anything that makes
// old tiles wrong draws them all: the sun moving, a camera jump, or a new atlas.
// Moment filters blur the whole atlas, so they always draw everything.
func (w *World) planShadowTurns(n int) {
	s := &w.shadow
	s.turnFrame++
	pos := s.tmpCamPos
	sun := s.tmpDir
	switch {
	case s.turnN != n, s.turnSize != s.size:
		s.turnFull = 2
	case sun.DistanceTo(&s.turnSun) > 0.002:
		s.turnFull = 2
	case pos.DistanceTo(&s.turnCam) > 40:
		s.turnFull = 2
	}
	s.turnN, s.turnSize, s.turnSun, s.turnCam = n, s.size, sun, pos

	s.turnsOn = !s.turnsOff && s.filter <= shadowFilterPCSS && !s.cache
	if !s.turnsOn || s.turnFull > 0 || !s.ready {
		s.turnFull = max(s.turnFull-1, 0)
		s.turnsOn = false
		s.resting = [maxShadowCascades]bool{}
		s.drawnVP = s.lightVP
		return
	}
	for i := 0; i < n && i < maxShadowCascades; i++ {
		s.resting[i] = !shadowCascadeDue(i, s.turnFrame)
		if s.resting[i] {
			s.lightVP[i] = s.drawnVP[i]
		} else {
			s.drawnVP[i] = s.lightVP[i]
		}
	}
}

func (w *World) shadowTurnCommands(n func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		// SetShadowTurns on: far shadow cascades redraw in turn, not every frame.
		// On by default. Turn it off if something moves fast far from the camera.
		"setshadowturns": n(func(a []value.Value) (value.Value, error) {
			w.shadow.turnsOff = argI(a, 0, 1) == 0
			return value.Num(float64(b01(!w.shadow.turnsOff))), nil
		}),
	}
}
