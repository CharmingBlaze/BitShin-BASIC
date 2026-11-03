package runtime

import (
	"fmt"
	"strings"

	bsaudio "moonbasic/internal/audio"
	"moonbasic/internal/value"
)

func (w *World) audioCommands(n func(func([]value.Value) (value.Value, error)) cmd, z func() (value.Value, error)) map[string]cmd {
	return map[string]cmd{
		"loadsound": n(func(a []value.Value) (value.Value, error) {
			return w.loadClip(a, false)
		}),
		"loadmusic": n(func(a []value.Value) (value.Value, error) {
			return w.loadClip(a, true)
		}),
		"playsound": n(func(a []value.Value) (value.Value, error) {
			return w.startClip(argI(a, 0, 0), false)
		}),
		"playmusic": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, w.musicID)
			if id == 0 {
				id = w.musicID
			}
			return w.startClip(id, true)
		}),
		"loopsound": n(func(a []value.Value) (value.Value, error) {
			return w.startClip(argI(a, 0, 0), true)
		}),
		"stopsound": n(func(a []value.Value) (value.Value, error) {
			w.pauseSound(argI(a, 0, 0))
			return z()
		}),
		"pausesound": n(func(a []value.Value) (value.Value, error) {
			w.pauseSound(argI(a, 0, 0))
			return z()
		}),
		"resumesound": n(func(a []value.Value) (value.Value, error) {
			w.resumeSound(argI(a, 0, 0))
			return z()
		}),
		"setsoundpan": n(func(a []value.Value) (value.Value, error) {
			w.setSoundPan(argI(a, 0, 0), argN(a, 1, 0))
			return z()
		}),
		"seeksound": n(func(a []value.Value) (value.Value, error) {
			if w.seekSound(argI(a, 0, 0), argN(a, 1, 0)) {
				return value.Num(1), nil
			}
			return z()
		}),
		"soundtime": n(func(a []value.Value) (value.Value, error) {
			return value.Num(w.soundTime(argI(a, 0, 0))), nil
		}),
		"fadesound": n(func(a []value.Value) (value.Value, error) {
			w.fadeSound(argI(a, 0, 0), argN(a, 1, 0), argN(a, 2, 0))
			return z()
		}),
		"setaudiobus": n(func(a []value.Value) (value.Value, error) {
			w.setAudioBus(argS(a, 0), argN(a, 1, 1))
			return z()
		}),
		"soundbus": n(func(a []value.Value) (value.Value, error) {
			w.soundBus(argI(a, 0, 0), argS(a, 1))
			return z()
		}),
		"setmastervolume": n(func(a []value.Value) (value.Value, error) {
			w.master = argN(a, 0, 1)
			if w.master < 0 {
				w.master = 0
			}
			if w.master > 1 {
				w.master = 1
			}
			for _, s := range w.sounds {
				w.applySoundVol(s)
			}
			return z()
		}),
		"setsoundecho": n(func(a []value.Value) (value.Value, error) {
			if w.setSoundEcho(argI(a, 0, 0), argN(a, 1, 0)) {
				return value.Num(1), nil
			}
			return z()
		}),
		"stopmusic": n(func(a []value.Value) (value.Value, error) {
			if s := w.sounds[w.musicID]; s != nil && s.voice != nil {
				s.voice.Pause()
			}
			return z()
		}),
		"freesound": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			if s := w.sounds[id]; s != nil {
				if s.voice != nil {
					s.voice.Close()
				}
			}
			delete(w.sounds, id)
			return z()
		}),
		"setsoundvolume": n(func(a []value.Value) (value.Value, error) {
			s := w.sounds[argI(a, 0, 0)]
			if s == nil {
				return z()
			}
			s.vol = argN(a, 1, 1)
			s.fadeDur = 0
			w.applySoundVol(s)
			return z()
		}),
		"setmusicvolume": n(func(a []value.Value) (value.Value, error) {
			if s := w.sounds[w.musicID]; s != nil {
				s.vol = argN(a, 0, 1)
				if len(a) > 1 {
					s = w.sounds[argI(a, 0, w.musicID)]
					if s != nil {
						s.vol = argN(a, 1, 1)
					}
				}
				if s != nil {
					s.fadeDur = 0
					w.applySoundVol(s)
				}
			}
			return z()
		}),
		"setsoundpitch": n(func(a []value.Value) (value.Value, error) {
			s := w.sounds[argI(a, 0, 0)]
			if s == nil {
				return z()
			}
			s.pitch = argN(a, 1, 1)
			if s.pitch <= 0 {
				s.pitch = 1
			}
			if s.voice != nil {
				s.voice.SetPitch(s.pitch)
			}
			return z()
		}),
		"emitsound": n(func(a []value.Value) (value.Value, error) {
			s := w.sounds[argI(a, 0, 0)]
			if s == nil || s.clip == nil {
				return z()
			}
			x, y, zpos := argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
			if len(a) == 2 {
				if e := w.ents[argI(a, 1, 0)]; e != nil && e.node != nil {
					p := worldPos(e.node.GetNode())
					fx, fy, fz := fromG3N(p.X, p.Y, p.Z)
					x, y, zpos = float64(fx), float64(fy), float64(fz)
				}
			}
			rangeDist := s.maxDist
			if rangeDist <= 0 {
				rangeDist = 40
			}
			w.debugNoteSound(float32(x), float32(y), float32(zpos), float32(rangeDist))
			lx, ly, lz, yaw := w.listenerPose()
			vol, pan := bsaudio.DistancePan(lx, ly, lz, yaw, x, y, zpos, rangeDist)
			voice, err := s.clip.PlayAt(vol*s.vol, s.pitch, pan, false)
			if err != nil {
				fmt.Println("EmitSound:", err)
				return z()
			}
			s.voice = voice
			return z()
		}),
		"emitsound3d": n(func(a []value.Value) (value.Value, error) {
			s := w.sounds[argI(a, 0, 0)]
			if s == nil || s.clip == nil {
				return z()
			}
			x, y, zpos := argN(a, 1, 0), argN(a, 2, 0), argN(a, 3, 0)
			maxD := s.maxDist
			if maxD <= 0 {
				maxD = 40
			}
			if len(a) >= 5 {
				maxD = argN(a, 4, maxD)
			}
			if len(a) == 2 {
				if e := w.ents[argI(a, 1, 0)]; e != nil && e.node != nil {
					p := worldPos(e.node.GetNode())
					fx, fy, fz := fromG3N(p.X, p.Y, p.Z)
					x, y, zpos = float64(fx), float64(fy), float64(fz)
				}
			}
			w.debugNoteSound(float32(x), float32(y), float32(zpos), float32(maxD))
			lx, ly, lz, yaw := w.listenerPose()
			vol, pan := bsaudio.DistancePan(lx, ly, lz, yaw, x, y, zpos, maxD)
			voice, err := s.clip.PlayAt(vol*s.vol, s.pitch, pan, false)
			if err != nil {
				fmt.Println("EmitSound3D:", err)
				return z()
			}
			s.voice = voice
			return z()
		}),
		"setsoundrange": n(func(a []value.Value) (value.Value, error) {
			s := w.sounds[argI(a, 0, 0)]
			if s == nil {
				return z()
			}
			s.minDist = argN(a, 1, 1)
			s.maxDist = argN(a, 2, 40)
			if s.maxDist <= s.minDist {
				s.maxDist = s.minDist + 1
			}
			return z()
		}),
		"setlistener": n(func(a []value.Value) (value.Value, error) {
			w.listenEnt = argI(a, 0, 0)
			return z()
		}),
	}
}

func (w *World) loadClip(a []value.Value, music bool) (value.Value, error) {
	file := argS(a, 0)
	if len(a) >= 2 && a[0].Kind != value.KindStr {
		file = argS(a, 1)
	}
	path, err := w.openPath(file)
	if err != nil {
		return value.Num(0), nil
	}
	// Music streams from disk while it plays; sounds decode up front so
	// they can start instantly and overlap.
	load := bsaudio.Load
	if music {
		load = bsaudio.LoadStream
	}
	clip, err := load(path)
	if err != nil {
		return value.Num(0), nil
	}
	id := w.nextSnd
	w.nextSnd++
	w.sounds[id] = &sndSlot{path: path, clip: clip, vol: 1, pitch: 1, minDist: 1, maxDist: 40, music: music}
	if music {
		w.musicID = id
	}
	return value.Num(float64(id)), nil
}

func (w *World) startClip(id int, loop bool) (v value.Value, err error) {
	return w.startClipWithVol(id, loop, 1.0)
}

func (w *World) startClipWithVol(id int, loop bool, volScale float64) (v value.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("PlaySound:", r)
			v, err = value.Num(0), nil
		}
	}()
	s := w.sounds[id]
	if s == nil || s.clip == nil {
		return value.Num(0), nil
	}
	if s.voice != nil {
		s.voice.Close()
	}
	vFinal := s.vol * volScale * w.busGain(s.bus) * w.master
	if vFinal > 1.0 {
		vFinal = 1.0
	}
	if vFinal < 0 {
		vFinal = 0
	}
	voice, playErr := s.clip.PlayAt(vFinal, s.pitch, s.pan, loop || s.music)
	if playErr != nil {
		fmt.Println("PlaySound:", playErr)
		return value.Num(0), nil
	}
	s.voice = voice
	if s.echo > 0 {
		s.voice.SetEcho(s.echo)
	}
	if s.music {
		w.musicID = id
	}
	return value.Num(0), nil
}

func (w *World) pauseSound(id int) {
	if s := w.sounds[id]; s != nil && s.voice != nil {
		s.voice.Pause()
	}
}

func (w *World) resumeSound(id int) {
	s := w.sounds[id]
	if s == nil {
		return
	}
	if s.voice != nil {
		s.voice.Play()
		return
	}
	_, _ = w.startClip(id, s.music)
}

func (w *World) setSoundPan(id int, pan float64) {
	s := w.sounds[id]
	if s == nil {
		return
	}
	if pan < -1 {
		pan = -1
	}
	if pan > 1 {
		pan = 1
	}
	s.pan = pan
	if s.voice != nil {
		s.voice.SetPan(pan)
	}
}

func (w *World) seekSound(id int, seconds float64) bool {
	s := w.sounds[id]
	if s == nil || s.voice == nil {
		return false
	}
	return s.voice.Seek(seconds)
}

func (w *World) soundTime(id int) float64 {
	s := w.sounds[id]
	if s == nil || s.voice == nil {
		return 0
	}
	t := s.voice.Seconds()
	if t < 0 {
		return 0
	}
	return t
}

func (w *World) fadeSound(id int, to, secs float64) {
	s := w.sounds[id]
	if s == nil {
		return
	}
	if to < 0 {
		to = 0
	}
	if to > 1 {
		to = 1
	}
	if secs <= 0 {
		s.vol = to
		s.fadeDur = 0
		w.applySoundVol(s)
		return
	}
	s.fadeFrom = s.vol
	s.fadeTo = to
	s.fadeT = 0
	s.fadeDur = secs
}

func (w *World) setAudioBus(name string, vol float64) {
	if w.buses == nil {
		w.buses = map[string]float64{}
	}
	if vol < 0 {
		vol = 0
	}
	if vol > 1 {
		vol = 1
	}
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return
	}
	w.buses[key] = vol
	for _, s := range w.sounds {
		if strings.ToLower(s.bus) == key {
			w.applySoundVol(s)
		}
	}
}

func (w *World) soundBus(id int, name string) {
	s := w.sounds[id]
	if s == nil {
		return
	}
	s.bus = strings.TrimSpace(name)
	w.applySoundVol(s)
}

func (w *World) setSoundEcho(id int, wet float64) bool {
	s := w.sounds[id]
	if s == nil {
		return false
	}
	if wet < 0 {
		wet = 0
	}
	if wet > 1 {
		wet = 1
	}
	s.echo = wet
	if s.voice == nil {
		return true
	}
	return s.voice.SetEcho(wet)
}

func (w *World) busGain(name string) float64 {
	if name == "" || w.buses == nil {
		return 1
	}
	if g, ok := w.buses[strings.ToLower(name)]; ok {
		return g
	}
	return 1
}

func (w *World) applySoundVol(s *sndSlot) {
	if s == nil || s.voice == nil {
		return
	}
	v := s.vol * w.busGain(s.bus) * w.master
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	s.voice.SetVolume(v)
}

func (w *World) tickFades() {
	d := w.delta
	if d <= 0 {
		return
	}
	for _, s := range w.sounds {
		if s == nil || s.fadeDur <= 0 {
			continue
		}
		s.fadeT += d
		u := s.fadeT / s.fadeDur
		if u >= 1 {
			u = 1
			s.fadeDur = 0
		}
		s.vol = s.fadeFrom + (s.fadeTo-s.fadeFrom)*u
		w.applySoundVol(s)
	}
}

func (w *World) listenerPose() (x, y, z, yaw float64) {
	id := w.listenEnt
	if id == 0 {
		for eid, e := range w.ents {
			if e != nil && e.cam != nil && e.cam == w.cam {
				id = eid
				break
			}
		}
	}
	e := w.ents[id]
	if e == nil || e.node == nil {
		return 0, 0, 0, 0
	}
	p := worldPos(e.node.GetNode())
	fx, fy, fz := fromG3N(p.X, p.Y, p.Z)
	return float64(fx), float64(fy), float64(fz), float64(e.yaw)
}
