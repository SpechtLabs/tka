package tui

import (
	"sync"

	"github.com/charmbracelet/lipgloss"
)

// Mutex to protect global options
var optionsMutex sync.RWMutex

var globalOptions = DefaultOptions()

// SetGlobalOptions sets global options for all PrettyPrint calls
func SetGlobalOptions(opts ...Option) {
	// Acquire exclusive lock for writing
	optionsMutex.Lock()
	defer optionsMutex.Unlock()

	for _, opt := range opts {
		opt(&globalOptions)
	}
}

// GetGlobalOptionsCopy returns a copy of the current global options
func GetGlobalOptionsCopy() PrintOptions {
	// Acquire read lock
	optionsMutex.RLock()
	defer optionsMutex.RUnlock()

	// Create a deep copy of global options
	options := globalOptions

	// Deep copy maps to avoid shared references
	options.LevelIcons = make(map[PrintLevel]string, len(globalOptions.LevelIcons))
	for k, v := range globalOptions.LevelIcons {
		options.LevelIcons[k] = v
	}

	options.IconStyles = make(map[PrintLevel]lipgloss.Style, len(globalOptions.IconStyles))
	for k, v := range globalOptions.IconStyles {
		options.IconStyles[k] = v
	}

	return options
}
