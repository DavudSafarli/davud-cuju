package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	// Create storage and scorer
	storage := NewInMemStorage(1 * time.Second) // Refresh leaderboard every second
	scorer := NewWeightBasedScorer(map[Skill]int{
		SkillDribble: 1,
		SkillShoot:   2,
		SkillPass:    3,
	})
	service := NewService(storage, scorer)

	// Start the background job to process score events
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := service.ProcessScoreEvents(ctx, 100); err != nil {
			log.Printf("Error processing score events: %v", err)
		}
		log.Println("ProcessScoreEvents stopped")
	}()

	handler := NewHTTPHandler(service)
	mux := handler.SetupRoutes()

	// Setup metrics server
	metricsHandler := GetGlobalMetrics().SetupRoutes()
	metricsServer := &http.Server{
		Addr:    ":9090",
		Handler: metricsHandler,
	}

	// todo: read from env variable
	port := 8080
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Start main API server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	// Start metrics server
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	fmt.Println("Cuju app started. You can try using ./test.sh to populate events and see the leaderboard in action")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()

	log.Println("Shutting down")

	// Shutdown both servers
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown API server: %v", err)
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Failed to shutdown metrics server: %v", err)
	}

	wg.Wait()
	log.Println("Exiting...")
}
