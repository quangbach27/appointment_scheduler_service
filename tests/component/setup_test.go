package component_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"scheduler/internal"
	appointmentClient "scheduler/internal/appointments/ports/http/client"
	"scheduler/internal/common/config"
	commonHTTP "scheduler/internal/common/http"
	"scheduler/internal/common/log"
	"scheduler/internal/common/testutils"
)

const baseURL = "http://localhost:9090"

func TestMain(m *testing.M) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Init(slog.LevelInfo)

	config := config.NewConfig()
	dbPgx := testutils.NewDB()

	svc, err := internal.New(
		ctx,
		config,
		dbPgx,
		internal.ExternalService{},
	)
	if err != nil {
		panic(err)
	}

	go func() {
		if err := svc.Run(ctx, ":9090"); err != nil {
			panic(err)
		}
	}()

	waitForHttpServerInMain()

	os.Exit(m.Run())
}

func waitForHttpServerInMain() {
	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		resp, err := http.Get(fmt.Sprintf("%s/healthz", baseURL))
		if err == nil && resp.StatusCode < 300 {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}
	panic("HTTP server did not start in time")
}

type testClients struct {
	Appointments *appointmentClient.ClientWithResponses
}

var (
	testClientsOnce sync.Once
	clients         testClients
)

func newTestClients(t *testing.T) testClients {
	testClientsOnce.Do(func() {
		httpClient := &http.Client{Timeout: 10 * time.Second}

		editorFn := func(ctx context.Context, req *http.Request) error {
			log.FromContext(ctx).
				With(
					"method", req.Method,
					"url", req.URL.String(),
					"test_name", t.Name(),
				).
				Info("Making component test API request")
			req.Header.Set(commonHTTP.TestNameHeader, t.Name())
			return nil
		}
		appointmentClients, err := appointmentClient.NewClientWithResponses(
			baseURL,
			appointmentClient.WithHTTPClient(httpClient),
			appointmentClient.WithRequestEditorFn(editorFn),
		)
		if err != nil {
			t.Fatalf("error creating appointments client: %v", err)
		}

		clients = testClients{
			Appointments: appointmentClients,
		}
	})

	return clients
}
