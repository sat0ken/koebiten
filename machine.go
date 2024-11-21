//go:build tinygo && !macropad_rp2040

package koebiten

import (
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/encoders"
	"tinygo.org/x/drivers/ssd1306"
)

var (
	colPins          []machine.Pin
	rowPins          []machine.Pin
	rotaryPins       []machine.Pin
	gpioPins         []machine.Pin
	adcPins          []ADCDevice
	enc              *encoders.QuadratureDevice
	encOld           int
	state            []State
	cycle            []int
	duration         []int
	invertRotaryPins = false
)

const (
	debounce = 0
)

type State uint8

const (
	None State = iota
	NoneToPress
	Press
	PressToRelease
)

type ADCDevice struct {
	ADC         machine.ADC
	PressedFunc func() bool
}

func (a ADCDevice) Get() bool {
	return a.PressedFunc()
}

func init() {
	i2c := machine.I2C0
	i2c.Configure(machine.I2CConfig{
		Frequency: 2_800_000,
		// rp2040
		SDA: machine.GPIO0,
		SCL: machine.GPIO1,
		// xiao-samd21 or xiao-rp2040
		//SDA: machine.D4,
		//SCL: machine.D5,
	})

	d := ssd1306.NewI2C(i2c)
	d.Configure(ssd1306.Config{
		Address: 0x3C,
		Width:   128,
		Height:  64,
	})
	d.SetRotation(drivers.Rotation180)
	d.ClearDisplay()
	display = &d

	// rp2040
	gpioPins = []machine.Pin{
		machine.GPIO4,  // up
		machine.GPIO5,  // left
		machine.GPIO6,  // down
		machine.GPIO7,  // right
		machine.GPIO27, // A
		machine.GPIO28, // B
	}
	// xiao-samd21 or xiao-rp2040
	//gpioPins = []machine.Pin{
	//	machine.D10, // up
	//	machine.D9,  // left
	//	machine.D8,  // down
	//	machine.D7,  // right
	//	machine.D1,  // A
	//	machine.D2,  // B
	//}

	for _, p := range gpioPins {
		p.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}

	state = make([]State, len(gpioPins))
	cycle = make([]int, len(gpioPins))
	duration = make([]int, len(gpioPins))

}

func keyUpdate() {
	keyGpioUpdate()
}

func keyGpioUpdate() {
	for r := range gpioPins {
		current := !gpioPins[r].Get()
		idx := r + len(colPins)*len(rowPins)

		switch state[idx] {
		case None:
			if current {
				if cycle[idx] >= debounce {
					state[idx] = NoneToPress
					cycle[idx] = 0
				} else {
					cycle[idx]++
				}
			} else {
				cycle[idx] = 0
			}
		case NoneToPress:
			state[idx] = Press
			theInputState.keyDurations[idx]++
			AppendJustPressedKeys([]Key{Key(idx)})
		case Press:
			AppendPressedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx]++
			if current {
				cycle[idx] = 0
				duration[idx]++
			} else {
				if cycle[idx] >= debounce {
					state[idx] = PressToRelease
					cycle[idx] = 0
					duration[idx] = 0
				} else {
					cycle[idx]++
				}
			}
		case PressToRelease:
			state[idx] = None
			AppendJustReleasedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx] = 0
		}
	}
}

func keyRotaryUpdate() {
	rot := []bool{false, false}
	if newValue := enc.Position(); newValue != encOld {
		if newValue < encOld {
			rot[0] = true
		} else {
			rot[1] = true
		}
		encOld = newValue
	}

	for c, current := range rot {
		idx := c + len(colPins)*len(rowPins) + 2
		switch state[idx] {
		case None:
			if current {
				state[idx] = NoneToPress
			} else {
			}
		case NoneToPress:
			if current {
				state[idx] = Press
			} else {
				state[idx] = PressToRelease
			}
			theInputState.keyDurations[idx]++
			AppendJustPressedKeys([]Key{Key(idx)})
		case Press:
			AppendPressedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx]++
			if current {
			} else {
				state[idx] = PressToRelease
			}
		case PressToRelease:
			if current {
				state[idx] = NoneToPress
			} else {
				state[idx] = None
			}
			AppendJustReleasedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx] = 0
		}
	}
}

func keyMatrixUpdate() {
	for c := range colPins {
		for r := range rowPins {
			colPins[c].Configure(machine.PinConfig{Mode: machine.PinOutput})
			colPins[c].High()
			current := rowPins[r].Get()
			idx := r*len(colPins) + c

			switch state[idx] {
			case None:
				if current {
					if cycle[idx] >= debounce {
						state[idx] = NoneToPress
						cycle[idx] = 0
					} else {
						cycle[idx]++
					}
				} else {
					cycle[idx] = 0
				}
			case NoneToPress:
				state[idx] = Press
				theInputState.keyDurations[idx]++
				AppendJustPressedKeys([]Key{Key(idx)})
			case Press:
				AppendPressedKeys([]Key{Key(idx)})
				theInputState.keyDurations[idx]++
				if current {
					cycle[idx] = 0
					duration[idx]++
				} else {
					if cycle[idx] >= debounce {
						state[idx] = PressToRelease
						cycle[idx] = 0
						duration[idx] = 0
					} else {
						cycle[idx]++
					}
				}
			case PressToRelease:
				state[idx] = None
				AppendJustReleasedKeys([]Key{Key(idx)})
				theInputState.keyDurations[idx] = 0
			}

			colPins[c].Low()
			colPins[c].Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
		}
	}
}

func keyJoystickUpdate() {
	for r, p := range adcPins {
		current := p.Get()
		idx := r + len(colPins)*len(rowPins) + len(gpioPins) + len(rotaryPins)

		switch state[idx] {
		case None:
			if current {
				if cycle[idx] >= debounce {
					state[idx] = NoneToPress
					cycle[idx] = 0
				} else {
					cycle[idx]++
				}
			} else {
				cycle[idx] = 0
			}
		case NoneToPress:
			state[idx] = Press
			theInputState.keyDurations[idx]++
			AppendJustPressedKeys([]Key{Key(idx)})
		case Press:
			AppendPressedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx]++
			if current {
				cycle[idx] = 0
				duration[idx]++
			} else {
				if cycle[idx] >= debounce {
					state[idx] = PressToRelease
					cycle[idx] = 0
					duration[idx] = 0
				} else {
					cycle[idx]++
				}
			}
		case PressToRelease:
			state[idx] = None
			AppendJustReleasedKeys([]Key{Key(idx)})
			theInputState.keyDurations[idx] = 0
		}
	}
}
