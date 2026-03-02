package main

import (
	"errors"
	"fmt"
)

type TCPPacket struct {
	Time     uint32
	Cwnd     uint32
	SeqNum   uint32
	Recovery int8
	Timeout  int8
	RX_drop  int8
}

type TCPConn struct {
	Handle uint16
	Mss    int
	Frame  []*TCPPacket
}

var ErrRange = errors.New("value out of range")
var ErrNotFound = errors.New("not found")

func (f *TCPPacket) Init(time uint32, cwnd uint32, rx_drop int8, recovery int8, timeout int8, RecvSeq uint32) {
	f.Time = time
	f.Cwnd = cwnd
	f.Recovery = recovery
	f.RX_drop = rx_drop
	if rx_drop == 0 {
		f.Timeout = timeout
		f.SeqNum = 0
	} else {
		f.SeqNum = RecvSeq
		f.Timeout = 0
	}
}

func NewTCPPacket(time uint32, cwnd uint32, recovery int8, timeout int8) *TCPPacket {
	var f *TCPPacket = new(TCPPacket)
	f.Init(time, cwnd, 0, recovery, timeout, 0)
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
	var duration uint32

	if len(conn.Frame) == 1 {
		return
	}

	txTimeoutOccurred = 0
	timeoutCwnd = uint32(conn.Mss >> 10)

	fmt.Printf("Handle=%x, MSS=%d byte\n", conn.Handle, conn.Mss)
	duration = 0

	for i := 0; i < len(conn.Frame); i++ {
		if conn.Frame[i].RX_drop == 0 {
			if conn.Frame[i].Recovery == 1 {
				if conn.Frame[i].Timeout == 1 {
					fmt.Printf("---- RTO Start [%d us]----\n", duration)
				} else {
					fmt.Printf("---- Fast Recovery Start [%d us]----\n", duration)
				}
				duration = 0
			}
			if conn.Frame[i].Timeout == 1 {
				if i == 0 {
					fmt.Printf("[TX]00000000              %08x\n", conn.Frame[i].Cwnd)
				} else {
					duration += (conn.Frame[i].Time - conn.Frame[i-1].Time) >> 2
					fmt.Printf("[TX]%08d     %6d   %08x\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd)
				}
			} else {
				if i == 0 {
					fmt.Printf("[TX]00000000              %08d\n", conn.Frame[i].Cwnd)
				} else {
					duration += (conn.Frame[i].Time - conn.Frame[i-1].Time) >> 2
					if txTimeoutOccurred == 0 && conn.Frame[i].Cwnd == timeoutCwnd {
						txTimeoutOccurred = 1
						fmt.Printf("[TX]%08d     %6d   %08d ++++\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd)
					} else {
						if conn.Frame[i-1].Timeout == 1 {
							fmt.Printf("[TX]%08d     %6d   %08d\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd)
						} else {
							if conn.Frame[i].Cwnd > conn.Frame[i-1].Cwnd {
								fmt.Printf("[TX]%08d     %6d   %08d +%04d\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd,
									int(conn.Frame[i].Cwnd-conn.Frame[i-1].Cwnd))
							} else if conn.Frame[i].Cwnd < conn.Frame[i-1].Cwnd {
								fmt.Printf("[TX]%08d     %6d   %08d -%04d\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd,
									int(conn.Frame[i-1].Cwnd-conn.Frame[i].Cwnd))
							} else {
								fmt.Printf("[TX]%08d     %6d   %08d\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[i-1].Time)>>2, conn.Frame[i].Cwnd)
							}
						}
					}
				}
			}
		} else {
			if conn.Frame[i].Recovery == 1 {
				fmt.Printf("[RX]%08d %08x        %08x   %05x   Start\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, conn.Frame[i].SeqNum, conn.Frame[i].Cwnd, conn.Frame[i].Cwnd-conn.Frame[i].SeqNum)
			} else {
				fmt.Printf("[RX]%08d %08x        %08x   %05x   End\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, conn.Frame[i].Cwnd, conn.Frame[i].SeqNum, conn.Frame[i].SeqNum-conn.Frame[i].Cwnd)
			}
		}
		if conn.Frame[i].Recovery == 1 {
			txTimeoutOccurred = 0
		}
		if conn.Frame[i].Recovery == 2 {
			txTimeoutOccurred = 0
			fmt.Printf("---- End Recovery [%d us]----\n", duration)
			duration = 0
		}
	}
	fmt.Println("***************************************")
	fmt.Println("")
}
