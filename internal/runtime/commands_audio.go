package runtime

import (
	"fmt"

	bsaudio "bitshinbasic/internal/audio"
	"bitshinbasic/internal/value"
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
			if s := w.sounds[argI(a, 0, 0)]; s != nil && s.voice != nil {
				s.voice.Pause()
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
			if s.voice != nil {
				s.voice.SetVolume(s.vol)
			}
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
				if s != nil && s.voice != nil {
					s.voice.SetVolume(s.vol)
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
			lx, ly, lz, yaw := w.listenerPose()
			vol, pan := bsaudio.DistancePan(lx, ly, lz, yaw, x, y, zpos, 40)
			voice, err := s.clip.PlayAt(vol*s.vol, s.pitch, pan, false)
			if err != nil {
				fmt.Println("EmitSound:", err)
				return z()
			}
			s.voice = voice
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
	clip, err := bsaudio.Load(path)
	if err != nil {
		return value.Num(0), nil
	}
	id := w.nextSnd
	w.nextSnd++
	w.sounds[id] = &sndSlot{path: path, clip: clip, vol: 1, pitch: 1, music: music}
	if music {
		w.musicID = id
	}
	return value.Num(float64(id)), nil
}

func (w *World) startClip(id int, loop bool) (v value.Value, err error) {
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
	voice, playErr := s.clip.PlayAt(s.vol, s.pitch, 0, loop || s.music)
	if playErr != nil {
		fmt.Println("PlaySound:", playErr)
		return value.Num(0), nil
	}
	s.voice = voice
	if s.music {
		w.musicID = id
	}
	return value.Num(0), nil
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
