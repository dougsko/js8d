package verbose

import "log"

var enabled bool

// SetEnabled sets the global verbose logging flag
func SetEnabled(enable bool) {
	enabled = enable
}

// IsEnabled returns whether verbose logging is enabled
func IsEnabled() bool {
	return enabled
}

// Printf prints a verbose log message if verbose logging is enabled
func Printf(format string, args ...interface{}) {
	if enabled {
		log.Printf("[VERBOSE] "+format, args...)
	}
}

// Print prints a verbose log message if verbose logging is enabled
func Print(args ...interface{}) {
	if enabled {
		log.Print(append([]interface{}{"[VERBOSE] "}, args...)...)
	}
}

// Println prints a verbose log message if verbose logging is enabled
func Println(args ...interface{}) {
	if enabled {
		log.Println(append([]interface{}{"[VERBOSE]"}, args...)...)
	}
}