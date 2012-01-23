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
    "os/signal"
	"syscall"
)

var startTime = time.Nanoseconds() / 1e6
var useDisco = flag.Bool("F", true, "use discovery to find SB server")
var lmsAddr = flag.String("S", "", "IP-address of the Logitech Media Server")
var lmsPortr = flag.Int("P", 3483, "Port of the Logitech Media Server")
var outputDevice = flag.String("o", "hw:0,0", "ALSA output device, use aplay -L to see the options")
var debug = flag.Bool("d", true, "view debug messages")

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

	go signalWatcher()

	// Connect to SB server
	go slimproto_main(addr, port)

	<-slimprotoChannel   // Wait for slimproto to finish; discard sent value.
}

func jiffies() uint32 {
	return uint32((time.Nanoseconds() / 1e6) - startTime)
}

func signalWatcher() {
	for {
		select {
		    case sig := <- signal.Incoming:
		        switch s := sig.(type) {
		        case os.UnixSignal:
		                switch s {
		                case syscall.SIGCHLD:
		                	continue
		                case syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT: 
							_ = slimprotoBye()
							os.Exit(0)
						default:
							continue
		                }
		        default:
					continue
		        }
		}
	}
}


func slimproto_main(addr net.IP, port int) {

	for {
		var reconnect = false

		if *debug { log.Printf("Using %v:%v for slimproto\n", addr, port) }

		slimprotoConnect(addr, port)
		defer slimprotoClose()

		err := slimprotoHello()
		if err != nil {
			if *debug { log.Println("Handshake failed, trying again") }
			time.Sleep(1e9)
			continue
		} else {
			if *debug { log.Println("HELO send succesfully") }
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
