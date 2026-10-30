package runtime

import (
	"bufio"
	"encoding/binary"
	"io"
	"math"
	"os"

	"moonbasic/internal/value"
)

// bbFile is one Blitz-style file handle. Reads are buffered; writes and
// seeks drop that buffer so FilePos stays the logical offset.
type bbFile struct {
	f    *os.File
	r    *bufio.Reader
	mode int // 1 read, 2 write, 3 read/write
}

func (w *World) fileTable() map[int]*bbFile {
	st := w.extState()
	if st.files == nil {
		st.files = map[int]*bbFile{}
	}
	return st.files
}

func (w *World) openBBFile(path string, flag int, mode int) (value.Value, error) {
	f, err := os.OpenFile(w.resolve(path), flag, 0o644)
	if err != nil {
		return value.Num(0), nil
	}
	st := w.extState()
	if st.nextFile < 1 {
		st.nextFile = 1
	}
	id := st.nextFile
	st.nextFile++
	w.fileTable()[id] = &bbFile{f: f, mode: mode}
	return value.Num(float64(id)), nil
}

func (w *World) bbFile(id int) *bbFile {
	if id <= 0 {
		return nil
	}
	return w.fileTable()[id]
}

func (b *bbFile) reader() *bufio.Reader {
	if b.r == nil {
		b.r = bufio.NewReader(b.f)
	}
	return b.r
}

func (b *bbFile) pos() (int64, error) {
	p, err := b.f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	if b.r != nil {
		p -= int64(b.r.Buffered())
	}
	return p, nil
}

func (b *bbFile) seek(off int64) error {
	if off < 0 {
		off = 0
	}
	b.r = nil
	_, err := b.f.Seek(off, io.SeekStart)
	return err
}

func (b *bbFile) prepareWrite() error {
	p, err := b.pos()
	if err != nil {
		return err
	}
	b.r = nil
	_, err = b.f.Seek(p, io.SeekStart)
	return err
}

func (b *bbFile) eof() bool {
	if b.mode == 2 {
		return false
	}
	_, err := b.reader().Peek(1)
	return err != nil
}

func (b *bbFile) readLine() string {
	line, err := b.reader().ReadString('\n')
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}
	if err != nil && err != io.EOF {
		return line
	}
	return line
}

func (w *World) fileCommands(n func(func([]value.Value) (value.Value, error)) cmd) map[string]cmd {
	return map[string]cmd{
		"readfile": n(func(a []value.Value) (value.Value, error) {
			return w.openBBFile(argS(a, 0), os.O_RDONLY, 1)
		}),
		"writefile": n(func(a []value.Value) (value.Value, error) {
			return w.openBBFile(argS(a, 0), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 2)
		}),
		"openfile": n(func(a []value.Value) (value.Value, error) {
			return w.openBBFile(argS(a, 0), os.O_RDWR, 3)
		}),
		"closefile": n(func(a []value.Value) (value.Value, error) {
			id := argI(a, 0, 0)
			b := w.bbFile(id)
			if b == nil {
				return value.Num(0), nil
			}
			err := b.f.Close()
			delete(w.fileTable(), id)
			if err != nil {
				return value.Num(0), nil
			}
			return value.Num(1), nil
		}),
		"eof": n(func(a []value.Value) (value.Value, error) {
			b := w.bbFile(argI(a, 0, 0))
			if b == nil || b.eof() {
				return value.Num(1), nil
			}
			return value.Num(0), nil
		}),
		"filepos": n(func(a []value.Value) (value.Value, error) {
			b := w.bbFile(argI(a, 0, 0))
			if b == nil {
				return value.Num(0), nil
			}
			p, err := b.pos()
			if err != nil {
				return value.Num(0), nil
			}
			return value.Num(float64(p)), nil
		}),
		"seekfile": n(func(a []value.Value) (value.Value, error) {
			b := w.bbFile(argI(a, 0, 0))
			if b == nil {
				return value.Num(0), nil
			}
			if err := b.seek(int64(argI(a, 1, 0))); err != nil {
				return value.Num(0), nil
			}
			return value.Num(1), nil
		}),
		"readline": n(func(a []value.Value) (value.Value, error) {
			b := w.bbFile(argI(a, 0, 0))
			if b == nil {
				return value.Str(""), nil
			}
			return value.Str(b.readLine()), nil
		}),
		"writeline": n(func(a []value.Value) (value.Value, error) {
			b := w.bbFile(argI(a, 0, 0))
			if b == nil || b.mode == 1 {
				return value.Num(0), nil
			}
			if err := b.prepareWrite(); err != nil {
				return value.Num(0), nil
			}
			if _, err := b.f.WriteString(argS(a, 1) + "\n"); err != nil {
				return value.Num(0), nil
			}
			return value.Num(1), nil
		}),
		"readbyte": n(func(a []value.Value) (value.Value, error) {
			b := w.bbFile(argI(a, 0, 0))
			if b == nil {
				return value.Num(0), nil
			}
			c, err := b.reader().ReadByte()
			if err != nil {
				return value.Num(0), nil
			}
			return value.Num(float64(c)), nil
		}),
		"writebyte": n(func(a []value.Value) (value.Value, error) {
			return w.writeBytes(argI(a, 0, 0), []byte{byte(argI(a, 1, 0))})
		}),
		"readshort": n(func(a []value.Value) (value.Value, error) {
			return w.readIntN(argI(a, 0, 0), 2)
		}),
		"writeshort": n(func(a []value.Value) (value.Value, error) {
			var buf [2]byte
			binary.LittleEndian.PutUint16(buf[:], uint16(int16(argI(a, 1, 0))))
			return w.writeBytes(argI(a, 0, 0), buf[:])
		}),
		"readint": n(func(a []value.Value) (value.Value, error) {
			return w.readIntN(argI(a, 0, 0), 4)
		}),
		"writeint": n(func(a []value.Value) (value.Value, error) {
			var buf [4]byte
			binary.LittleEndian.PutUint32(buf[:], uint32(int32(argI(a, 1, 0))))
			return w.writeBytes(argI(a, 0, 0), buf[:])
		}),
		"readfloat": n(func(a []value.Value) (value.Value, error) {
			b := w.bbFile(argI(a, 0, 0))
			if b == nil {
				return value.Num(0), nil
			}
			var bits uint32
			if err := binary.Read(b.reader(), binary.LittleEndian, &bits); err != nil {
				return value.Num(0), nil
			}
			return value.Num(float64(math.Float32frombits(bits))), nil
		}),
		"writefloat": n(func(a []value.Value) (value.Value, error) {
			var buf [4]byte
			binary.LittleEndian.PutUint32(buf[:], math.Float32bits(float32(argN(a, 1, 0))))
			return w.writeBytes(argI(a, 0, 0), buf[:])
		}),
		"readstring": n(func(a []value.Value) (value.Value, error) {
			b := w.bbFile(argI(a, 0, 0))
			if b == nil {
				return value.Str(""), nil
			}
			var n int32
			if err := binary.Read(b.reader(), binary.LittleEndian, &n); err != nil || n < 0 || n > 1<<24 {
				return value.Str(""), nil
			}
			buf := make([]byte, n)
			if _, err := io.ReadFull(b.reader(), buf); err != nil {
				return value.Str(""), nil
			}
			return value.Str(string(buf)), nil
		}),
		"writestring": n(func(a []value.Value) (value.Value, error) {
			s := argS(a, 1)
			buf := make([]byte, 4+len(s))
			binary.LittleEndian.PutUint32(buf[:4], uint32(len(s)))
			copy(buf[4:], s)
			return w.writeBytes(argI(a, 0, 0), buf)
		}),
		"filesize": n(func(a []value.Value) (value.Value, error) {
			st, err := os.Stat(w.resolve(argS(a, 0)))
			if err != nil || st.IsDir() {
				return value.Num(0), nil
			}
			return value.Num(float64(st.Size())), nil
		}),
	}
}

func (w *World) writeBytes(id int, buf []byte) (value.Value, error) {
	b := w.bbFile(id)
	if b == nil || b.mode == 1 {
		return value.Num(0), nil
	}
	if err := b.prepareWrite(); err != nil {
		return value.Num(0), nil
	}
	if _, err := b.f.Write(buf); err != nil {
		return value.Num(0), nil
	}
	return value.Num(1), nil
}

func (w *World) readIntN(id, size int) (value.Value, error) {
	b := w.bbFile(id)
	if b == nil {
		return value.Num(0), nil
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(b.reader(), buf); err != nil {
		return value.Num(0), nil
	}
	switch size {
	case 2:
		return value.Num(float64(int16(binary.LittleEndian.Uint16(buf)))), nil
	default:
		return value.Num(float64(int32(binary.LittleEndian.Uint32(buf)))), nil
	}
}
