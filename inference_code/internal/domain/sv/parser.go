package sv

import (
	"encoding/binary"
	"fmt"
)

// Offsets in an IEC 61850-9-2LE APDU after Ethernet and VLAN headers.
// Structure: AppID(2) + Length(2) + Reserved1(2) + Reserved2(2), then ASN.1 BER:
// savPdu tag(1) + len(1) + noASDU tag(1) + len(1) + value(1) + seqASDU tag(1) + len...
//
// One ASDU in the 9-2LE profile contains 8 channels with 4 bytes of value
// and 4 bytes of quality per channel, for 64 bytes of sequence data:
//
//	svID, smpCnt(2), confRev(4), smpSynch(1), seqData(64)
const (
	apduHeaderLen = 8 // AppID + Length + Reserved1 + Reserved2
	channelCount  = 8 // number of analog channels in 9-2LE
	channelBytes  = 8 // 4 bytes value + 4 bytes quality per channel
)

// Parser decodes raw Ethernet-frame bytes into an SVFrame.
// It expects the payload to start with the APDU, without Ethernet/VLAN headers.
type Parser struct {
	AppID uint16 // AppID filter; 0 = no filter
}

// Parse parses APDU bytes and returns an SVFrame.
// payload must start with the AppID[0] byte, octet 14 of the Ethernet frame.
func (p *Parser) Parse(payload []byte) (*SVFrame, error) {
	if len(payload) < apduHeaderLen {
		return nil, fmt.Errorf("sv: payload too short: %d bytes", len(payload))
	}

	appID := binary.BigEndian.Uint16(payload[0:2])
	if p.AppID != 0 && appID != p.AppID {
		return nil, fmt.Errorf("sv: appID mismatch: got %#x, want %#x", appID, p.AppID)
	}

	// strip the APDU header
	payload = payload[apduHeaderLen:]

	// Enter savPDU: tag 0x60 is a container, and expectTagAt returns an offset inside it.
	offset, err := expectTagAt(payload, 0, 0x60)
	if err != nil {
		return nil, fmt.Errorf("sv: savPDU: %w", err)
	}

	// Skip noASDU: tag 0x80, length 1, value 1 byte.
	offset, err = skipTagAt(payload, offset)
	if err != nil {
		return nil, fmt.Errorf("sv: noASDU: %w", err)
	}

	// Enter seqASDU: tag 0xa2 is a container.
	offset, err = expectTagAt(payload, offset, 0xa2)
	if err != nil {
		return nil, fmt.Errorf("sv: seqASDU: %w", err)
	}

	// Enter ASDU: tag 0x30 is a container.
	offset, err = expectTagAt(payload, offset, 0x30)
	if err != nil {
		return nil, fmt.Errorf("sv: ASDU: %w", err)
	}

	// Skip svID: tag 0x80 is a variable-length string.
	offset, err = skipTagAt(payload, offset)
	if err != nil {
		return nil, fmt.Errorf("sv: svID: %w", err)
	}

	// read smpCnt: tag 0x82, length 2
	offset, err = expectTagAt(payload, offset, 0x82)
	if err != nil {
		return nil, fmt.Errorf("sv: smpCnt tag: %w", err)
	}
	if offset+2 > len(payload) {
		return nil, fmt.Errorf("sv: smpCnt value out of bounds")
	}
	smpCnt := binary.BigEndian.Uint16(payload[offset : offset+2])
	offset += 2

	// Skip confRev: tag 0x83, length 4.
	offset, err = skipTagAt(payload, offset)
	if err != nil {
		return nil, fmt.Errorf("sv: confRev: %w", err)
	}

	// Skip smpSynch: tag 0x85, length 1.
	offset, err = skipTagAt(payload, offset)
	if err != nil {
		return nil, fmt.Errorf("sv: smpSynch: %w", err)
	}

	// read seqData: tag 0x87, length 64
	offset, err = expectTagAt(payload, offset, 0x87)
	if err != nil {
		return nil, fmt.Errorf("sv: seqData tag: %w", err)
	}

	if offset+channelCount*channelBytes > len(payload) {
		return nil, fmt.Errorf("sv: seqData out of bounds")
	}

	var frame SVFrame
	frame.AppID = appID
	frame.SmpCnt = smpCnt

	for i := 0; i < channelCount; i++ {
		base := offset + i*channelBytes
		frame.Channels[i] = int32(binary.BigEndian.Uint32(payload[base : base+4]))
		frame.Quality[i] = binary.BigEndian.Uint32(payload[base+4 : base+8])
	}

	return &frame, nil
}

// skipTagAt skips tag+length+value at payload[offset] and returns the offset after the value.
func skipTagAt(payload []byte, offset int) (int, error) {
	if offset >= len(payload) {
		return offset, fmt.Errorf("offset %d out of range", offset)
	}
	offset++ // tag
	l, n, err := readLength(payload, offset)
	if err != nil {
		return offset, err
	}
	offset += n + l
	return offset, nil
}

// expectTagAt checks the tag and returns the offset to the start of the value after the length.
func expectTagAt(payload []byte, offset int, tag byte) (int, error) {
	if offset >= len(payload) {
		return offset, fmt.Errorf("offset %d out of range", offset)
	}
	if payload[offset] != tag {
		return offset, fmt.Errorf("expected tag %#x, got %#x", tag, payload[offset])
	}
	offset++
	_, n, err := readLength(payload, offset)
	if err != nil {
		return offset, err
	}
	return offset + n, nil
}

// readLength reads BER length, returns (length, length-byte count, error).
func readLength(payload []byte, offset int) (int, int, error) {
	if offset >= len(payload) {
		return 0, 0, fmt.Errorf("length byte out of range at %d", offset)
	}
	b := payload[offset]
	if b < 0x80 {
		return int(b), 1, nil
	}
	n := int(b & 0x7f)
	if n == 0 || offset+1+n > len(payload) {
		return 0, 0, fmt.Errorf("invalid long-form length at %d", offset)
	}
	var l int
	for i := 0; i < n; i++ {
		l = l<<8 | int(payload[offset+1+i])
	}
	return l, 1 + n, nil
}
