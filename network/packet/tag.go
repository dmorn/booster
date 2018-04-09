/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package packet

import (
	"bufio"
	"fmt"
	"io"
)

type TagReader struct {
	tag string
	r   io.Reader

	tagRaw []byte
	cur    int
}

func NewTagReader(r io.Reader, tag string) *TagReader {
	if _, ok := r.(io.ByteReader); !ok {
		r = bufio.NewReader(r)
	}

	return &TagReader{
		r:      r,
		tag:    tag,
		tagRaw: []byte(tag),
		cur:    0,
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

		ct := t.tagRaw[t.cur]
		if b != ct {
			return n, fmt.Errorf("unexpected tag: wanted %v, found %v", string(ct), string(b))
		}
		t.cur++
	}
}

type TagWriter struct {
	w io.Writer
}

func NewTagWriter(w io.Writer) *TagWriter {
	return &TagWriter{
		w: w,
	}
}

func (w *TagWriter) Write(tag string) (int, error) {
	buf := []byte(tag)
	n, err := w.w.Write(buf)
	if err != nil {
		return n, fmt.Errorf("unable to write tag (%v): %v", tag, err)
	}

	return n, err
}
