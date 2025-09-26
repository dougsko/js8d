package hardware

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LinuxGPIO implements GPIOInterface using Linux sysfs GPIO
type LinuxGPIO struct {
	exportedPins map[int]bool
	mutex        sync.RWMutex
}

// NewLinuxGPIO creates a new Linux GPIO interface
func NewLinuxGPIO() *LinuxGPIO {
	return &LinuxGPIO{
		exportedPins: make(map[int]bool),
	}
}

// Initialize initializes the Linux GPIO system
func (g *LinuxGPIO) Initialize() error {
	// Check if we have access to GPIO
	if _, err := os.Stat("/sys/class/gpio"); os.IsNotExist(err) {
		return fmt.Errorf("GPIO not available on this system")
	}

	log.Printf("LinuxGPIO: Initialized")
	return nil
}

// Close closes the Linux GPIO system and unexports all pins
func (g *LinuxGPIO) Close() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Unexport all exported pins
	for pin := range g.exportedPins {
		g.unexportPin(pin)
	}

	log.Printf("LinuxGPIO: Closed")
	return nil
}

// SetPin sets a GPIO pin value
func (g *LinuxGPIO) SetPin(pin int, value bool) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Export pin if not already exported
	if !g.exportedPins[pin] {
		if err := g.exportPin(pin); err != nil {
			return fmt.Errorf("failed to export pin %d: %w", pin, err)
		}

		// Set pin direction to output
		if err := g.setPinDirection(pin, "out"); err != nil {
			return fmt.Errorf("failed to set pin %d direction: %w", pin, err)
		}

		g.exportedPins[pin] = true
	}

	// Set pin value
	valuePath := fmt.Sprintf("/sys/class/gpio/gpio%d/value", pin)
	valueStr := "0"
	if value {
		valueStr = "1"
	}

	if err := os.WriteFile(valuePath, []byte(valueStr), 0644); err != nil {
		return fmt.Errorf("failed to set pin %d value: %w", pin, err)
	}

	return nil
}

// GetPin gets a GPIO pin value
func (g *LinuxGPIO) GetPin(pin int) (bool, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Export pin if not already exported
	if !g.exportedPins[pin] {
		if err := g.exportPin(pin); err != nil {
			return false, fmt.Errorf("failed to export pin %d: %w", pin, err)
		}

		// Set pin direction to input
		if err := g.setPinDirection(pin, "in"); err != nil {
			return false, fmt.Errorf("failed to set pin %d direction: %w", pin, err)
		}

		g.exportedPins[pin] = true
	}

	// Read pin value
	valuePath := fmt.Sprintf("/sys/class/gpio/gpio%d/value", pin)
	data, err := os.ReadFile(valuePath)
	if err != nil {
		return false, fmt.Errorf("failed to read pin %d value: %w", pin, err)
	}

	valueStr := strings.TrimSpace(string(data))
	return valueStr == "1", nil
}

// exportPin exports a GPIO pin to userspace
func (g *LinuxGPIO) exportPin(pin int) error {
	exportPath := "/sys/class/gpio/export"
	pinStr := strconv.Itoa(pin)

	// Check if pin is already exported
	pinPath := fmt.Sprintf("/sys/class/gpio/gpio%d", pin)
	if _, err := os.Stat(pinPath); err == nil {
		return nil // Already exported
	}

	// Export the pin
	if err := os.WriteFile(exportPath, []byte(pinStr), 0644); err != nil {
		return fmt.Errorf("failed to export GPIO pin %d: %w", pin, err)
	}

	// Wait for the pin directory to appear
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(pinPath); err == nil {
			log.Printf("LinuxGPIO: Exported pin %d", pin)
			return nil
		}
		// Small delay to allow kernel to create the directory
		time.Sleep(10 * time.Millisecond)
	}

	return fmt.Errorf("pin %d directory did not appear after export", pin)
}

// unexportPin unexports a GPIO pin from userspace
func (g *LinuxGPIO) unexportPin(pin int) error {
	unexportPath := "/sys/class/gpio/unexport"
	pinStr := strconv.Itoa(pin)

	if err := os.WriteFile(unexportPath, []byte(pinStr), 0644); err != nil {
		return fmt.Errorf("failed to unexport GPIO pin %d: %w", pin, err)
	}

	log.Printf("LinuxGPIO: Unexported pin %d", pin)
	return nil
}

// setPinDirection sets the direction of a GPIO pin
func (g *LinuxGPIO) setPinDirection(pin int, direction string) error {
	directionPath := fmt.Sprintf("/sys/class/gpio/gpio%d/direction", pin)

	if err := os.WriteFile(directionPath, []byte(direction), 0644); err != nil {
		return fmt.Errorf("failed to set pin %d direction to %s: %w", pin, direction, err)
	}

	return nil
}