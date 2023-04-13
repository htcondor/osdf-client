package stashcp

import (
	"fmt"
	"os"
	"strings"
)

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

/*
	Environent variable lookup.  Eventually, we should probably
	just use something like viper for this.
*/
// Environment variable lookup
var env_prefixes = [...]string{"OSG", "OSDF"}

// LookupBool returns true if the environment variable is set
func EnvLookupExists(name string) bool {
	for _, prefix := range env_prefixes {
		if _, isSet := os.LookupEnv(prefix + "_" + name); isSet {
			return true
		}
	}
	return false
}

// LookupBool returns true if the environment variable is set to "true"
func EnvLookupBool(name string) bool {
	for _, prefix := range env_prefixes {
		if val, isSet := os.LookupEnv(prefix + "_" + name); isSet {
			return strings.ToLower(val) == "true"
		}
	}
	return false
}

// LookupString returns the value of the environment variable
func EnvLookupString(name string) string {
	for _, prefix := range env_prefixes {
		if val, isSet := os.LookupEnv(prefix + "_" + name); isSet {
			return val
		}
	}
	return ""
}
