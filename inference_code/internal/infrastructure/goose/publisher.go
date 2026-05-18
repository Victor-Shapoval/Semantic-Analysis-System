package goose

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	domaingoose "semantic-analysis-system/internal/domain/goose"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type RawPublisher struct {
	handle     *pcap.Handle
	srcMAC     net.HardwareAddr
	dstMAC     net.HardwareAddr
	appID      uint16
	goCbRef    string
	goID       string
	invertTrip bool
}

func NewRawPublisher(iface, dstMAC, goCbRef, goID string, appID uint16, invertTrip bool) (*RawPublisher, error) {
	handle, err := pcap.OpenLive(iface, 65536, true, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("goose publisher: open %s: %w", iface, err)
	}

	src, err := getInterfaceMAC(iface)
	if err != nil {
		handle.Close()
		return nil, err
	}
	dst, err := net.ParseMAC(dstMAC)
	if err != nil {
		handle.Close()
		return nil, fmt.Errorf("goose publisher: invalid dst MAC: %w", err)
	}

	return &RawPublisher{
		handle:     handle,
		srcMAC:     src,
		dstMAC:     dst,
		appID:      appID,
		goCbRef:    goCbRef,
		goID:       goID,
		invertTrip: invertTrip,
	}, nil
}

func (p *RawPublisher) Publish(msg domaingoose.Message) error {
	if p.invertTrip {
		msg.Trip = !msg.Trip
	}
	payload := encodeGOOSEPayload(msg, p.goCbRef, p.goID, p.appID)

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{}

	eth := &layers.Ethernet{
		SrcMAC:       p.srcMAC,
		DstMAC:       p.dstMAC,
		EthernetType: 0x88b8, // IEC 61850 GOOSE EtherType
	}

	err := gopacket.SerializeLayers(buf, opts,
		eth,
		gopacket.Payload(payload),
	)
	if err != nil {
		return fmt.Errorf("goose: serialize: %w", err)
	}

	return p.handle.WritePacketData(buf.Bytes())
}

func (p *RawPublisher) Close() {
	p.handle.Close()
}

func encodeGOOSEPayload(msg domaingoose.Message, goCbRef, goID string, appID uint16) []byte {
	cbRefBytes := []byte(goCbRef)
	goIDBytes := []byte(goID)

	var tripByte byte
	if msg.Trip {
		tripByte = 0x01
	}

	// [gocbRef][timeAllowedToLive][datSet][goID][t][stNum][sqNum][simulation][confRev][ndsCom][numDatSetEntries][allData]
	inner := buildBER(0x80, cbRefBytes)                      // gocbRef
	inner = append(inner, buildBER(0x81, uint16BE(1000))...) // timeAllowedToLive = 1000 ms
	inner = append(inner, buildBER(0x82, cbRefBytes)...)
	inner = append(inner, buildBER(0x83, goIDBytes)...) // goID
	inner = append(inner, buildBER(0x84, utcTimestamp())...)
	inner = append(inner, buildBER(0x85, uint32BE(msg.StNum))...) // stNum
	inner = append(inner, buildBER(0x86, uint32BE(msg.SqNum))...) // sqNum
	inner = append(inner, buildBER(0x87, []byte{0x00})...)        // simulation = FALSE
	inner = append(inner, buildBER(0x88, uint32BE(1))...)         // confRev = 1
	inner = append(inner, buildBER(0x89, []byte{0x00})...)        // ndsCom = FALSE
	inner = append(inner, buildBER(0x8a, uint32BE(1))...)         // numDatSetEntries = 1

	allData := buildBER(0x83, []byte{tripByte}) // BOOLEAN
	inner = append(inner, buildBER(0xab, allData)...)

	pdu := buildBER(0x61, inner) // goosePDU

	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], appID)
	binary.BigEndian.PutUint16(header[2:4], uint16(len(pdu)+8)) // Length
	// Reserved1, Reserved2 = 0

	return append(header, pdu...)
}

func buildBER(tag byte, value []byte) []byte {
	l := len(value)
	var lenBytes []byte
	switch {
	case l < 128:
		lenBytes = []byte{byte(l)}
	case l < 256:
		lenBytes = []byte{0x81, byte(l)}
	default:
		lenBytes = []byte{0x82, byte(l >> 8), byte(l)}
	}
	result := []byte{tag}
	result = append(result, lenBytes...)
	result = append(result, value...)
	return result
}

func utcTimestamp() []byte {
	now := time.Now().UTC()
	t := make([]byte, 8)
	binary.BigEndian.PutUint32(t[0:4], uint32(now.Unix()))
	frac := uint32(float64(now.Nanosecond()) / 1e9 * (1 << 24))
	t[4] = byte(frac >> 16)
	t[5] = byte(frac >> 8)
	t[6] = byte(frac)
	t[7] = 0x18
	return t
}

func uint16BE(v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return b
}

func uint32BE(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

func getInterfaceMAC(name string) (net.HardwareAddr, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Name == name {
			return iface.HardwareAddr, nil
		}
	}
	return nil, fmt.Errorf("interface %q not found", name)
}
