package runtime

import (
	"math"

	bsaudio "moonbasic/internal/audio"
)

// engineLoop keeps a low loop on a car, bike, or tank. Volume and pitch follow
// the throttle. A window must be open; tests without one stay silent.
func (w *World) engineLoop(id int, throttle, speed float32) {
	if w == nil || w.app == nil {
		return
	}
	if w.engineClip == nil {
		w.engineClip = bsaudio.GenerateEngine()
	}
	if w.engineVoices == nil {
		w.engineVoices = map[int]*bsaudio.Voice{}
	}
	drive := throttle
	if drive < 0 {
		drive = -drive
	}
	if speed < 0 {
		speed = -speed
	}
	if speed > 40 {
		speed = 40
	}
	voice := w.engineVoices[id]
	if drive < 0.02 && speed < 0.4 {
		if voice != nil {
			voice.SetVolume(0)
		}
		return
	}
	if voice == nil {
		started, err := w.engineClip.PlayAt(0.15, 0.8, 0, true)
		if err != nil {
			return
		}
		w.engineVoices[id] = started
		voice = started
	}
	voice.SetVolume(0.1 + float64(drive)*0.42)
	voice.SetPitch(0.72 + float64(drive)*0.65 + float64(speed)/90)
}

// footstep plays a thud each stride while a character is on the ground.
func (w *World) footstep(id int, hx, hz float32) {
	if w == nil || w.app == nil || w.phys3 == nil {
		return
	}
	if w.phys3.CharacterGroundState(id) != 0 {
		return
	}
	speed := float32(math.Hypot(float64(hx), float64(hz)))
	if w.stepDist == nil {
		w.stepDist = map[int]float32{}
	}
	next, play := stepDue(w.stepDist[id], speed, w.controllerDT())
	w.stepDist[id] = next
	if !play {
		return
	}
	if w.stepClip == nil {
		w.stepClip = bsaudio.GenerateFootstep()
	}
	x, y, z := w.blitzPosOf(id)
	lx, ly, lz, yaw := w.listenerPose()
	vol, pan := bsaudio.DistancePan(lx, ly, lz, yaw, float64(x), float64(y), float64(z), 28)
	if vol <= 0 {
		return
	}
	if w.stepVoice != nil {
		w.stepVoice.Close()
	}
	voice, err := w.stepClip.PlayAt(vol*0.55, 0.9+float64(speed)/18, pan, false)
	if err != nil {
		return
	}
	w.stepVoice = voice
}

func stepDue(accum, speed, dt float32) (float32, bool) {
	if speed < 0.8 || dt <= 0 {
		return accum, false
	}
	accum += speed * dt
	stride := float32(1.55)
	if speed > 7 {
		stride = 2.1
	}
	if accum < stride {
		return accum, false
	}
	return accum - stride, true
}

// clothWindAudio loops a soft rustle while any sheet has wind.
func (w *World) clothWindAudio() {
	if w == nil || w.app == nil || len(w.cloths) == 0 {
		return
	}
	wind := float32(0)
	for _, c := range w.cloths {
		if c != nil && c.wind > wind {
			wind = c.wind
		}
	}
	if w.clothClip == nil {
		w.clothClip = bsaudio.GenerateCloth()
	}
	level := float64(wind) / 16
	if level > 1 {
		level = 1
	}
	if level < 0.04 {
		if w.clothVoice != nil {
			w.clothVoice.SetVolume(0)
		}
		return
	}
	if w.clothVoice == nil {
		voice, err := w.clothClip.PlayAt(0.05, 1, 0, true)
		if err != nil {
			return
		}
		w.clothVoice = voice
	}
	w.clothVoice.SetVolume(level * 0.38)
	w.clothVoice.SetPitch(0.85 + level*0.4)
}
