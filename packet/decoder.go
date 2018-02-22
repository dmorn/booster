package packet

import (
	"bufio"
	"io"
	"fmt"
)

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	d := new(Decoder)
	d.r = r

	return d
}

func (d *Decoder) Decode(packet *Packet) error {
	p := new(Packet)
	otr := NewTagReader(d.r, ">") // open tag reader
	ctr := NewTagReader(d.r, "<") // close tag reader
	md := NewModuleDecoder(d.r) // module decoder

	buf := make([]byte, 4) // 1 is also enough
	_, err := otr.Read(buf)
	if err != io.EOF {
		return err
	}

	// read modules number
	buf = buf[:2]
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return fmt.Errorf("packat: unable to read modules number: %v", err)
	}

	mn := int(buf[0])<<8 | int(buf[1])
	i := 0

	for {
		i++
		if i > mn {
			buf = buf[:4]
			if _, err := ctr.Read(buf); err != nil {
				if err == io.EOF {
					packet = p
					return nil // we're done
				} else {
					// we counldn't read the closing tags
					return err
				}
			}

			// no error occurred, it means that our buffer is too small
			// for the tag to be fully read, but we know that this is
			// not true.
			return fmt.Errorf("packet: unexpected closing tag: %s", buf)
		}

		// if no closing tag, a module must be present
		m := new(Module)
		if err = md.Decode(m); err != nil {
			return err
		}

		p.modules[m.ID] = m
	}
}

type ModuleDecoder struct {
	r io.Reader
}

func NewModuleDecoder(r io.Reader) *ModuleDecoder {
	d := new(ModuleDecoder)
	d.r = r

	return d
}

func (d *ModuleDecoder) Decode(module *Module) error {
	r := d.r
	m := new(Module)
	sr := NewTagReader(r, ":") // separator reader
	otr := NewTagReader(r, "[") // open tag reader
	ctr := NewTagReader(r, "]") // close tag reader

	// read module id
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("packet: unable to read module id: %v", err)
	}
	m.ID = string(buf)

	// separator
	if _, err := sr.Read(buf); err != nil {
		return err
	}
	sr.Flush()

	// read payload size
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("packet: unable to read payload size: %v", err)
	}
	m.Size = uint16(buf[0])<<8 | uint16(buf[1])

	// separator
	if _, err := sr.Read(buf); err != nil {
		return err
	}
	sr.Flush()

	// read encoding type
	buf = buf[:1]
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("packet: unable to read encoding type: %v", err)
	}
	m.Encoding = buf[0]

	// payload open tag
	if _, err := otr.Read(buf); err != nil {
		return err
	}

	buf = make([]byte, m.Size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return fmt.Errorf("packet: unable to read payload: %v", err)
	}
	copy(m.Payload, buf)

	// payload close tag
	if _, err := ctr.Read(buf); err != nil {
		return err
	}

	module = m

	return nil
}

type TagReader struct {
	tag string
	r io.Reader

	tagRaw []byte
	cur int
}

func NewTagReader(r io.Reader, tag string) *TagReader {
	if _, ok := r.(io.ByteReader); !ok {
		r = bufio.NewReader(r)
	}

	return &TagReader {
		r: r,
		tag: tag,
		tagRaw: []byte(tag),
		cur: 0,
	}
}

func (t *TagReader) Flush() error {
	t.cur = 0

	return nil
}

func (t *TagReader) Read(p []byte) (int, error) {
	n := 0
	buf := make([]byte, len(p))
	r := t.r.(io.ByteReader)

	defer func() {
		copy(p, buf)
	}()

	for {
		if t.cur == len(t.tagRaw) {
			return n, io.EOF
		}

		if n == len(p) {
			return n, nil
		}

		b, err := r.ReadByte()
		if err != nil {
			return n, err
		}
		buf[n] = b
		n++ // incr byte read count

		if b != t.tagRaw[t.cur] {
			return n, fmt.Errorf("packet: unexpected open tag: %v", b)
		}
		t.cur++
	}
}
