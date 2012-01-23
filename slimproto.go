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
	//"fmt"
	"net"
	"os"
	"log"
    "encoding/binary"
	"time"
	"strconv"
)

func slimprotoDisco() (addr net.IP, port int) {

	type discoSend struct {
		JSON [11]byte
	}

	type discoResponse struct {
		Type [4]byte
		Lenght uint8
	}

	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{
		    IP:   net.IPv4bcast,
		    Port: 3483,
	}) 
	if err != nil {
		log.Fatalf("Fatal error: %s", err.String())
	}
	defer conn.Close()
	conn.SetTimeout(1e9)
	
	// send a packet
	msg := discoSend{}
	copy(msg.JSON[:], "eNAME\x00JSON\x00")
	//log.Println(msg)
	err = binary.Write(conn, binary.BigEndian, &msg)
	if err != nil {
		log.Fatalf("Fatal error: %s", err.String())
	}
	
    // receive the response
	conn, err = net.ListenUDP("udp4", &net.UDPAddr{
		    IP:   net.IPv4zero,
		    Port: 3483,
	}) 
	conn.SetTimeout(1e9)
	if err != nil {
		if e, ok := err.(*net.OpError); ok {
			if e.Error.(os.Errno) == 98 {
				// Presumably is it already in use by LMS on same machine
				log.Println("Discovery failed due to the discovery port already in use, so we presume that the server is running on this machine. Please supply command line parameters if otherwise.")
				return net.IP{127, 0, 0, 1}, 3483
			}
		}
	}

	remoteAddr := new(net.UDPAddr)
	for i := 0; i < 5; i++ {
		data := make([]byte, 1) 
		_, remoteAddr, err = conn.ReadFromUDP(data)
		if string(data[:]) == "E" {
			break
		} else if i == 4 {
			log.Fatalln("No servers found using discovery, supply IP and port with command line arguments.")
		}
	}

    var response discoResponse
    err = binary.Read(conn, binary.BigEndian, &response)
	checkError(err)

    if *debug { log.Printf("Response: %s %s:%s\n", remoteAddr, response.Type, response.Lenght) }

	// Parse response
	//addr = net.IPv4(response.IPaddr[0],response.IPaddr[1],response.IPaddr[2],response.IPaddr[3])
	//port = int(byte(response.Port[0])+byte(response.Port[1]))

	return
}

func slimprotoConnect(addr net.IP, port int) {

	sbsAddr := new(net.TCPAddr)
	sbsAddr.IP = addr
	sbsAddr.Port = port

	var err os.Error
	slimproto.Conn, err = net.DialTCP("tcp", nil, sbsAddr)
	checkError(err)
	slimproto.Conn.SetTimeout(5e9)

	if *debug { log.Println("Connected to slimproto") }

	return

}


type header struct {
	Lenght uint16
	CommandHeader [4]byte
}

type strm struct {
	Command uint8
	Autostart uint8
	Formatbyte uint8
	Pcmsamplesize uint8
	Pcmsamplerate uint8
	Pcmchannels uint8
	Pcmendian uint8
	Threshold uint8
	Spdif_enable uint8
	Trans_period uint8
	Trans_type uint8
	Flags uint8
	Output_threshold uint8
	RESERVED uint8
	Replay_gain uint32
	Server_port uint16
	Server_ip [4]byte
}

type audg struct {
	Old_left uint32
	Old_right uint32
	Dvc uint8
	Preamp uint8
	New_left uint16
	New_left_deci uint16
	New_right uint16
	New_right_deci uint16
}

func slimprotoRecv() (err os.Error) {

    var headerResponse header
    err = binary.Read(slimproto.Conn, binary.BigEndian, &headerResponse)

	if err == nil {
		// convert [4]uint8 to string
		var cmdHdr = string(headerResponse.CommandHeader[:])
		switch cmdHdr {
			case "strm":
				// read into strm struct
				var streamResponse strm
				err = binary.Read(slimproto.Conn, binary.BigEndian, &streamResponse)

				if *debug { log.Printf("[Recv strm] Command: %s, Autostart: %s, Formatbyte: %s, Pcmsamplesize: %s, Pcmsamplerate: %s, Pcmchannels: %s, Pcmendian: %s\n", 
					string(streamResponse.Command),	string(streamResponse.Autostart), string(streamResponse.Formatbyte),
					string(streamResponse.Pcmsamplesize),	string(streamResponse.Pcmsamplerate), 
					string(streamResponse.Pcmchannels), string(streamResponse.Pcmendian) ) }

				switch string(streamResponse.Command) {
				case "t":
					_ = slimprotoSend(slimproto.Conn, streamResponse.Replay_gain, "STMt")
				case "s":
					slimaudio.State = "PLAY"
					_ = slimprotoSend(slimproto.Conn, 0, "STMc")
				case "p":
					slimaudio.Handle.Pause()
					slimaudio.State = "PAUSED"
					if streamResponse.Replay_gain == 0 {
						_ = slimprotoSend(slimproto.Conn, 0, "STMp")
					}
				case "u":
					if slimaudio.State == "PAUSED" {
						slimaudio.Handle.Unpause()
						slimaudioChannel <- 1
						slimaudio.State = "PLAYING"
					}
				case "q":
					slimaudio.Handle.Flush()
					_ = slimbuffer.Reader.Flush()
					slimaudio.State = "STOPPED"
					err = slimprotoSend(slimproto.Conn, 0, "STMf")
				case "f":
					//flush
					slimaudio.Handle.Flush()
					_ = slimbuffer.Reader.Flush()
				case "a":
					//skip-ahead
				default:
					if *debug { log.Println("Did not recognise strm message with cmd: %s", string(streamResponse.Command)) }
				}

				if *debug { log.Printf("slimaudio.State: %s\n", slimaudio.State) }


				// check if a http header is sent
				if headerResponse.Lenght > 28 {
					httpHeader := make([]byte, headerResponse.Lenght-28)
					_, _ = slimproto.Conn.Read(httpHeader[0:])

					if string(streamResponse.Formatbyte) == "p" {
						format, rate, channels := slimaudioProto2Param(streamResponse.Pcmsamplesize, 
									streamResponse.Pcmsamplerate, 
									streamResponse.Pcmchannels, 
									streamResponse.Pcmendian)
						port := strconv.Itoa(int(streamResponse.Server_port))
						go slimbufferOpen(httpHeader, slimproto.Addr.String(), port, format, rate, channels)
						_ = slimprotoSend(slimproto.Conn, 0, "STMh")
						slimaudio.State = "PLAYING"
						//slimaudioChannel <- 2
					} else {
						if *debug { log.Printf("Format not supported, Formatbyte: %s", string(streamResponse.Formatbyte)) }
						_ = slimprotoSend(slimproto.Conn, 0, "STMn")
					}
				}

			default:
				body := make([]byte, headerResponse.Lenght-4) 
				_, err = slimproto.Conn.Read(body[0:])
		}
	}

	return

}

type STAT struct {
	Operation	[4]byte
	Length		uint32
	EventCode 	[4]byte
	CRLF 		uint8
	MASInit 	uint8
	MASMode 	uint8
	BufferSize 	uint32
	BufferFullness uint32
	BytesReceived uint64
	WirelessStrength uint16
	Jiffies		uint32
	OutputBufferSize uint32
	OutputBufferFullness uint32
	ElapsedSeconds	uint32
	Voltage		uint16
	ElapsedMillis uint32
	Timestamp	uint32
	ErrorCode	uint16
}
/*
u32 	Event Code (a 4 byte string)
u8 		Number of consecutive CRLF recieved while parsing headers
u8 		MAS Initalized - 'm' or 'p'
u8 		MAS Mode - serdes mode?
u32 	buffer size - in bytes, of the player's (network/stream) buffer
u32 	fullness - data bytes in the player's (network/stream) buffer
u64 	Bytes Recieved
u16 	Wireless Signal Strength (0-100 - Larger values mean hardwired)
u32 	jiffies - a timestamp from the player (@1kHz)
u32 	output buffer size - the decoded audio data buffer size
u32 	output buffer fullness - bytes in the decoded audio data buffer
u32 	elapsed seconds - of the current stream
u16 	voltage
u32 	elapsed milliseconds - of the current stream
u32 	server timestamp - reflected from an strm-t command
u16 	error code - used with STMn */

func slimprotoSend(conn *net.TCPConn, timestamp uint32, eventcode string) (err os.Error) {

	var ElapsedSeconds uint32
	var ElapsedMillis uint32
	if slimaudio.StartSeconds != 0 {
		ElapsedSeconds = uint32(time.Seconds() - slimaudio.StartSeconds)
		ElapsedMillis = uint32((time.Nanoseconds() - slimaudio.StartNanos) / 1e6)
	}

	var BufferFullness int
	var BufferSize int
	if slimbuffer.Init == true {
		BufferFullness = 0 //slimbuffer.Reader.Buffered()
		BufferSize = 0 //slimbuffer.Reader.Size()
	} else {
		BufferFullness = 0
		BufferSize = 0
	}

	msg := STAT{Length: 53, 
		Timestamp: timestamp, 
		WirelessStrength: 65534, 
		Jiffies: jiffies(),
		OutputBufferSize: uint32(BufferSize),
		OutputBufferFullness: uint32(BufferFullness),
		ElapsedSeconds: ElapsedSeconds, 
		ElapsedMillis: ElapsedMillis}
	copy(msg.Operation[:], "STAT")
	copy(msg.EventCode[:], eventcode)

	err = binary.Write(conn, binary.BigEndian, &msg)
	if *debug { log.Println("Send", eventcode, msg) }

	return

}

func slimprotoClose() {
	err := slimproto.Conn.Close()
	checkError(err)
	if *debug { log.Println("Connection to slimproto closed") }
}

func slimprotoHello(macAddr [6]uint8) (err os.Error) {

	type HELO struct {
		Operation [4]byte
		Length uint32
		DeviceID uint8
		Revision uint8
		MAC [6]uint8
		UUID [16]uint8
		WLanChannelList [2]uint8
		Bytes_recv [8]uint8
		Language [2]uint8
		Capabilities [59]byte
	}

	// send a packet
	msg := HELO{Length: 36+59, DeviceID: 12, Revision: 255, MAC: macAddr}
	copy(msg.Operation[:], "HELO")
	//copy(msg.MAC[:], "\x12\x34\x56\x78\x90\xab")
	copy(msg.Capabilities[:], "model=squeezeplay,modelName=SlimGo,pcm,MaxSampleRate=192000")
	err = binary.Write(slimproto.Conn, binary.BigEndian, &msg)

	return
}

func slimprotoBye() (err os.Error) {

	type BYE struct {
		Operation [4]byte
		Length uint32
		Upgrade uint8
	}

	// send a packet
	msg := BYE{Length: 1, Upgrade: 0}
	copy(msg.Operation[:], "BYE!")
	err = binary.Write(slimproto.Conn, binary.BigEndian, &msg)
	if *debug { log.Printf("Sent BYE! msg: %v", msg) }

	return
}

func checkError(err os.Error) {
	if err != nil {
		log.Println("reader %v\n", err)
	}

	/*
	if err != nil {
		// print error string e.g.
		// "read tcp example.com:80: resource temporarily unavailable"
		fmt.Printf("reader %v\n", err)

		// print type of the error, e.g. "*net.OpError"
		fmt.Printf("%T\n", err)

		if err == os.EINVAL {
		  // socket is not valid or already closed
		  fmt.Println("EINVAL");
		}
		if err == os.EOF {
		  // remote peer closed socket
		  fmt.Println("EOF");
		}

		// matching rest of the codes needs typecasting, errno is
		// wrapped on OpError
		if e, ok := err.(*net.OpError); ok {
		   // print wrapped error string e.g.
		   // "os.Errno : resource temporarily unavailable"
		   fmt.Printf("%T : %v\n", e.Error, e.Error)
		   if e.Timeout() {
		     // is this timeout error?
		     fmt.Println("TIMEOUT")
		   }
		   if e.Temporary() {
		     // is this temporary error?  True on timeout,
		     // socket interrupts or when buffer is full
		     fmt.Println("TEMPORARY")
		   }

		  // specific granular error codes in case we're interested
		 switch e.Error {
		    case os.EAGAIN:
		       // timeout
		       fmt.Println("EAGAIN")
		   case os.EPIPE:
		      // broken pipe (e.g. on connection reset)
		      fmt.Println("EPIPE")
		   default:
		      // just write raw errno code, can be platform specific
		      // (see syscall for definitions)
		      fmt.Printf("%d\n", int64(e.Error.(os.Errno)))
		 }
		}
	}
	*/
}

