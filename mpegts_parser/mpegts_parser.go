package mpegts_parser

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

type MPEGTSParser struct {
	Source        *io.Reader
	Synced        bool
	BytesRead     int
	BytesParsed   int
	PacketsParsed int
}

type MPEGTSPacket struct {
	PID     int
	Payload []byte
	Flags   []bool
}

type MPEGTSParserError struct {
	Message       string
	BytesRead     int
	PacketsParsed int
}

func (e MPEGTSParserError) Error() string {
	return fmt.Sprintf("%s in packet %d, offset %d", e.Message, e.PacketsParsed, e.BytesRead)
}

const MPEGTS_PACKET_SIZE = 188
const MPEGTS_SYNC_BYTE = 0x47
const MINIMUM_PACKETS_BUFFER_SIZE = 2

func NewMPEGTSParser(source *io.Reader) *MPEGTSParser {
	return &MPEGTSParser{
		Source:        source,
		Synced:        false,
		BytesRead:     0, // total bytes read from the source
		BytesParsed:   0, // total bytes read from the source excluding not yet parsed portion
		PacketsParsed: 0, // total packets parsed
	}
}

// Parse MPEG-TS packets from the source into the provided packetBuffer, filling it up to len(packetBuffer). Return number of packets successfully parsed and any error encountered.
//
// * err = io.ErrUnexpectedEOF if an EOF is reached when a packet has been partially parsed (truncated).
// * err = io.EOF if EOF is reached after successfully parsing the last packet. (n = 0)
func (p *MPEGTSParser) Parse(packetBuffer *[]MPEGTSPacket) (n int, err error) {
	numberOfPackets := 0
	packetsBufferOffset := 0
	if !p.Synced {
		packets, err := p.sync()
		if err != nil {
			return 0, err
		}
		p.Synced = true

		(*packetBuffer)[0] = packets[0]
		(*packetBuffer)[1] = packets[1]
		(*packetBuffer)[2] = packets[2]
		numberOfPackets = 3
		packetsBufferOffset = 3
	}

	// read packets until filling up PacketBuffer
	buffer := make([]byte, MPEGTS_PACKET_SIZE)
	for packetsBufferOffset < len(*packetBuffer) {
		n, err := io.ReadFull(*p.Source, buffer)
		p.BytesRead += n
		if err != nil {
			if err == io.EOF && numberOfPackets > 0 {
				// EOF reached, return the number of packets read so far
				return numberOfPackets, nil
			}
			return numberOfPackets, err
		}
		if err = p.ParsePacket(&(*packetBuffer)[packetsBufferOffset], &buffer, 0); err != nil {
			return numberOfPackets, err
		}
		packetsBufferOffset++
		numberOfPackets++
	}

	return numberOfPackets, nil
}

// Try to locate the sync byte offset by verifying it in the following 2 packets. Return the 3 already parsed packets.
//
// Note: There is a very very slight chance that the sync byte (0x47) happens to apear in the payload at the same position in 3 consecutive packets, this does not handle that situation.
func (p *MPEGTSParser) sync() (packets []MPEGTSPacket, err error) {
	// need to read at least two packets length to verify sync byte location
	buffer := make([]byte, MPEGTS_PACKET_SIZE*3)
	n, err := io.ReadFull(*p.Source, buffer)
	p.BytesRead += n
	if err != nil {
		return nil, err
	}
	for firstPacketOffset := 0; firstPacketOffset < MPEGTS_PACKET_SIZE; firstPacketOffset++ {
		secondPacketOffset := firstPacketOffset + MPEGTS_PACKET_SIZE
		thirdPacketOffset := secondPacketOffset + MPEGTS_PACKET_SIZE
		if buffer[firstPacketOffset] == MPEGTS_SYNC_BYTE &&
			buffer[secondPacketOffset] == MPEGTS_SYNC_BYTE &&
			buffer[thirdPacketOffset] == MPEGTS_SYNC_BYTE {
			// found potential sync byte, parse the first packet
			packets := make([]MPEGTSPacket, 3)
			if err = p.ParsePacket(&packets[0], &buffer, firstPacketOffset); err != nil {
				return nil, err
			}
			p.BytesParsed += firstPacketOffset
			if err = p.ParsePacket(&packets[1], &buffer, secondPacketOffset); err != nil {
				return nil, err
			}

			// read the rest of the last packet, and parse
			buffer2 := make([]byte, MPEGTS_PACKET_SIZE-(len(buffer)-thirdPacketOffset))
			n, err = io.ReadFull(*p.Source, buffer2)
			p.BytesRead += n
			if err != nil {
				return nil, err
			}
			thirdPacketBuffer := append(buffer[thirdPacketOffset:], buffer2...)
			if err = p.ParsePacket(&packets[2], &thirdPacketBuffer, 0); err != nil {
				return nil, err
			}

			return packets, nil
		}
	}
	return nil, p.NoSyncBytePresentError()
}

// Parse a single packet from the buffer at the given offset.
func (p *MPEGTSParser) ParsePacket(packet *MPEGTSPacket, buffer *[]byte, offset int) error {
	if len(*buffer) < offset+MPEGTS_PACKET_SIZE {
		return errors.New("buffer too small to parse packet")
	}
	// first byte: sync byte
	if (*buffer)[offset] != MPEGTS_SYNC_BYTE {
		return p.NoSyncBytePresentError()
	}
	// flags: the first three bits of the second byte
	packet.Flags = make([]bool, 3)
	packet.Flags[0] = (*buffer)[offset+1]&0b10000000 != 0
	packet.Flags[1] = (*buffer)[offset+1]&0b01000000 != 0
	packet.Flags[2] = (*buffer)[offset+1]&0b00100000 != 0

	// PID: last 5 bits of the second byte and the third byte
	packet.PID = int((*buffer)[offset+1]&0b00011111)<<8 | int((*buffer)[offset+2])

	packet.Payload = (*buffer)[offset+3 : offset+MPEGTS_PACKET_SIZE]
	p.PacketsParsed += 1
	p.BytesParsed += MPEGTS_PACKET_SIZE
	return nil
}

func (p *MPEGTSParser) ParseError(message string) MPEGTSParserError {
	return MPEGTSParserError{
		Message:       message,
		BytesRead:     p.BytesRead,
		PacketsParsed: p.PacketsParsed,
	}
}

func (p *MPEGTSParser) NoSyncBytePresentError() MPEGTSParserError {
	return p.ParseError("No sync byte present")
}
