package main

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient implements RiverClient for testing
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Stop(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestShutdownHandler(t *testing.T) {
	// Create a mock client
	mockClient := new(MockClient)
	mockClient.On("Stop", mock.Anything).Return(nil)

	// Create a channel to track when the function returns
	done := make(chan struct{})

	// Run shutdownHandler in a goroutine
	go func() {
		shutdownHandler(context.Background(), mockClient)
		close(done)
	}()

	// Verify the function hasn't returned after a short delay
	select {
	case <-done:
		t.Fatal("shutdownHandler returned before receiving signal")
	case <-time.After(100 * time.Millisecond):
		// This is expected - handler should still be running
	}

	// Send shutdown signal
	process, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	assert.NoError(t, process.Signal(syscall.SIGTERM))

	// Verify the function returns after signal
	select {
	case <-done:
		// Success - handler stopped after signal
	case <-time.After(1 * time.Second):
		t.Fatal("shutdownHandler didn't return after signal")
	}

	// Verify the client's Stop method was called
	mockClient.AssertExpectations(t)
}
