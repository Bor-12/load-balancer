package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestServer_GracefulShutdownLetsActiveRequestFinish(t *testing.T) {
	requestStarted := make(chan struct{})
	finishRequest := make(chan struct{})
	var closeRequestStarted sync.Once

	server := &http.Server{
		Addr: listenerAddress(t),
		Handler: http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			closeRequestStarted.Do(func() {
				close(requestStarted)
			})
			<-finishRequest
			responseWriter.WriteHeader(http.StatusOK)
		}),
	}

	runner := NewRunner(server, testLogger(), time.Second)
	runContext, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	runErrors := make(chan error, 1)
	go func() {
		runErrors <- runner.Run(runContext)
	}()

	waitForServer(t, server.Addr)

	responseErrors := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + server.Addr)
		if err != nil {
			responseErrors <- err
			return
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			responseErrors <- http.ErrBodyNotAllowed
			return
		}

		responseErrors <- nil
	}()

	<-requestStarted
	cancelRun()
	close(finishRequest)

	if err := <-responseErrors; err != nil {
		t.Fatalf("expected active request to finish: %v", err)
	}
	if err := <-runErrors; err != nil {
		t.Fatalf("expected server to stop cleanly: %v", err)
	}
}

func listenerAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	return listener.Addr().String()
}

func waitForServer(t *testing.T, address string) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("server did not start")
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
