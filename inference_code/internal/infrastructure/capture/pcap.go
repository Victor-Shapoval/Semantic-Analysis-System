package capture

import (
	"fmt"

	"semantic-analysis-system/internal/domain/sv"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// svEtherType — EtherType IEC 61850-9-2 Sampled Values.
const svEtherType = 0x88ba

type Capturer struct {
	iface  string
	appID  uint16
	srcMAC string
	dstMAC string
	parser *sv.Parser
}

func NewCapturer(iface string, appID uint16, srcMAC, dstMAC string) *Capturer {
	return &Capturer{
		iface:  iface,
		appID:  appID,
		srcMAC: srcMAC,
		dstMAC: dstMAC,
		parser: &sv.Parser{AppID: appID},
	}
}

func (c *Capturer) bpfFilter() string {
	var mac string
	if c.srcMAC != "" {
		mac += " and ether src " + c.srcMAC
	}
	if c.dstMAC != "" {
		mac += " and ether dst " + c.dstMAC
	}
	return fmt.Sprintf("(ether proto 0x%04x%s) or (vlan and ether proto 0x%04x%s)",
		svEtherType, mac, svEtherType, mac)
}

func (c *Capturer) Run(done <-chan struct{}, out chan<- *sv.SVFrame, errs chan<- error) error {
	handle, err := pcap.OpenLive(c.iface, 65536, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("capture: open %s: %w", c.iface, err)
	}
	defer handle.Close()

	if err := handle.SetBPFFilter(c.bpfFilter()); err != nil {
		return fmt.Errorf("capture: BPF filter: %w", err)
	}

	src := gopacket.NewPacketSource(handle, layers.LayerTypeEthernet)
	src.NoCopy = true

	for {
		select {
		case <-done:
			return nil
		case pkt, ok := <-src.Packets():
			if !ok {
				return nil
			}
			var payload []byte
			if dot1q := pkt.Layer(layers.LayerTypeDot1Q); dot1q != nil {
				payload = dot1q.LayerPayload()
			} else {
				ethLayer := pkt.Layer(layers.LayerTypeEthernet)
				if ethLayer == nil {
					continue
				}
				payload = ethLayer.(*layers.Ethernet).LayerPayload()
			}
			if len(payload) == 0 {
				continue
			}
			frame, err := c.parser.Parse(payload)
			if err != nil {
				select {
				case errs <- err:
				default:
				}
				continue
			}
			select {
			case out <- frame:
			case <-done:
				return nil
			}
		}
	}
}
