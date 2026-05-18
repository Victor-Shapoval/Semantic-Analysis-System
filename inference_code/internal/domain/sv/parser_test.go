package sv_test

import (
	"encoding/binary"
	"testing"

	"semantic-analysis-system/internal/domain/sv"
)

// AppID(2) + Length(2) + Reserved(4)
// savPDU [0x60 len]
//
//	noASDU [0x80 01] 01
//	seqASDU [0xa2 len]
//	  ASDU [0x30 len]
//	    svID [0x80 04] "test"
//	    smpCnt [0x82 02] smpCnt
//	    confRev [0x83 04] 00000001
//	    smpSynch [0x85 01] 00
//	    seqData [0x87 40] channels×8
func buildTestAPDU(appID, smpCnt uint16, channels [8]int32, quality [8]uint32) []byte {
	seqData := make([]byte, 64)
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint32(seqData[i*8:], uint32(channels[i]))
		binary.BigEndian.PutUint32(seqData[i*8+4:], quality[i])
	}

	svID := []byte("test")

	asduInner := []byte{}
	asduInner = append(asduInner, 0x80, byte(len(svID)))
	asduInner = append(asduInner, svID...)
	asduInner = append(asduInner, 0x82, 0x02,
		byte(smpCnt>>8), byte(smpCnt))
	asduInner = append(asduInner, 0x83, 0x04, 0x00, 0x00, 0x00, 0x01)
	asduInner = append(asduInner, 0x85, 0x01, 0x00)
	asduInner = append(asduInner, 0x87, 0x40)
	asduInner = append(asduInner, seqData...)

	asdu := append([]byte{0x30, byte(len(asduInner))}, asduInner...)
	seqASDU := append([]byte{0xa2, byte(len(asdu))}, asdu...)
	noASDU := []byte{0x80, 0x01, 0x01}
	pduInner := append(noASDU, seqASDU...)
	savPDU := append([]byte{0x60, byte(len(pduInner))}, pduInner...)

	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], appID)
	binary.BigEndian.PutUint16(header[2:4], uint16(len(savPDU)+8))

	return append(header, savPDU...)
}

func TestParser_Parse_BasicFrame(t *testing.T) {
	var channels [8]int32
	// Ia = 1000 A × 1000 = 1_000_000
	channels[0] = 1_000_000
	// Ua = 132790 V × 100 = 13_279_000
	channels[4] = 13_279_000

	var quality [8]uint32
	const appID = 0x4000
	const smpCnt = 42

	payload := buildTestAPDU(appID, smpCnt, channels, quality)

	p := &sv.Parser{AppID: appID}
	frame, err := p.Parse(payload)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if frame.AppID != appID {
		t.Errorf("AppID: got %#x, want %#x", frame.AppID, appID)
	}
	if frame.SmpCnt != smpCnt {
		t.Errorf("SmpCnt: got %d, want %d", frame.SmpCnt, smpCnt)
	}
	if frame.Channels[0] != channels[0] {
		t.Errorf("Channels[0]: got %d, want %d", frame.Channels[0], channels[0])
	}
	if frame.Channels[4] != channels[4] {
		t.Errorf("Channels[4]: got %d, want %d", frame.Channels[4], channels[4])
	}

	wantCurrentA := float64(channels[0]) / 1000.0
	if got := frame.CurrentA(); got != wantCurrentA {
		t.Errorf("CurrentA(): got %v, want %v", got, wantCurrentA)
	}

	wantVoltageA := float64(channels[4]) / 100.0
	if got := frame.VoltageA(); got != wantVoltageA {
		t.Errorf("VoltageA(): got %v, want %v", got, wantVoltageA)
	}
}

func TestParser_Parse_AppIDMismatch(t *testing.T) {
	var channels [8]int32
	var quality [8]uint32
	payload := buildTestAPDU(0x1234, 0, channels, quality)

	p := &sv.Parser{AppID: 0x4000}
	_, err := p.Parse(payload)
	if err == nil {
		t.Fatal("expected AppID mismatch error, got nil")
	}
}

func TestParser_Parse_TooShort(t *testing.T) {
	p := &sv.Parser{}
	_, err := p.Parse([]byte{0x01, 0x02})
	if err == nil {
		t.Fatal("expected error for short payload, got nil")
	}
}
