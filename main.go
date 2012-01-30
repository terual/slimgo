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
	"os/signal"
	"syscall"
	"strings"
	"strconv"
)

// startTime is used by jiffies()
var startTime = time.Nanoseconds() / 1e6

// Setup flags for command line options
var useDisco = flag.Bool("F", true, "use discovery to find SB server")
var lmsAddr = flag.String("S", "", "IP-address of the Logitech Media Server")
var lmsPortr = flag.Int("P", 3483, "Port of the Logitech Media Server")
var outputDevice = flag.String("o", "default", "ALSA output device, use aplay -L to see the options")
var debug = flag.Bool("d", true, "view debug messages")
var macAddr = flag.String("m", "00:00:00:00:00:02", "Sets the mac address for this instance. Use the colon-separated notation. The default is 00:00:00:00:00:02. Squeezebox Server uses this value to distinguish multiple instances, allowing per-player settings.")

// slimaudio struct
type audio struct {
	Handle            *alsa.Handle
	State             string
	Pcmsamplesize     uint8
	Pcmsamplerate     uint8
	Pcmchannels       uint8
	Pcmendian         uint8
	FramesWritten     int
	LastFramesWritten int
	NewTrack          bool
}

var slimaudio audio

// slimproto struct
type proto struct {
	Conn *net.TCPConn
	Addr net.IP
	Port int
}

var slimproto proto

//slimbuffer struct
type buffer struct {
	Reader *Reader
	Init   bool
}

var slimbuffer buffer

// channel which blocks until slimproto is ready
var slimprotoChannel = make(chan int) // Allocate a channel.
var slimaudioChannel = make(chan int) // Allocate a channel.

func main() {
	// First parse the command line options
	flag.Parse()

	mac, err := macConvert(*macAddr)
	if err != nil {
		log.Fatalf("Cannot parse MAC address: %v", *macAddr)
	}

	// Use discovery for SB server
	if *useDisco == true {
		slimproto.Addr, slimproto.Port = slimprotoDisco()
	} else if *lmsAddr != "" {
		slimproto.Addr, slimproto.Port = net.ParseIP(*lmsAddr), 3483
	} else {
		log.Fatalln("Please use server discovery or supply the IP-address of the server, see --help for more information.")
	}

	// TODO
	slimbuffer.Reader = new(Reader)
	slimbuffer.Reader.buf = make([]byte, 1048576)
	slimbuffer.Init = false

	// Open a ALSA handle
	if *outputDevice == "default" {
		log.Println("Using output device 'default', consider using 'hw:0,0' to avoid conversion in ALSA")
	}
	slimaudio.Handle = slimaudioOpen(*outputDevice)
	defer slimaudioClose(slimaudio.Handle)
	maxRate, _ := slimaudio.Handle.MaxSampleRate()
	log.Printf("Maximum sample rate of %s: %v Hz.", *outputDevice, maxRate)

	// This catches a SIGTERM et al. to be able to send a BYE! message
	go signalWatcher()

	// Connect to SB server
	go slimproto_main(slimproto.Addr, slimproto.Port, mac, maxRate)

	<-slimprotoChannel // Wait for slimproto to finish; discard sent value.
}

// jiffies returns a 1kHz counter since start of program
func jiffies() uint32 {
	return uint32((time.Nanoseconds() / 1e6) - startTime)
}

// signalWatcher waits for a signal and send a BYE! message on SIGTERM, SIGINT and SIGQUIT
func signalWatcher() {
	for {
		select {
		case sig := <-signal.Incoming:
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

// Convert a colon seperated mac-address to a uint8 array
func macConvert(macAddr string) (decMac [6]uint8, err os.Error) {
	f := func(i int) bool {
		if string(i) == ":" {
			return true
		}
		return false
	}
	mac := strings.FieldsFunc(macAddr, f)
	if len(mac) != 6 {
		return [6]uint8{0, 0, 0, 0, 0, 0}, os.NewError("Cannot parse mac-address")
	}
	for i, v := range mac {
		decMac64, err := strconv.Btoui64(v, 16)
		if err != nil {
			return [6]uint8{0, 0, 0, 0, 0, 0}, err
		}
		decMac[i] = uint8(decMac64)
	}
	return decMac, nil
}

func getMacAddr() (mac [6]uint8, err os.Error) {
	ifaces, _ := net.Interfaces()
	for i := range ifaces {

		iface, err := net.InterfaceByIndex(i + 1)
		if err != nil {
			log.Println(err, i)
			break
		}
		log.Println(iface.HardwareAddr)

	}
	return
}

// Main loop
func slimproto_main(addr net.IP, port int, mac [6]uint8, maxRate int) {

	for {
		var reconnect = false

		if *debug {
			log.Printf("Using %v:%v for slimproto\n", addr, port)
		}

		slimprotoConnect(addr, port)
		defer slimprotoClose()

		err := slimprotoHello(mac, maxRate)
		if err != nil {
			if *debug {
				log.Println("Handshake failed, trying again")
			}
			time.Sleep(1e9)
			continue
		} else {
			if *debug {
				log.Println("HELO send succesfully")
			}
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
		slimaudio.Handle.Drop()
		_ = slimbuffer.Reader.Flush()
	}
	slimprotoChannel <- 1 // Send a signal; value does not matter. 

}
