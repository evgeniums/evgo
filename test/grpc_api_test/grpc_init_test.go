package grpc_api

import (
	"context"
	"testing"
	"time"

	"github.com/evgeniums/go-utils/pkg/background_worker"
)

func TestInitServer(t *testing.T) {
	app, _, server := InitServer(t)

	t.Logf("Running server")

	fin := background_worker.NewFinisher()
	go func() {
		server.ApiServer().Run(fin)
	}()

	time.Sleep(time.Second * time.Duration(300))
	t.Logf("Shutdown server")
	fin.Shutdown(context.Background())

	t.Logf("Wait for shutdown")
	fin.Wait()
	t.Logf("Waiting complete")
	t.Logf("Closing app")
	app.Close()
	t.Logf("App closed")
}
