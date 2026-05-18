package sv

type SVFrame struct {
	// AppID identifies the SV stream (from the APDU header)
	AppID uint16

	SmpCnt uint16

	Channels [8]int32

	Quality [8]uint32
}

// CurrentA returns phase A current in amperes.
func (f *SVFrame) CurrentA() float64 { return float64(f.Channels[0]) / 1000.0 }

// CurrentB returns phase B current in amperes.
func (f *SVFrame) CurrentB() float64 { return float64(f.Channels[1]) / 1000.0 }

// CurrentC returns phase C current in amperes.
func (f *SVFrame) CurrentC() float64 { return float64(f.Channels[2]) / 1000.0 }

// CurrentN returns neutral current in amperes.
func (f *SVFrame) CurrentN() float64 { return float64(f.Channels[3]) / 1000.0 }

// VoltageA returns phase A voltage in volts.
func (f *SVFrame) VoltageA() float64 { return float64(f.Channels[4]) / 100.0 }

// VoltageB returns phase B voltage in volts.
func (f *SVFrame) VoltageB() float64 { return float64(f.Channels[5]) / 100.0 }

// VoltageC returns phase C voltage in volts.
func (f *SVFrame) VoltageC() float64 { return float64(f.Channels[6]) / 100.0 }

// VoltageN returns neutral voltage in volts.
func (f *SVFrame) VoltageN() float64 { return float64(f.Channels[7]) / 100.0 }
