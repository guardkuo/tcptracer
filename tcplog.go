package main

import (
	"errors"
	"fmt"
)

type TCPPacket struct {
	Time     uint32
	Cwnd     uint32
	Recovery int8
}

type TCPConn struct {
	Handle uint16
	Mss    int
	Frame  []*TCPPacket
}

var ErrRange = errors.New("value out of range")
var ErrNotFound = errors.New("not found")

func (f *TCPPacket) Init(time uint32, cwnd uint32, recovery int8) {
	f.Time = time
	f.Cwnd = cwnd
	f.Recovery = recovery
}

func NewTCPPacket(time uint32, cwnd uint32, recovery int8) *TCPPacket {
	var f *TCPPacket = new(TCPPacket)
	f.Init(time, cwnd, recovery)
	return f
}

func (conn *TCPConn) Init(handle uint16, mss int) {
	conn.Handle = handle
	conn.Mss = mss
}

func NewTCPConn(handle uint16, mss int) *TCPConn {
	var c *TCPConn = new(TCPConn)
	c.Init(handle, mss)
	return c
}

func (conn *TCPConn) AppendNewFrame(f *TCPPacket) {
	conn.Frame = append(conn.Frame, f)
}

func (conn *TCPConn) AddFrame(f *TCPPacket) {
	conn.Frame = append(conn.Frame, f)
}

func (conn *TCPConn) Dump() {
	var txTimeoutOccurred int8
	var timeoutCwnd uint32

	if len(conn.Frame) == 1 {
		return
	}

	txTimeoutOccurred = 0
	timeoutCwnd = uint32(conn.Mss >> 10)

	fmt.Printf("Handle=%x, MSS=%d byte\n", conn.Handle, conn.Mss)
	for i := 0; i < len(conn.Frame); i++ {
		if i == 0 {
			fmt.Printf("                    %08d\n", conn.Frame[i].Cwnd)
		} else {
			if txTimeoutOccurred == 0 && conn.Frame[i].Cwnd == timeoutCwnd {
				txTimeoutOccurred = 1
				fmt.Printf("%08d     %6d   %08d ++++\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd)
			} else {
				if conn.Frame[i].Cwnd > conn.Frame[i-1].Cwnd {
					fmt.Printf("%08d     %6d   %08d +%04d\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd,
						int(conn.Frame[i].Cwnd-conn.Frame[i-1].Cwnd))
				} else if conn.Frame[i].Cwnd < conn.Frame[i-1].Cwnd {
					fmt.Printf("%08d     %6d   %08d -%04d\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd,
						int(conn.Frame[i-1].Cwnd-conn.Frame[i].Cwnd))
				} else {
					fmt.Printf("%08d     %6d   %08d\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd)
				}
			}
		}
		if conn.Frame[i].Recovery == 1 {
			txTimeoutOccurred = 0
			fmt.Println("---- Start Recovery ----")
		}
		if conn.Frame[i].Recovery == 2 {
			txTimeoutOccurred = 0
			fmt.Println("---- End Recovery ----")
		}
	}
}
