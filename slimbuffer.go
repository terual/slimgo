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
	"strings"
	"./alsa-go/_obj/alsa"
)

func slimbufferOpen(httpHeader []byte, addr string, port string, format alsa.SampleFormat, rate int, channels int) (err os.Error) {

	hdrSlice := strings.Fields(string(httpHeader[:]))
	req, _ := http.NewRequest(hdrSlice[0], "http://"+addr+":"+port+hdrSlice[1], nil)

	//var req *http.Request
	//u, err := url.Parse(hdrSlice[1])
	/*req := &http.Request{Method: hdrSlice[0], 
	RawURL: "http://127.0.0.1:9000" + hdrSlice[1], 
	Proto: hdrSlice[2], 
	Body: nil}*/

	r, err := http.DefaultClient.Do(req)
	checkError(err)

	// Create buffer with size 1MB
	buf, err := slimbuffer.Reader.NewReaderSize(r.Body, 1048576)

	if r.StatusCode == 200 { // 200 OK

		_ = slimprotoSend(slimproto.Conn, 0, "STMe") // Stream connection Established

		// This tracks the streamtime
		slimaudio.FramesWritten = 0
		slimaudio.NewTrack = true

		// TODO inBufLen
		inBufLen := 2 * 3 * 4 * channels * 1024
		inBuf := make([]byte, inBufLen)

		_ = slimprotoSend(slimproto.Conn, 0, "STMl") //	Buffer threshold reached 

		n, inErr := buf.Read(inBuf)
		slimbuffer.Init = true

		for inErr == nil {

			if slimaudio.State == "STOPPED" {
				if *debug {
					log.Println("Stopping goroutine slimbufferOpen")
				}
				return
			} else if slimaudio.State == "PAUSED" {
				<-slimaudioChannel
			}

			
			// Send data to ALSA interface
			_, alsaErr, writeErr := slimaudioWrite(slimaudio.Handle, n, inBuf, format, rate, channels)

			// An alsaErr is raised if for instance S24_3LE is not supported by hw:0,0
			if alsaErr != nil {
				log.Printf("Format not supported, if using hw as output device, try plughw: %v", alsaErr)
				_ = slimprotoSend(slimproto.Conn, 0, "STMn")
				slimaudio.State = "STOPPED"
				return
			}
			//TODO:
			// Reset ALSA
			if writeErr != nil {
				slimaudio.Handle.Close()
				slimaudio.Handle = slimaudioOpen(*outputDevice)
				//_ = slimprotoSend(slimproto.Conn, 0, "STMn")
				//slimaudio.State = "STOPPED"
				return
			}

			n, inErr = buf.Read(inBuf)
		}

		if inErr == os.EOF {
			// Close connection on EOF
			r.Body.Close()
			//err = slimprotoSend(slimproto.Conn, 0, "STMu")

			// STMd triggers the switch in the server to the next track
			err = slimprotoSend(slimproto.Conn, 0, "STMd")
			slimaudio.State = "STOPPED"
		}

	} else {
		r.Body.Close()
	}
	return

}
