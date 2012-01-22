/*
 *  slimgo - Squeezebox Client
 *  Copyright (C) 2012 Bart Lauret
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */
package main

import (
	"log"
	"flag"
	"net"
	"time"
	"os"
	"./alsa-go/_obj/alsa"
	//"bufio"
)

var startTime = time.Nanoseconds() / 1e6
var useDisco = flag.Bool("d", true, "use discovery to find SB server")
var lmsAddr = flag.String("S", "", "IP-address of the Logitech Media Server")
var outputDevice = flag.String("D", "hw:0,0", "ALSA output device, use aplay -L to see the options")

// slimaudio struct
type audio struct {
	Handle *alsa.Handle
	State string
	Pcmsamplesize uint8
	Pcmsamplerate uint8
	Pcmchannels uint8
	Pcmendian uint8
	StartSeconds int64
	StartNanos int64
	ElapsedSeconds uint32
	ElapsedMillis uint32
}
var slimaudio audio

// slimproto struct
type proto struct {
	Conn *net.TCPConn
	Addr net.IP
}
var slimproto proto

//slimbuffer struct
type buffer struct {
	Reader *Reader
	Init bool
}
var slimbuffer buffer

// channel which blocks until slimproto is ready
var slimprotoChannel = make(chan int)  // Allocate a channel.
var slimaudioChannel = make(chan int)  // Allocate a channel.

func main() {
	// First parse the command line options
	flag.Parse()
    //for i := 0; i < flag.NArg(); i++ {
	//	log.Println(flag.Arg(i))
    //}

	var addr net.IP
	var port int
	// Use discovery for SB server
	if *useDisco == true {
		addr, port = slimprotoDisco()
	} else if *lmsAddr != "" {
		addr, port = net.ParseIP(*lmsAddr), 3483
	} else {
		log.Fatalln("Please use server discovery or supply the IP-address of the server, see --help for more information.")
	}
	slimproto.Addr = addr

	slimbuffer.Init = false

	// Open a ALSA handle
	slimaudio.Handle = slimaudioOpen(*outputDevice)
	defer slimaudioClose(slimaudio.Handle)

	// Connect to SB server
	go slimproto_main(addr, port)

	<-slimprotoChannel   // Wait for slimproto to finish; discard sent value.
}

func jiffies() uint32 {
	return uint32((time.Nanoseconds() / 1e6) - startTime)
}

func slimproto_main(addr net.IP, port int) {

	for {
		var reconnect = false

		log.Printf("Using %v:%v for slimproto\n", addr, port)

		slimprotoConnect(addr, port)
		defer slimprotoClose()

		err := slimprotoHello()
		if err != nil {
			log.Println("Handshake failed, trying again")
			time.Sleep(1e9)
			continue
		} else {
			log.Println("HELO send succesfully")
		}

		for {
			err := slimprotoRecv()
			if err != nil {
				switch err {
				case os.EAGAIN:
					log.Println("Slimproto timeout")
					time.Sleep(1e9)
					reconnect = true
				default:
					log.Println("Slimproto error", err)
					time.Sleep(1e9)
					reconnect = true
				}
			}
			if reconnect {
				break
			}
		}
		slimaudio.Handle.Flush()
		_ = slimbuffer.Reader.Flush()
	}
	slimprotoChannel <- 1  // Send a signal; value does not matter. 

}
