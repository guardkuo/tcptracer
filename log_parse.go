package main

import (
	"bufio"
	"flag"
	"fmt"
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
	//var val uint64
	//var err error
	var tm uint32 = 0

	switch {
	case strings.Contains(log, "IOC 101"):
		recovery = 1
	case strings.Contains(log, "IOC 100"), strings.Contains(log, "IOC 102"):
		recovery = 0
	default:
		return // Ignore lines that don't contain the specific IOC markers
	}

	for _, field := range fields {
		parts := strings.SplitN(field, "=", 2)
		if len(parts) != 2 {
			continue // Skip fields that don't have the key=value format
		}
		key := parts[0]
		value := parts[1]
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
						}
						var frame = new(TCPPacket)
						if isRecoveryEnd {
							recovery = 2
						} else {
							recovery = 1
						}
						frame.Init(tm, uint32(cwnd), recovery)
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
						}
						var frame = new(TCPPacket)
						frame.Init(tm, uint32(cwnd&0xffff), 0)
						c.AppendNewFrame(frame)
					}
				}
			}
		}
	}
}

func main() {
	fileName := flag.String("file", "tcp.log", "file to parse")
	flag.Parse()

	fmt.Println("file name=", *fileName)

	file, err := os.Open(*fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		ParseTCPLogV2(line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	for _, conn := range Conn {
		conn.Dump()
	}
}
