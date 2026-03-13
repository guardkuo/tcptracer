package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var Conn []*TCPConn

func TCPConn_FindExistConn(conn []*TCPConn, handle uint16) (*TCPConn, error) {
	if handle >= 4096 {
		return nil, ErrRange
	}
	for i := 0; i < len(conn); i++ {
		if conn[i].Handle == handle {
			return conn[i], nil
		}
	}

	return nil, ErrNotFound
}

func MoveToKeyword(s []byte, key byte) int {
	for i := 0; i < len(s); i++ {
		if key == s[i] {
			return i
		}
	}

	return -1
}

func ParseTCPLogV2(log string) {
	var fields = strings.Fields(log)
	var recovery int8
	var timeout int8 = 0
	var tm uint32 = 0
	var key string
	var value string
	var rx_drop int

	switch {
	case strings.Contains(log, "IOC 101"):
		recovery = 1
		timeout = 0
		rx_drop = 0
	case strings.Contains(log, "IOC 100"), strings.Contains(log, "IOC 102"):
		recovery = 0
		timeout = 0
		rx_drop = 0
	case strings.Contains(log, "IOC 103"):
		recovery = 1
		timeout = 1
		rx_drop = 0
	case strings.Contains(log, "IOC 104"):
		rx_drop = 1
		recovery = 0
		timeout = 1
	default:
		return // Ignore lines that don't contain the specific IOC markers
	}

	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			if timeout == 1 {
				//fmt.Println("the number of parts: ", len(parts), parts[0])
				if strings.Contains(parts[0], "Misc") {
					misc := strings.SplitN(parts[0], "(", 2)
					if len(misc) == 2 {
						key = "MISC"
						misc1 := strings.SplitN(misc[1], ")", 2)
						value = misc1[0]
						//fmt.Println(value)
					} else {
						continue
					}
				} else {
					continue
				}
			} else {
				continue // Skip fields that don't have the key=value format
			}
		} else {
			key = parts[0]
			value = parts[1]
		}
		// fmt.Println("key = ", key, " value = ", value)
		var c *TCPConn
		switch key {
		case "T":
			if len(value) >= 8 {
				val, err := strconv.ParseUint(value[:8], 16, 32)
				if err == nil {
					tm = uint32(val)
				}
			}
		case "MSS":
			subParts := strings.Split(value, "(")
			if len(subParts) == 2 {
				innerValue := strings.TrimSuffix(subParts[1], ")")
				dataParts := strings.Split(innerValue, ",")
				if len(dataParts) == 4 {
					if recovery == 1 {
						// dataParts[0] = handle + ?
						// dataParts[1] = cwnd
						// dataParts[2] = mss
						handleWithFlagsStr := dataParts[0]
						cwndStr := dataParts[1]
						mssStr := dataParts[2]
						handleWithFlags, err := strconv.ParseUint(handleWithFlagsStr, 16, 32)
						if err != nil {
							continue
						}
						handle := handleWithFlags >> 16
						isRecoveryEnd := (handleWithFlags & 0xffff) == 1
						cwnd, err := strconv.ParseUint(cwndStr, 16, 32)
						if err != nil {
							continue
						}
						mssVal, err := strconv.ParseUint(mssStr, 16, 32)
						if err != nil {
							continue
						}
						c, err = TCPConn_FindExistConn(Conn, uint16(handle))
						if err != nil {
							if err == ErrRange {
								return
							}
							c = new(TCPConn)
							c.Init(uint16(handle), int(mssVal))
							Conn = append(Conn, c)
						} else {
							if c.Mss == 0 {
								c.Mss = int(mssVal)
							}
						}
						var frame = new(TCPPacket)
						if isRecoveryEnd {
							recovery = 2
						} else {
							recovery = 1
						}
						frame.Init(tm, uint32(cwnd), 0, recovery, timeout, 0)
						c.AppendNewFrame(frame)
					} else {
						// dataParts[0] = cwnd + add_cwnd
						// dataParts[1] = handle
						// dataParts[2] = mss
						handleWithFlagsStr := dataParts[1]
						cwndStr := dataParts[0]
						mssStr := dataParts[2]
						handleWithFlags, err := strconv.ParseUint(handleWithFlagsStr, 16, 32)
						if err != nil {
							continue
						}
						handle := handleWithFlags & 0xffff
						cwnd, err := strconv.ParseUint(cwndStr, 16, 32)
						if err != nil {
							continue
						}
						add_wnd := cwnd >> 16
						mssVal, err := strconv.ParseUint(mssStr, 16, 32)
						if err != nil {
							continue
						}
						c, err = TCPConn_FindExistConn(Conn, uint16(handle))
						if err != nil {
							if err == ErrRange {
								return
							}
							c = new(TCPConn)
							c.Init(uint16(handle), int(mssVal))
							Conn = append(Conn, c)
						} else {
							if c.Mss == 0 {
								c.Mss = int(mssVal)
							}
						}
						var frame = new(TCPPacket)
						frame.Init(tm, uint32(cwnd&0xffff), 0, 0, 0, 0)
						if add_wnd == 0 {
							frame.SetEst()
						}
						c.AppendNewFrame(frame)
					}
				}
			}
		case "MISC":
			dataParts := strings.Split(value, ",")
			if len(dataParts) == 4 {
				if rx_drop == 0 {
					// dataParts[0] = CurSeqNum
					// dataParts[1] = LastAckedSeqNum or HighRx
					// dataParts[2] = handle
					// dataParts[3] = state, 0: start, 1: end
					handleStr := dataParts[2]
					handle, err := strconv.ParseUint(handleStr, 16, 32)
					c, err = TCPConn_FindExistConn(Conn, uint16(handle))
					if err != nil {
						if err == ErrRange {
							return
						}
						c = new(TCPConn)
						c.Init(uint16(handle), 0)
						Conn = append(Conn, c)
					}
					var frame = new(TCPPacket)
					recoveryStr := dataParts[3]
					isRecoveryEnd, err := strconv.ParseUint(recoveryStr, 16, 32)
					SeqNumStr := dataParts[0]
					if isRecoveryEnd == 1 {
						SeqNumStr = dataParts[1]
						recovery = 2
					}
					SeqNum, err := strconv.ParseUint(SeqNumStr, 16, 32)
					frame.Init(tm, uint32(SeqNum), 0, recovery, timeout, 0)
					c.AppendNewFrame(frame)
				} else {
					// dataParts[0] = Expected SeqNum/the latest seq num
					// dataParts[1] = Received SeqNum/starting miss seq num
					// dataParts[2] = handle
					// dataParts[3] = state, 0: end, other:
					handleStr := dataParts[2]
					handle, err := strconv.ParseUint(handleStr, 16, 32)
					c, err = TCPConn_FindExistConn(Conn, uint16(handle))
					if err != nil {
						if err == ErrRange {
							return
						}
						c = new(TCPConn)
						c.Init(uint16(handle), 0)
						Conn = append(Conn, c)
					}
					var frame = new(TCPPacket)
					recoveryStr := dataParts[3]
					isRecoveryEnd, err := strconv.ParseUint(recoveryStr, 16, 32)
					SeqNumStr := dataParts[0]
					if isRecoveryEnd == 0 {
						recovery = 0
					} else {
						recovery = 1
					}
					SeqNum, err := strconv.ParseUint(SeqNumStr, 16, 32)
					RecvSeqNumStr := dataParts[1]
					RecvSeqNum, err := strconv.ParseUint(RecvSeqNumStr, 16, 32)
					frame.Init(tm, uint32(RecvSeqNum), 1, recovery, 0, uint32(SeqNum))
					//println("%d %x %x", recovery, RecvSeqNum , SeqNum)
					c.AppendNewFrame(frame)
				}
			}
		}
	}
}

func main() {
	var output io.Writer
	fileName := flag.String("file", "tcp.log", "file to parse")
	OutfileName := flag.String("o", "", "file to write")
	flag.Parse()

	fmt.Println("file name=", *fileName)
	fmt.Println("output file name=", *OutfileName)

	file, err := os.Open(*fileName)
	if err != nil {
		fmt.Println("Error opening input file:", err)
		os.Exit(1)
	}
	defer file.Close()
	if *OutfileName != "" {
		outfile, err := os.Create(*OutfileName)
		if err != nil {
			fmt.Println("Error opening output file:", err)
			os.Exit(1)
		}
		defer outfile.Close() // Ensure the file is closed after the function exits.
		output = outfile
	} else {
		output = os.Stdout
	}
	writer := bufio.NewWriter(output)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		ParseTCPLogV2(line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	for _, conn := range Conn {
		conn.Dump(writer)
	}
	writer.Flush()
}
