package stashcp

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteCountSI(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1 B", ByteCountSI(1))
	assert.Equal(t, "1.0 kB", ByteCountSI(1024))
	assert.Equal(t, "1.3 MB", ByteCountSI(1024*1024+1024*200))

}

func TestEnvLookupExists(t *testing.T) {
	assert.Equal(t, false, EnvLookupExists("TEST"))
	os.Setenv("OSG_TEST", "1")
	assert.Equal(t, true, EnvLookupExists("TEST"))
	os.Unsetenv("OSG_TEST")
	os.Setenv("OSDF_TEST", "1")
	assert.Equal(t, true, EnvLookupExists("TEST"))
	os.Unsetenv("OSDF_TEST")
}

func TestEnvLookupString(t *testing.T) {
	assert.Equal(t, "", EnvLookupString("TEST"))
	os.Setenv("OSG_TEST", "1")
	assert.Equal(t, "1", EnvLookupString("TEST"))
	os.Unsetenv("OSG_TEST")
	os.Setenv("OSDF_TEST", "2")
	assert.Equal(t, "2", EnvLookupString("TEST"))
	os.Unsetenv("OSDF_TEST")
}

func TestEnvLookupBool(t *testing.T) {
	assert.Equal(t, false, EnvLookupBool("TEST"))
	os.Setenv("OSG_TEST", "True")
	assert.Equal(t, true, EnvLookupBool("TEST"))
	os.Unsetenv("OSG_TEST")
	os.Setenv("OSDF_TEST", "False")
	assert.Equal(t, false, EnvLookupBool("TEST"))
	os.Unsetenv("OSDF_TEST")
}
