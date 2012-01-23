/*
 *  (c) 2012 Bart Lauret
 *
 *  This file is part of slimgo.
 *
 *  slimgo is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  slimgo is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with slimgo.  If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"log"
	"http"
	"os"
	"time"
	//"bufio"
	"io"
	"strconv"
	"strings"
	//"url"
	"./alsa-go/_obj/alsa"
	//"log"
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
	b, ok := rd.(*Reader)
	if ok && len(b.buf) >= size {
		//return b, nil
		//b = new(Reader)
		//b.buf = make([]byte, size)
		b.rd = rd
		b.lastByte = -1
		b.lastRuneSize = -1
	} else {
		b = new(Reader)
		b.buf = make([]byte, size)
		b.rd = rd
		b.lastByte = -1
		b.lastRuneSize = -1
	}
	return b, nil
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
	if b.w == b.r {
		if b.err != nil {
			return 0, b.readErr()
		}
		if len(p) >= len(b.buf) {
			// Large read, empty buffer.
			// Read directly into p to avoid copy.
			n, b.err = b.rd.Read(p)
			if n > 0 {
				b.lastByte = int(p[n-1])
				b.lastRuneSize = -1
			}
			return n, b.readErr()
		}
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
	return n, nil
}

// Buffered returns the number of bytes that can be read from the current buffer.
func (b *Reader) Buffered() int { return b.w - b.r }

func (b *Reader) Flush() (err os.Error) {
	b = new(Reader)
	return nil
}

func (b *Reader) Size() int { return len(b.buf) }

func slimbufferOpen(httpHeader []byte, addr string, port string, format alsa.SampleFormat, rate int, channels int) (err os.Error) {

	//var req *http.Request
	hdrSlice := strings.Fields(string(httpHeader[:]))
	req, _ := http.NewRequest(hdrSlice[0], "http://" + addr + ":" + port + hdrSlice[1], nil)

	//u, err := url.Parse(hdrSlice[1])
	/*req := &http.Request{Method: hdrSlice[0], 
		RawURL: "http://127.0.0.1:9000" + hdrSlice[1], 
		Proto: hdrSlice[2], 
		Body: nil}*/

	r, err := http.DefaultClient.Do(req)
	checkError(err)

	buf, err := slimbuffer.Reader.NewReaderSize(r.Body, 1048576)

	if r.StatusCode == 200 { // 200 OK

		_ = slimprotoSend(slimproto.Conn, 0, "STMe") // Stream connection Established

		slimaudio.StartSeconds = time.Seconds()
		slimaudio.StartNanos = time.Nanoseconds()

		inBufLen := 2 * 3 * 4 * channels * 1024
		inBuf := make([]byte, inBufLen)

		_ = slimprotoSend(slimproto.Conn, 0, "STMl") //	Buffer threshold reached 
		_ = slimprotoSend(slimproto.Conn, 0, "STMs") // Track Started 

		n, inErr := buf.Read(inBuf)
		slimbuffer.Init = true

		for inErr == nil {
			//v, ok := <-slimaudioChannel // peek in channel
			//fmt.Printf("slimaudioChannel; v: %v, ok: %v", v, ok)

			if slimaudio.State == "STOPPED" {
				if *debug { log.Println("Stopping goroutine slimbufferOpen") }
				return
			} else if slimaudio.State == "PAUSED" {
				<-slimaudioChannel
			}

			_, alsaErr, writeErr := slimaudioWrite(slimaudio.Handle, n, inBuf, format, rate, channels)
			if alsaErr != nil {
				log.Printf("Format not supported, if using hw as output device, try plughw: %v", alsaErr)
				_ = slimprotoSend(slimproto.Conn, 0, "STMn")
				slimaudio.State = "STOPPED"
				return
			}
			if writeErr != nil {
				slimaudio.Handle.Close()
				slimaudio.Handle = slimaudioOpen(*outputDevice)
				_ = slimprotoSend(slimproto.Conn, 0, "STMn")
				slimaudio.State = "STOPPED"
				return
			}
			n, inErr = buf.Read(inBuf)
		}

		if inErr == os.EOF {
			r.Body.Close()
			//err = slimprotoSend(slimproto.Conn, 0, "STMu")
			err = slimprotoSend(slimproto.Conn, 0, "STMd")
			slimaudio.State = "STOPPED"
		}

	} else {
		r.Body.Close()
	}
	return

}

