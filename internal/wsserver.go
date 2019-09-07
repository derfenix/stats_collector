package internal

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"gopkg.in/olahol/melody.v1"
)

// StartWSServer creates and starts websocket server with specified handler for incoming messages
func StartWSServer(ctx context.Context, wg *sync.WaitGroup, host string, port uint, endpoint string, handler func(s *melody.Session, msg []byte)) error {
	m := melody.New()

	// Init web-server
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Add handler which upgrades http connection to the ws connection
	e.GET(endpoint, func(c echo.Context) error {
		if err := m.HandleRequest(c.Response(), c.Request()); err != nil {
			return fmt.Errorf("failed to handle ws request: %s", err.Error())
		}
		return nil
	})

	m.HandleMessage(handler)

	// Start server
	go func() {
		defer wg.Done()
		log.Printf("starting ws server at %s:%d", host, port)
		if err := e.Start(fmt.Sprintf("%s:%d", host, port)); err != nil {
			log.Printf("ws server stopped with error: %s", err.Error())
		} else {
			log.Println("ws server stopped without errors")
		}
	}()

	// Wait for context cancellation and graceful shutdown the server
	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*2)
			defer cancel()

			_ = e.Server.Shutdown(shutdownCtx)
		}
	}()

	return nil
}
