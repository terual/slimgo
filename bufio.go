// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.bufio file.

// Package bufio implements buffered I/O.  It wraps an io.Reader or io.Writer
// object, creating another object (Reader or Writer) that also implements
// the interface but provides buffering and some help for textual I/O.
package main

import (
	"strconv"
	"io"
	"os"
	//"fmt"
	//"encoding/binary"
)

// BufSizeError is the error representing an invalid buffer size.
type BufSizeError int

func (b BufSizeError) String() string {
	return "bufio: bad buffer size " + strconv.Itoa(int(b))
}

// Reader implements buffering for an io.Reader object.
type Reader struct {
	buf          []byte
	rd           io.Reader
	r, w         int
	err          os.Error
	lastByte     int
	lastRuneSize int
}

// NewReaderSize creates a new Reader whose buffer has the specified size,
// which must be greater than zero.  If the argument io.Reader is already a
// Reader with large enough size, it returns the underlying Reader.
// It returns the Reader and any error.
func (b *Reader) NewReaderSize(rd io.Reader, size int) (*Reader, os.Error) {
	if size <= 0 {
		return nil, BufSizeError(size)
	}
	// Is it already a Reader?
	//b, ok := rd.(*Reader)
	//if ok && len(b.buf) >= size {
	b.rd = rd
	b.lastByte = -1
	b.lastRuneSize = -1
	return b, nil
	/*} else {
		fmt.Println("NEW READER", ok)
		b = new(Reader)
		b.buf = make([]byte, size)
		b.rd = rd
		b.lastByte = -1
		b.lastRuneSize = -1
	}
	return b, nil*/
}

// fill reads a new chunk into the buffer.
func (b *Reader) fill() {
	// Slide existing data to beginning.
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}

	// Read new data.
	n, e := b.rd.Read(b.buf[b.w:])
	b.w += n
	if e != nil {
		b.err = e
	}
}

func (b *Reader) readErr() os.Error {
	err := b.err
	b.err = nil
	return err
}

// Read reads data into p.
// It returns the number of bytes read into p.
// It calls Read at most once on the underlying Reader,
// hence n may be less than len(p).
// At EOF, the count will be zero and err will be os.EOF.
func (b *Reader) Read(p []byte) (n int, err os.Error) {
	n = len(p)
	if n == 0 {
		return 0, b.readErr()
	}
	//if b.w == b.r {
	if b.w < (b.r+8192) { // 8192 is an arbitrary fill threshold
		if b.err != nil {
			return 0, b.readErr()
		}
		/*if len(p) >= len(b.buf) {
			// Large read, empty buffer.
			// Read directly into p to avoid copy.
			n, b.err = b.rd.Read(p)
			if n > 0 {
				b.lastByte = int(p[n-1])
				b.lastRuneSize = -1
			}
			return n, b.readErr()
		}*/
		b.fill()
		if b.w == b.r {
			return 0, b.readErr()
		}
	}

	if n > b.w-b.r {
		n = b.w - b.r
	}
	copy(p[0:n], b.buf[b.r:])
	b.r += n
	b.lastByte = int(b.buf[b.r-1])
	b.lastRuneSize = -1

	/*for i:=0; i<(n/4); i+=4 {
		sampleLeft := int16(binary.LittleEndian.Uint16(p[i:i+2]))/4
		sampleRight :=  int16(binary.LittleEndian.Uint16(p[i+2:i+4]))/4
		//fmt.Println(p[i:i+4], sampleLeft, sampleRight)

		binary.LittleEndian.PutUint16(p[i:i+2], uint16(sampleLeft))
		binary.LittleEndian.PutUint16(p[i+2:i+4], uint16(sampleRight))
	}*/
	return n, nil
}

func (b *Reader) FillBuffer() (n int, err os.Error) {
	if b.w == b.r {
		if b.err != nil {
			return 0, b.readErr()
		}
		b.fill()
		if b.w == b.r {
			return 0, b.readErr()
		}
	}
	n = b.w - b.r
	return n, nil
}

// Buffered returns the number of bytes that can be read from the current buffer.
func (b *Reader) Buffered() int { return b.w - b.r }

func (b *Reader) Flush() (err os.Error) {
	b = new(Reader)
	b.w = 0
	b.r = 0
	return nil
}

func (b *Reader) Size() int { return len(b.buf) }
