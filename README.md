# IP-Addr-Counter solution

Task: https://github.com/Ecwid/new-job/blob/master/IP-Addr-Counter.md

## Explanation

The general idea is basically the same as the naive approach suggested in the test:
“read line by line, put lines into a HashSet”, but with several optimizations:

- As IPv4 is basically a `uint32`, we can represent it as a bitmap. A straightforward idea would be to store a `[]bool`, 
  but `bool` takes 1 byte in memory (not 1 bit as I expected), so a full bitmap would take `2^32 * 1 byte ≈ 4GB`. 
  Instead, I store an array of integers and update particular bits in it.

- As the structure of the file is pretty simple, we can split it into N parts and process them with N threads. 
  But we need to take care of IPs on the boundaries, as we can fall into the middle of an IP.

- I found that `atomic.CompareAndSwapUint64` is a very efficient way to update an integer (instead of using a mutex), 
  as it's a CPU-level primitive. I could potentially implement N mutexes for different parts of the bitmap, but I didn't try it.

- I used `file.ReadAt` because it allows reusing the buffer I initialized. As I understand, `bufio.Scanner` would create a new buffer each time.

## Execution (time & memory)

I ran it only on half of the data, as that's the most that fits into my WSL Ubuntu.
(I wanted to utilize CPU at the fullest for benchmarking instead of reading from another drive.)

```shell
➜ go run ip_addr_counter.go ip_addresses.half00
Unique IPs: 1000000000 # It seems weird but for the whole file it also returns 1b
Time taken: 35.028797803s
```

Memory taken: ~500MB, see profile results

[memory_profile.html](https://html-preview.github.io/?url=https://github.com/mikhliuk-k/ip_addr_counter/blob/main/memory_profile.html)

