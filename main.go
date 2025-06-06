package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	. "github.com/layerssss/mpegts-parser/mpegts_parser"
)

func main() {
	packetsBuffer := make([]MPEGTSPacket, 1000)
	var reader io.Reader = os.Stdin
	parser := NewMPEGTSParser(&reader)

	// track unique PIDs
	pids := make([]int, 0)
	pidsMap := make(map[int]bool)
	for {
		n, err := parser.Parse(&packetsBuffer)
		for i := 0; i < n; i++ {
			pid := packetsBuffer[i].PID
			if !pidsMap[pid] {
				pids = append(pids, pid)
				pidsMap[pid] = true
			}
		}

		if err != nil {
			// truncated or normal EOF are both acceptable
			if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
				break
			} else {
				fmt.Println("Error:", err.Error())
				os.Exit(1)
				return
			}
		}
	}
	// sort and print unique PIDs
	slices.Sort(pids)
	for _, pid := range pids {
		fmt.Printf("0x%04x\n", pid)
	}
}
