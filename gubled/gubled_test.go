package gubled

import (
	"github.com/stretchr/testify/assert"

	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestCreateKVStoreBackend(t *testing.T) {
	a := assert.New(t)

	memory := CreateKVStoreBackend(Args{KVBackend: "memory"})
	a.Equal("*store.MemoryKVStore", reflect.TypeOf(memory).String())

	dir, _ := ioutil.TempDir("", "guble_test")
	defer os.RemoveAll(dir)
	sqlite := CreateKVStoreBackend(Args{KVBackend: "file", StoragePath: dir})
	a.Equal("*store.SqliteKVStore", reflect.TypeOf(sqlite).String())
}

func TestParsingOfEnviromentVariables(t *testing.T) {
	a := assert.New(t)

	originalArgs := os.Args
	os.Args = []string{os.Args[0]}
	defer func() { os.Args = originalArgs }()

	// given: some environment variables
	os.Setenv("GUBLE_LISTEN", "listen")
	defer os.Unsetenv("GUBLE_LISTEN")

	os.Setenv("GUBLE_LOG_INFO", "true")
	defer os.Unsetenv("GUBLE_LOG_INFO")

	os.Setenv("GUBLE_LOG_DEBUG", "true")
	defer os.Unsetenv("GUBLE_LOG_DEBUG")

	os.Setenv("GUBLE_KV_BACKEND", "kv-backend")
	defer os.Unsetenv("GUBLE_KV_BACKEND")

	os.Setenv("GUBLE_STORAGE_PATH", "storage-path")
	defer os.Unsetenv("GUBLE_STORAGE_PATH")

	os.Setenv("GUBLE_MS_BACKEND", "ms-backend")
	defer os.Unsetenv("GUBLE_MS_BACKEND")

	os.Setenv("GUBLE_GCM_API_KEY", "gcm-api-key")
	defer os.Unsetenv("GUBLE_GCM_API_KEY")

	os.Setenv("GUBLE_GCM_ENABLE", "true")
	defer os.Unsetenv("GUBLE_GCM_ENABLE")

	// when we parse the arguments
	args := loadArgs()

	// the the arg parameters are set
	assertArguments(a, args)
}

func TestParsingArgs(t *testing.T) {
	a := assert.New(t)

	originalArgs := os.Args

	defer func() { os.Args = originalArgs }()

	// given: a command line
	os.Args = []string{os.Args[0],
		"--listen", "listen",
		"--log-info",
		"--log-debug",
		"--kv-backend", "kv-backend",
		"--storage-path", "storage-path",
		"--ms-backend", "ms-backend",
		"--gcm-api-key", "gcm-api-key",
		"--gcm-enable"}

	// when we parse the arguments
	args := loadArgs()

	// the the arg parameters are set
	assertArguments(a, args)
}

func assertArguments(a *assert.Assertions, args Args) {
	a.Equal("listen", args.Listen)
	a.Equal(true, args.LogInfo)
	a.Equal(true, args.LogDebug)
	a.Equal("kv-backend", args.KVBackend)
	a.Equal("storage-path", args.StoragePath)
	a.Equal("ms-backend", args.MSBackend)
	a.Equal("gcm-api-key", args.GcmApiKey)
	a.Equal(true, args.GcmEnable)
}

func TestArgDefaultValues(t *testing.T) {
	a := assert.New(t)

	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// given: a command line
	os.Args = []string{os.Args[0]}

	// when we parse the arguments
	args := loadArgs()

	// the we have correct defaults set
	a.Equal(":8080", args.Listen)
	a.Equal(false, args.LogInfo)
	a.Equal(false, args.LogDebug)
	a.Equal("file", args.KVBackend)
	a.Equal("/var/lib/guble", args.StoragePath)
	a.Equal("file", args.MSBackend)
	a.Equal("", args.GcmApiKey)
	a.Equal(false, args.GcmEnable)
}

func TestGcmOnlyStartedIfEnabled(t *testing.T) {
	a := assert.New(t)

	a.True(containsGcmModule(CreateModules(Args{GcmEnable: true, GcmApiKey: "xyz"})))
	a.False(containsGcmModule(CreateModules(Args{GcmEnable: false})))
}

func containsGcmModule(modules []interface{}) bool {
	for _, module := range modules {
		if reflect.TypeOf(module).String() == "*gcm.GCMConnector" {
			return true
		}
	}
	return false
}

func TestPanicOnMissingGcmApiKey(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Log("expect panic, because the gcm api key was not supplied")
			t.Fail()
		}
	}()

	CreateModules(Args{GcmEnable: true})
}

func TestCreateStoreBackendPanicInvalidBackend(t *testing.T) {
	a := assert.New(t)

	var p interface{}
	func() {
		defer func() {
			p = recover()
		}()

		CreateKVStoreBackend(Args{KVBackend: "foo bar"})
	}()
	a.NotNil(p)
}
