package mpegts_parser_test

import (
	"bytes"
	"io"
	"testing"

	. "github.com/layerssss/mpegts-parser/mpegts_parser"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMpegtsParser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MpegtsParser Suite")
}

var _ = Describe("parsePacket", func() {
	It("should verify buffer size", func() {
		packet := MPEGTSPacket{}
		buffer := make([]byte, 10)
		parser := MPEGTSParser{}
		err := parser.ParsePacket(&packet, &buffer, 0)
		Expect(err.Error()).To(ContainSubstring("buffer too small"))
	})

	It("should verify sync byte", func() {
		packet := MPEGTSPacket{}
		parser := MPEGTSParser{}
		buffer := make([]byte, MPEGTS_PACKET_SIZE)
		buffer[0] = 0x00 // not sync byte
		err := parser.ParsePacket(&packet, &buffer, 0)
		Expect(err.Error()).To(ContainSubstring("No sync byte present"))
	})

	It("should parse valid packet", func() {
		packet := MPEGTSPacket{}
		parser := MPEGTSParser{}
		buffer := make([]byte, MPEGTS_PACKET_SIZE)
		for i := 1; i < MPEGTS_PACKET_SIZE; i++ {
			buffer[i] = byte(i) // fill the rest of the packet with dummy data
		}
		buffer[0] = MPEGTS_SYNC_BYTE // valid sync byte
		err := parser.ParsePacket(&packet, &buffer, 0)
		Expect(err).To(BeNil())
		Expect(packet.PID).To(Equal(258))
		Expect(packet.Flags).To(Equal([]bool{false, false, false}))
		Expect(packet.Payload).To(HaveLen(MPEGTS_PACKET_SIZE - 3))
	})
})

var _ = Describe("Parse", func() {
	It("Parse a 5 packets valid stream", func() {
		buffer := make([]byte, MPEGTS_PACKET_SIZE*5)
		for i := 0; i < 5; i++ {
			makePacket(&buffer, i*MPEGTS_PACKET_SIZE, i)
		}
		var reader io.Reader = bytes.NewReader(buffer)
		packetsBuffer := make([]MPEGTSPacket, 1000)
		parser := NewMPEGTSParser(&reader)

		numberOfPacket, err := parser.Parse(&packetsBuffer)
		Expect(err).To(BeNil())
		Expect(numberOfPacket).To(Equal(5))
		Expect(parser.BytesRead).To(Equal(MPEGTS_PACKET_SIZE * 5))
		Expect(parser.PacketsParsed).To(Equal(5))
		Expect(parser.BytesParsed).To(Equal(MPEGTS_PACKET_SIZE * 5))

		numberOfPacket, err = parser.Parse(&packetsBuffer)
		Expect(err).To(MatchError(io.EOF))
		Expect(numberOfPacket).To(Equal(0))
	})

	It("Parse a 3 packets valid stream starting with garbage", func() {
		buffer := make([]byte, 5+MPEGTS_PACKET_SIZE*3)
		for i := 0; i < 3; i++ {
			makePacket(&buffer, 5+i*MPEGTS_PACKET_SIZE, i)
		}
		var reader io.Reader = bytes.NewReader(buffer)
		packetsBuffer := make([]MPEGTSPacket, 1000)
		parser := NewMPEGTSParser(&reader)

		numberOfPacket, err := parser.Parse(&packetsBuffer)
		Expect(err).To(BeNil())
		Expect(numberOfPacket).To(Equal(3))
		Expect(parser.BytesRead).To(Equal(5 + MPEGTS_PACKET_SIZE*3))
		Expect(parser.PacketsParsed).To(Equal(3))
		Expect(parser.BytesParsed).To(Equal(5 + MPEGTS_PACKET_SIZE*3))

		numberOfPacket, err = parser.Parse(&packetsBuffer)
		Expect(err).To(MatchError(io.EOF))
		Expect(numberOfPacket).To(Equal(0))
	})

	It("Should handle garbage with sync byte before valid packets", func() {
		buffer := make([]byte, 5+MPEGTS_PACKET_SIZE*5)
		for i := 0; i < 5; i++ {
			buffer[i] = MPEGTS_SYNC_BYTE
		}
		for i := 0; i < 5; i++ {
			makePacket(&buffer, 5+i*MPEGTS_PACKET_SIZE, i)
		}
		var reader io.Reader = bytes.NewReader(buffer)
		packetsBuffer := make([]MPEGTSPacket, 1000)
		parser := NewMPEGTSParser(&reader)

		numberOfPacket, err := parser.Parse(&packetsBuffer)
		Expect(err).To(BeNil())
		Expect(numberOfPacket).To(Equal(5))
		Expect(parser.BytesRead).To(Equal(5 + MPEGTS_PACKET_SIZE*5))
		Expect(parser.PacketsParsed).To(Equal(5))
		Expect(parser.BytesParsed).To(Equal(5 + MPEGTS_PACKET_SIZE*5))

		numberOfPacket, err = parser.Parse(&packetsBuffer)
		Expect(err).To(MatchError(io.EOF))
		Expect(numberOfPacket).To(Equal(0))
	})

	It("Should parse packets followed by garbage", func() {
		buffer := make([]byte, 5+MPEGTS_PACKET_SIZE*5)
		for i := 0; i < 5; i++ {
			makePacket(&buffer, 5+i*MPEGTS_PACKET_SIZE, i)
		}
		buffer[5+MPEGTS_PACKET_SIZE*4] = 0x00 // make the last packet invalid
		var reader io.Reader = bytes.NewReader(buffer)
		packetsBuffer := make([]MPEGTSPacket, 1000)
		parser := NewMPEGTSParser(&reader)

		numberOfPacket, err := parser.Parse(&packetsBuffer)
		Expect(err.Error()).To(ContainSubstring("No sync byte present"))
		Expect(numberOfPacket).To(Equal(4))
		Expect(parser.BytesRead).To(Equal(5 + MPEGTS_PACKET_SIZE*5))
		Expect(parser.PacketsParsed).To(Equal(4))
		Expect(parser.BytesParsed).To(Equal(5 + MPEGTS_PACKET_SIZE*4))
	})

	It("Should only fill up packetBuffer size", func() {
		buffer := make([]byte, MPEGTS_PACKET_SIZE*5)
		for i := 0; i < 5; i++ {
			makePacket(&buffer, i*MPEGTS_PACKET_SIZE, i)
		}
		bytesReader := bytes.NewReader(buffer)
		var reader io.Reader = bytesReader
		packetsBuffer := make([]MPEGTSPacket, 3)
		parser := NewMPEGTSParser(&reader)

		numberOfPacket, err := parser.Parse(&packetsBuffer)
		Expect(err).To(BeNil())
		Expect(numberOfPacket).To(Equal(3))
		Expect(parser.BytesRead).To(Equal(MPEGTS_PACKET_SIZE * 3))
		Expect(parser.PacketsParsed).To(Equal(3))
		Expect(parser.BytesParsed).To(Equal(MPEGTS_PACKET_SIZE * 3))
		Expect(bytesReader.Len()).To(Equal(MPEGTS_PACKET_SIZE * 2))

		numberOfPacket, err = parser.Parse(&packetsBuffer)
		Expect(err).To(BeNil())
		Expect(numberOfPacket).To(Equal(2))
		Expect(parser.BytesRead).To(Equal(MPEGTS_PACKET_SIZE * 5))
		Expect(parser.PacketsParsed).To(Equal(5))
		Expect(parser.BytesParsed).To(Equal(MPEGTS_PACKET_SIZE * 5))
		Expect(bytesReader.Len()).To(Equal(0)) // all packets read
	})

	It("Should return EOF reading an empty stream", func() {
		buffer := make([]byte, 0)
		var reader io.Reader = bytes.NewReader(buffer)
		packetsBuffer := make([]MPEGTSPacket, 1000)
		parser := NewMPEGTSParser(&reader)

		numberOfPacket, err := parser.Parse(&packetsBuffer)
		Expect(err).To(MatchError(io.EOF))
		Expect(numberOfPacket).To(Equal(0))
	})

	It("Should return Unexpected EOF reading a stream being truncated", func() {
		buffer := make([]byte, MPEGTS_PACKET_SIZE*5-5)
		for i := 0; i < 5; i++ {
			makePacket(&buffer, i*MPEGTS_PACKET_SIZE, i)
		}
		var reader io.Reader = bytes.NewReader(buffer)
		packetsBuffer := make([]MPEGTSPacket, 1000)
		parser := NewMPEGTSParser(&reader)

		numberOfPacket, err := parser.Parse(&packetsBuffer)
		Expect(err).To(MatchError(io.ErrUnexpectedEOF))
		Expect(numberOfPacket).To(Equal(4)) // 4 packets can be read, the last one is truncated
		Expect(parser.BytesRead).To(Equal(len(buffer)))
		Expect(parser.PacketsParsed).To(Equal(4))
		Expect(parser.BytesParsed).To(Equal(MPEGTS_PACKET_SIZE * 4))
	})

})

func makePacket(buffer *[]byte, offset int, pid int) {
	(*buffer)[offset] = MPEGTS_SYNC_BYTE         // set sync byte
	(*buffer)[offset+1] = byte(pid >> 8)         // set PID high byte
	(*buffer)[offset+2] = byte(pid & 0b11111111) // set PID low byte
}
