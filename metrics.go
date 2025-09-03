package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	ScoreEventsTotal      uint64
	ScoreEventsDuplicates uint64
)

type MetricsServer struct{}

func NewMetricsServer() *MetricsServer {
	return &MetricsServer{}
}

// ServeHTTP implements the http.Handler interface
func (m *MetricsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	total := atomic.LoadUint64(&ScoreEventsTotal)
	duplicates := atomic.LoadUint64(&ScoreEventsDuplicates)
	timestamp := time.Now().UnixMilli()

	fmt.Fprintf(w, "score_events_total %d %d\n", total, timestamp)
	fmt.Fprintf(w, "score_events_duplicate %d %d\n", duplicates, timestamp)
}

func (m *MetricsServer) SetupRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", m)
	return mux
}

func IncScoreEventsTotal() {
	atomic.AddUint64(&ScoreEventsTotal, 1)
}

func IncScoreEventsDuplicate() {
	atomic.AddUint64(&ScoreEventsDuplicates, 1)
}

func GetGlobalMetrics() *MetricsServer {
	return &MetricsServer{}
}
