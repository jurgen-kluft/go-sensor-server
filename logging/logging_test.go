package logging

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLoggerLevelAndOwnedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.log")
	logger, err := New("info", path)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	logger.LogDebug("hidden")
	logger.LogInfo("visible")
	logger.SetLevel(LevelDebug)
	logger.LogDebug("debug-visible")
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	output := string(data)
	if strings.Contains(output, "hidden") || !strings.Contains(output, "visible") || !strings.Contains(output, "debug-visible") {
		t.Fatalf("unexpected log output %q", output)
	}
}

func TestRateLimiterIsConcurrentAndKeyed(t *testing.T) {
	now := time.Unix(100, 0)
	limiter, err := NewRateLimiter(time.Minute, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewRateLimiter() error = %v", err)
	}
	var allowed int
	var mu sync.Mutex
	var waitGroup sync.WaitGroup
	for index := 0; index < 100; index++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if limiter.Allow("malformed") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	waitGroup.Wait()
	if allowed != 1 || !limiter.Allow("queue-full") {
		t.Fatalf("allowed = %d, distinct key allowed = %v", allowed, limiter.Allow("queue-full"))
	}
	now = now.Add(time.Minute)
	if !limiter.Allow("malformed") {
		t.Fatal("key was not allowed after interval")
	}
}
