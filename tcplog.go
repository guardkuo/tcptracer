package main

import (
	"errors"
	"fmt"
	"io"
)

type TCPPacket struct {
	Time     uint32
	Cwnd     uint32
	SeqNum   uint32
	Recovery int8
	Timeout  int8
	RX_drop  int8
	Est      int8
}

type TCPConn struct {
	Handle uint16
	Mss    int
	Frame  []*TCPPacket
}

var ErrRange = errors.New("value out of range")
var ErrNotFound = errors.New("not found")
var ErrExists = errors.New("Exists")

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
	f.Est = 0
}

func (f *TCPPacket) SetEst() {
	f.Est = 1
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

func (conn *TCPConn) DumpTx_InFastRecovery(w io.Writer, now int, prev int) {
	if conn.Frame[now].Cwnd > conn.Frame[prev].Cwnd {
		fmt.Fprintf(w, "[TX]%08d     %6d   %08d +%04d\n", (conn.Frame[now].Time-conn.Frame[0].Time)>>2, (conn.Frame[now].Time-conn.Frame[prev].Time)>>2, conn.Frame[now].Cwnd,
			int(conn.Frame[now].Cwnd-conn.Frame[prev].Cwnd))
	} else if conn.Frame[now].Cwnd < conn.Frame[prev].Cwnd {
		fmt.Fprintf(w, "[TX]%08d     %6d   %08d -%04d\n", (conn.Frame[now].Time-conn.Frame[0].Time)>>2, (conn.Frame[now].Time-conn.Frame[prev].Time)>>2, conn.Frame[now].Cwnd,
			int(conn.Frame[prev].Cwnd-conn.Frame[now].Cwnd))
	} else {
		fmt.Fprintf(w, "[TX]%08d     %6d   %08d\n", (conn.Frame[now].Time-conn.Frame[0].Time)>>2, (conn.Frame[now].Time-conn.Frame[prev].Time)>>2, conn.Frame[now].Cwnd)
	}
}

func (conn *TCPConn) DumpTx_InRTO(w io.Writer, now int, prev int) uint32 {
	var duration uint32
	if prev == -1 {
		fmt.Fprintf(w, "[TX]00000000              %08x\n", conn.Frame[now].Cwnd)
		return 0
	} else {
		duration = (conn.Frame[now].Time - conn.Frame[prev].Time) >> 2
		fmt.Fprintf(w, "[TX]%08d     %6d   %08x\n", (conn.Frame[now].Time-conn.Frame[0].Time)>>2, duration, conn.Frame[now].Cwnd)
		return duration
	}
}

func (conn *TCPConn) DumpRx_InRecovery(w io.Writer, now int, prev int) {
	if conn.Frame[now].Recovery == 1 {
		fmt.Fprintf(w, "[RX]%08d %08X        %08X   %05X   Start\n", (conn.Frame[now].Time-conn.Frame[prev].Time)>>2, conn.Frame[now].SeqNum, conn.Frame[now].Cwnd, conn.Frame[now].Cwnd-conn.Frame[now].SeqNum)
	} else {
		fmt.Fprintf(w, "[RX]%08d %08X        %08X   %05X   End\n", (conn.Frame[now].Time-conn.Frame[prev].Time)>>2, conn.Frame[now].Cwnd, conn.Frame[now].SeqNum, conn.Frame[now].SeqNum-conn.Frame[now].Cwnd)
	}
}

func (conn *TCPConn) Dump(w io.Writer) {
	var txTimeoutOccurred int8
	var timeoutCwnd uint32
	var duration uint32
	var prev_tx_index int
	var prev_rx_index int
	var numOfFrame int

	numOfFrame = len(conn.Frame)
	if numOfFrame == 1 {
		return
	}
	// .Time >> 2: us
	// .Mss >> 10: KB
	txTimeoutOccurred = 0
	prev_tx_index = -1
	prev_rx_index = 0
	timeoutCwnd = uint32(conn.Mss >> 10)

	fmt.Fprintf(w, "Handle=%x, MSS=%d byte\n", conn.Handle, conn.Mss)
	duration = 0

	for i := 0; i < numOfFrame; i++ {
		if conn.Frame[i].Est == 1 {
			fmt.Fprintf(w, "%08d     EST\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2)
		} else {
			if conn.Frame[i].RX_drop == 0 {
				if conn.Frame[i].Recovery == 1 {
					if conn.Frame[i].Timeout == 1 {
						fmt.Fprintf(w, "---- RTO Start [%d us]----\n", duration)
					} else {
						fmt.Fprintf(w, "---- Fast Recovery Start [%d us]----\n", duration)
					}
					duration = 0
				}
				if conn.Frame[i].Timeout == 1 {
					duration += conn.DumpTx_InRTO(w, i, prev_tx_index)
				} else {
					if prev_tx_index == -1 {
						fmt.Fprintf(w, "[TX]00000000              %08d\n", conn.Frame[i].Cwnd)
					} else {
						duration += (conn.Frame[i].Time - conn.Frame[prev_tx_index].Time) >> 2
						if txTimeoutOccurred == 0 && conn.Frame[i].Cwnd == timeoutCwnd {
							// just enter the RTO, because Cwnd is set to 1 frame
							txTimeoutOccurred = 1
							fmt.Fprintf(w, "[TX]%08d     %6d   %08d ++++\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[prev_tx_index].Time)>>2, conn.Frame[i].Cwnd)
						} else {
							if conn.Frame[prev_tx_index].Timeout == 1 {
								// just leave the RTO
								fmt.Fprintf(w, "[TX]%08d     %6d   %08d\n", (conn.Frame[i].Time-conn.Frame[0].Time)>>2, (conn.Frame[i].Time-conn.Frame[prev_tx_index].Time)>>2, conn.Frame[i].Cwnd)
							} else {
								conn.DumpTx_InFastRecovery(w, i, prev_tx_index)
							}
						}
					}
				}
				prev_tx_index = i
			} else {
				conn.DumpRx_InRecovery(w, i, prev_rx_index)
				prev_rx_index = i
			}
		}
		if conn.Frame[i].Recovery == 1 {
			txTimeoutOccurred = 0
		}
		if conn.Frame[i].Recovery == 2 {
			txTimeoutOccurred = 0
			fmt.Fprintf(w, "---- End Recovery [%d us]----\n", duration)
			duration = 0
		}
	}
	fmt.Fprint(w, "***************************************\n")
	fmt.Fprint(w, "\n")

}
