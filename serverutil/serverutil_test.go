package serverutil_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/aosanya/CodeValdSharedLib/serverutil"
)

func TestNewGRPCServer_NotNil(t *testing.T) {
	srv, healthSrv := serverutil.NewGRPCServer()
	if srv == nil {
		t.Fatal("NewGRPCServer: expected non-nil *grpc.Server")
	}
	if healthSrv == nil {
		t.Fatal("NewGRPCServer: expected non-nil *health.Server")
	}
	srv.Stop()
}

func TestRunWithGracefulShutdown_ExitsOnCancel(t *testing.T) {
	srv, _ := serverutil.NewGRPCServer()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		serverutil.RunWithGracefulShutdown(ctx, srv, lis, 5*time.Second)
	}()

	cancel()
	select {
	case <-done:
		// expected — function returned after ctx cancellation
	case <-time.After(10 * time.Second):
		t.Fatal("RunWithGracefulShutdown did not return within 10 s after cancel")
	}
}

func TestEnvOrDefault_UseDefault(t *testing.T) {
	os.Unsetenv("_SUTIL_TEST_ABSENT")
	got := serverutil.EnvOrDefault("_SUTIL_TEST_ABSENT", "fallback")
	if got != "fallback" {
		t.Errorf("EnvOrDefault: got %q, want %q", got, "fallback")
	}
}

func TestEnvOrDefault_UseEnv(t *testing.T) {
	os.Setenv("_SUTIL_TEST_SET", "value")
	defer os.Unsetenv("_SUTIL_TEST_SET")
	got := serverutil.EnvOrDefault("_SUTIL_TEST_SET", "fallback")
	if got != "value" {
		t.Errorf("EnvOrDefault: got %q, want %q", got, "value")
	}
}

func TestParseDurationSeconds_Valid(t *testing.T) {
	os.Setenv("_SUTIL_DUR_S", "30")
	defer os.Unsetenv("_SUTIL_DUR_S")
	got := serverutil.ParseDurationSeconds("_SUTIL_DUR_S", 5*time.Second)
	if got != 30*time.Second {
		t.Errorf("ParseDurationSeconds: got %v, want 30s", got)
	}
}

func TestParseDurationSeconds_Invalid(t *testing.T) {
	os.Setenv("_SUTIL_DUR_S_BAD", "notanint")
	defer os.Unsetenv("_SUTIL_DUR_S_BAD")
	got := serverutil.ParseDurationSeconds("_SUTIL_DUR_S_BAD", 5*time.Second)
	if got != 5*time.Second {
		t.Errorf("ParseDurationSeconds (invalid): got %v, want 5s", got)
	}
}

func TestParseDurationSeconds_Zero(t *testing.T) {
	os.Setenv("_SUTIL_DUR_S_ZERO", "0")
	defer os.Unsetenv("_SUTIL_DUR_S_ZERO")
	got := serverutil.ParseDurationSeconds("_SUTIL_DUR_S_ZERO", 7*time.Second)
	if got != 7*time.Second {
		t.Errorf("ParseDurationSeconds (zero): got %v, want 7s", got)
	}
}

func TestParseDurationSeconds_Absent(t *testing.T) {
	os.Unsetenv("_SUTIL_DUR_S_ABSENT")
	got := serverutil.ParseDurationSeconds("_SUTIL_DUR_S_ABSENT", 10*time.Second)
	if got != 10*time.Second {
		t.Errorf("ParseDurationSeconds (absent): got %v, want 10s", got)
	}
}

func TestParseDurationString_Valid(t *testing.T) {
	os.Setenv("_SUTIL_DUR_STR", "1m30s")
	defer os.Unsetenv("_SUTIL_DUR_STR")
	got := serverutil.ParseDurationString("_SUTIL_DUR_STR", 5*time.Second)
	if got != 90*time.Second {
		t.Errorf("ParseDurationString: got %v, want 1m30s", got)
	}
}

func TestParseDurationString_Invalid(t *testing.T) {
	os.Setenv("_SUTIL_DUR_STR_BAD", "not-a-duration")
	defer os.Unsetenv("_SUTIL_DUR_STR_BAD")
	got := serverutil.ParseDurationString("_SUTIL_DUR_STR_BAD", 5*time.Second)
	if got != 5*time.Second {
		t.Errorf("ParseDurationString (invalid): got %v, want 5s", got)
	}
}

func TestParseDurationString_Absent(t *testing.T) {
	os.Unsetenv("_SUTIL_DUR_STR_ABSENT")
	got := serverutil.ParseDurationString("_SUTIL_DUR_STR_ABSENT", 10*time.Second)
	if got != 10*time.Second {
		t.Errorf("ParseDurationString (absent): got %v, want 10s", got)
	}
}
