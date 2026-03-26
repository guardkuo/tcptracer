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

var MaxNumOfTCPConn uint16 = 4096
var Conn []*TCPConn

func TCPConn_FindExistConn(conn []*TCPConn, handle uint16) (*TCPConn, error) {
	if handle >= MaxNumOfTCPConn {
		return nil, ErrRange
	}
	for i := 0; i < len(conn); i++ {
		if conn[i].Handle == handle {
			return conn[i], nil
		}
	}

	return nil, ErrNotFound
}

func TCPConn_GetConn(conn []*TCPConn, handle uint16) (*TCPConn, error) {
	var c *TCPConn

	c, err := TCPConn_FindExistConn(conn, handle)
	if err != nil {
		if err == ErrRange {
			return nil, ErrRange
		}
		c = new(TCPConn)
		c.Init(uint16(handle), 0)
		Conn = append(Conn, c)
		return c, nil
	}
	return c, ErrExists
}

func MoveToKeyword(s []byte, key byte) int {
	for i := 0; i < len(s); i++ {
		if key == s[i] {
			return i
		}
	}

	return -1
}

func StrToTime(str string) uint32 {
	var tm uint32 = 0
	if len(str) >= 8 {
		val, err := strconv.ParseUint(str[:8], 16, 32)
		if err == nil {
			tm = uint32(val)
		}
	}
	return tm
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
		// TX fast revoery start or end
		recovery = 1
		timeout = 0
		rx_drop = 0
	case strings.Contains(log, "IOC 100"), strings.Contains(log, "IOC 102"):
		// cwnd is updated or this connection is going to close ("IOC 102")
		recovery = 0
		timeout = 0
		rx_drop = 0
	case strings.Contains(log, "IOC 103"):
		// TX RTO start or end
		recovery = 1
		timeout = 1
		rx_drop = 0
	case strings.Contains(log, "IOC 104"):
		// RX drop recovery is started or ended
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
				if strings.Contains(parts[0], "Misc") {
					misc := strings.SplitN(parts[0], "(", 2)
					if len(misc) == 2 {
						key = "MISC"
						misc1 := strings.SplitN(misc[1], ")", 2)
						value = misc1[0]
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
		var c *TCPConn
		switch key {
		case "T":
			tm = StrToTime(value)
		case "MSS":
			subParts := strings.Split(value, "(")
			if len(subParts) == 2 {
				innerValue := strings.TrimSuffix(subParts[1], ")")
				dataParts := strings.Split(innerValue, ",")
				if len(dataParts) == 4 {
					var handleWithFlagsStr string
					var cwndStr string
					var handle uint16
					var add_wnd uint16
					mssStr := dataParts[2]
					if recovery == 1 {
						handleWithFlagsStr = dataParts[0]
						cwndStr = dataParts[1]
					} else {
						handleWithFlagsStr = dataParts[1]
						cwndStr = dataParts[0]
					}
					handleWithFlags, err := strconv.ParseUint(handleWithFlagsStr, 16, 32)
					if err != nil {
						continue
					}

					cwnd, err := strconv.ParseUint(cwndStr, 16, 32)
					if err != nil {
						continue
					}
					mssVal, err := strconv.ParseUint(mssStr, 16, 32)
					if err != nil {
						continue
					}
					if recovery == 1 {
						handle = uint16(handleWithFlags >> 16)
					} else {
						handle = uint16(handleWithFlags & 0xffff)
						add_wnd = uint16(cwnd >> 16)
						cwnd = cwnd & 0xffff
					}
					c, err = TCPConn_GetConn(Conn, handle)
					if err != nil {
						if err == ErrExists {
							if c.Mss == 0 {
								c.Mss = int(mssVal)
							}
						} else {
							continue
						}
					} else {
						c.Mss = int(mssVal)
					}
					var frame = new(TCPPacket)
					if recovery == 1 {
						if (handleWithFlags & 0xffff) == 1 {
							recovery = 2
						}
						frame.Init(tm, uint32(cwnd), 0, recovery, timeout, 0)
					} else {
						frame.Init(tm, uint32(cwnd), 0, 0, 0, 0)
						if add_wnd == 0 {
							frame.SetEst()
						}
					}
					c.AppendNewFrame(frame)
				}
			}
		case "MISC":
			// dataParts[2] = handle
			// rx_drop = 0
			// dataParts[0] = CurSeqNum
			// dataParts[1] = LastAckedSeqNum or HighRx
			// dataParts[3] = state, 0: start, 1: end
			// rx_drop = 1
			// dataParts[0] = Expected SeqNum/the latest seq num
			// dataParts[1] = Received SeqNum/starting miss seq num
			// dataParts[3] = state, 0: end, other:
			dataParts := strings.Split(value, ",")
			if len(dataParts) == 4 {
				handleStr := dataParts[2]
				handle, err := strconv.ParseUint(handleStr, 16, 32)
				c, err = TCPConn_GetConn(Conn, uint16(handle))
				if err == ErrRange {
					continue
				}
				var frame = new(TCPPacket)
				recoveryStr := dataParts[3]
				isRecoveryEnd, err := strconv.ParseUint(recoveryStr, 16, 32)
				if err != nil {
					isRecoveryEnd = 0
				}
				SeqNumStr := dataParts[0]

				if rx_drop == 0 {
					if isRecoveryEnd == 1 {
						SeqNumStr = dataParts[1]
						recovery = 2
					}
					SeqNum, err := strconv.ParseUint(SeqNumStr, 16, 32)
					if err != nil {
						SeqNum = 0
					}
					frame.Init(tm, uint32(SeqNum), 0, recovery, timeout, 0)
				} else {

					if isRecoveryEnd == 0 {
						recovery = 0
					} else {
						recovery = 1
					}
					SeqNum, err := strconv.ParseUint(SeqNumStr, 16, 32)
					if err != nil {
						SeqNum = 0
					}
					RecvSeqNumStr := dataParts[1]
					RecvSeqNum, err := strconv.ParseUint(RecvSeqNumStr, 16, 32)
					if err != nil {
						RecvSeqNum = 0
					}
					frame.Init(tm, uint32(RecvSeqNum), 1, recovery, 0, uint32(SeqNum))

				}
				c.AppendNewFrame(frame)

			}
		default:
			continue
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
