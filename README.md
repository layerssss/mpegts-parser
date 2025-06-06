# MPEG-TS Parser

This repo contains the the code for parsing MPEG-TS packets from a portion of byte stream, validating this packet format:

1. The streaming format is made up of individual packets
2. Each packet is 188 bytes long
3. Every packet begins with a “sync byte” which has hex value `0x47`. Note this is also a valid value in the payload of the packet.
4. Each packet has an ID, known as the PID that is 13 bits long. The PID is stored in the last 5 bits of the second byte, and all 8 bits of the third byte of a packet eg:

```
0               1               2               3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|SYNC BYTE(0x47)|Flags|      PID                | ... Packet payload
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

The `MPEGTSParser` class is designed with streaming in mind. Input can be file byte stream / TCP / HTTP stream where start and finish points are unknown at a time point when buffered data needs to be processed / parsed. That means the parser needs to process the input data from `io.reader` where the input data can continously read without reaching an `EOF`. Parser output should also be copied into a supplied fixed-size buffer which can go into next step of the pipeline. The `Parse` interface mimics the interface of [`io.reader`](https://pkg.go.dev/io#Reader).

The byte stream can be interrupted / truncated just like real world streaming scenario. `Parse` method can return as many already completedly parsed packets when this happens.

The parser can also recover from "corrupted" input stream. The parser can "retry" calling `Parse` method to resume parsing the rest of the stream after encounter format errors.

The binary (in `main.go`) can parse files starting with a full or partial packet, having following packets valid, while allowing the last packet to be truncated (discarded).

The parser requires at least 3 complete valid packets to start parsing packets, in order to avoid misidentifying the bytes from the payload to be the sync byte.

## Build (binary)

```
brew install go
go build
cat input-file.ts | ./mpegts-parser
echo $?
```

Expected output for `test_success.ts`:

```
0x0000
0x0011
0x0020
0x0021
0x0022
0x0023
0x0024
0x0025
0x1fff
```

Expected output for `test_failure.ts`:

```
Error:  No sync byte present in packet 20535, offset 3860768
```

## Run tests

```
go test -v ./mpegts_parser
```
