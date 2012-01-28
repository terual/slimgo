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
	"os"
	"log"
	"./alsa-go/_obj/alsa"
)

// Open ALSA
func slimaudioOpen(device string) (handle *alsa.Handle) {
	handle = alsa.New()

	err := handle.Open(device, alsa.StreamTypePlayback, alsa.ModeBlock)
	if err != nil {
		log.Fatalf("Open failed. %s", err)
	} else {
		if *debug {
			log.Printf("ALSA device %s opened", device)
		}
	}

	return
}

// Close ALSA
func slimaudioClose(handle *alsa.Handle) {
	handle.Close()
	if *debug {
		log.Println("ALSA closed")
	}
	return
}

// Apply the hw parameters
func slimaudioSetParams(handle *alsa.Handle, sampleFormat alsa.SampleFormat, sampleRate int, channels int) (err os.Error) {
	handle.SampleFormat = sampleFormat
	handle.SampleRate = sampleRate
	handle.Channels = channels
	handle.Periods = 2
	handle.Buffersize = 8192

	err = handle.ApplyHwParams()
	return
}

// Writes data to ALSA
func slimaudioWrite(handle *alsa.Handle, nStart int, nEnd int, data []byte, format alsa.SampleFormat, rate int, channels int) (n int, alsaErr os.Error, writeErr os.Error) {

	if slimaudio.NewTrack == true {
		if handle.SampleFormat != format || handle.SampleRate != rate || handle.Channels != channels {

			_ = handle.Drop()
			alsaErr = slimaudioSetParams(handle, format, rate, channels) // This also drains the alsa buffer
			_ = slimprotoSend(slimproto.Conn, 0, "STMs")                 // Track Started
			slimaudio.NewTrack = false

			if alsaErr != nil {
				return 0, alsaErr, nil
			} else {
				if *debug {
					log.Println("ALSA set to", format, rate, channels)
				}
			}

		} else {
			_ = slimprotoSend(slimproto.Conn, 0, "STMs") // Track Started
			slimaudio.NewTrack = false
		}
	}

	if nEnd > nStart {
		n, writeErr = handle.Write(data[nStart:nEnd])

		if writeErr != nil {
			log.Printf("Write failed. %s\n", writeErr)

			alsaErr = slimaudioSetParams(handle, format, rate, channels)
			n, writeErr = handle.Write(data[nStart:nEnd])
			if writeErr != nil {
				log.Printf("Write failed AGAIN. %s\n", writeErr)
			}
		}

		if n > 0 {
			slimaudio.FramesWritten += (n / (handle.SampleSize() * handle.Channels))
		}

	} else {
		return 0, nil, nil
	}

	return n, nil, writeErr
}

// Convert slimproto format to ALSA format parameters
func slimaudioProto2Param(pcmsamplesize uint8, pcmsamplerate uint8, pcmchannels uint8, pcmendian uint8) (format alsa.SampleFormat, rate int, channels int) {

	switch pcmchannels {
	case 49:
		channels = 1
	case 50:
		channels = 2
	}

	switch pcmsamplerate {
	case 48: //0
		rate = 11025
	case 49: //1
		rate = 22050
	case 50: //2
		rate = 32000
	case 51: //3
		rate = 44100
	case 52: //4
		rate = 48000
	case 53: //5
		rate = 8000
	case 54: //6
		rate = 12000
	case 55: //7
		rate = 16000
	case 56: //8
		rate = 24000
	case 57: //9
		rate = 96000
	case 58: //:
		rate = 88200
	case 59: //;
		rate = 192000
	case 60: //<
		rate = 176400
	}

	switch pcmendian {
	case 48:
		// 0: big-endian
		switch pcmsamplesize {
		case 48: //0: 8-bits
			format = alsa.SampleFormatS8
		case 49: //1: 16-bits
			format = alsa.SampleFormatS16BE
		case 50: //2: 24-bits
			format = alsa.SampleFormatS24_3BE
		case 51: //3: 32-bits
			format = alsa.SampleFormatS32BE
		}
	case 49:
		// 1: little-endian
		switch pcmsamplesize {
		case 48: //0: 8-bits
			format = alsa.SampleFormatS8
		case 49: //1: 16-bits
			format = alsa.SampleFormatS16LE
		case 50: //2: 24-bits
			format = alsa.SampleFormatS24_3LE
		case 51: //3: 32-bits
			format = alsa.SampleFormatS32LE
		}
	}

	return
}
