package main

import (
	"fmt"
	"math/bits"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	bitmapSize     = 1 << 26 // 2^32 IPs / 64
	workerBufSize  = 4 * 1024 * 1024
	ipMaxLength    = uint8(19)
	octetMaxLength = uint8(3)
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ip_addr_counter.go <file>")
		return
	}

	//f, _ := os.Create("mem_profile.prof")
	//defer f.Close()
	//defer pprof.WriteHeapProfile(f)

	fileName := os.Args[1]
	file, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	stat, _ := file.Stat()
	fileSize := stat.Size()

	start := time.Now()

	bitmap := make([]uint64, bitmapSize)

	numWorkers := runtime.NumCPU()
	var wg sync.WaitGroup

	jobs := make(chan int64, numWorkers*10)

	for range numWorkers {
		wg.Go(func() {
			worker(file, jobs, bitmap)
		})
	}

	for offset := int64(0); offset < fileSize; offset += workerBufSize {
		jobs <- offset
	}
	close(jobs)

	wg.Wait()

	total := countSetBits(bitmap)

	fmt.Printf("Unique IPs: %d\n", total)
	fmt.Printf("Time taken: %v\n", time.Since(start))
}

func worker(file *os.File, jobs <-chan int64, bitmap []uint64) {
	buf := make([]byte, workerBufSize+128)

	for offset := range jobs {
		endIdx, _ := file.ReadAt(buf, offset)

		if endIdx == 0 {
			continue
		}

		startIdx := findStartIdx(buf, offset, endIdx)

		parseAndSet(buf[startIdx:endIdx], offset, bitmap)
	}
}

func findStartIdx(buf []byte, offset int64, endIdx int) int {
	startIdx := 0

	if offset > 0 {
		for startIdx < endIdx && buf[startIdx] != '\n' {
			startIdx++
		}
		startIdx++
	}

	return startIdx
}

func parseAndSet(data []byte, offset int64, bitmap []uint64) {
	ip := uint32(0)
	octet := uint32(0)

	ipLength := uint8(0)
	octetLength := uint8(0)
	numDotSwitch := false
	switchedTimes := int8(0)

	ipValid := true

	for i := 0; i < len(data); i++ {
		b := data[i]

		if b >= '0' && b <= '9' {
			octet = octet*10 + uint32(b-'0')

			ipLength += 1
			octetLength += 1

			if !numDotSwitch {
				numDotSwitch = true
				switchedTimes += 1
			}
		} else if b == '.' {
			if octet > 255 || octetLength > octetMaxLength {
				ipValid = false
			}

			ip = (ip << 8) | octet
			octet = 0

			ipLength += 1
			octetLength = 0

			if numDotSwitch {
				numDotSwitch = false
				switchedTimes += 1
			}
		} else if b == '\n' {
			if octet > 255 || switchedTimes != 7 || ipLength > ipMaxLength {
				ipValid = false
			}

			if ipValid {
				ip = (ip << 8) | octet
				setBit(ip, bitmap)
			}

			ip = 0
			octet = 0

			ipLength = 0
			octetLength = 0
			numDotSwitch = false
			switchedTimes = 0

			ipValid = true
		} else {
			ipValid = false
			fmt.Printf("Invalid IP detected in line %d\n", offset+int64(i))
		}
	}
}

func setBit(ip uint32, bitmap []uint64) {
	idx := ip / 64
	mask := uint64(1) << (ip % 64)

	for {
		old := atomic.LoadUint64(&bitmap[idx])
		if old&mask != 0 {
			return
		}
		if atomic.CompareAndSwapUint64(&bitmap[idx], old, old|mask) {
			return
		}
	}
}

func countSetBits(bitmap []uint64) int64 {
	var count int64 = 0
	for _, v := range bitmap {
		count += int64(bits.OnesCount64(v))
	}
	return count
}
