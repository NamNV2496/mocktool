package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"
)

type APIStat struct {
	Feature    string
	Scenario   string
	Path       string
	Method     string
	Hits       int64
	CacheHits  int64
	TotalLatMs float64
}

type StatsStore struct {
	mu     sync.RWMutex
	data   map[string]*APIStat
	cancel context.CancelFunc
}

func NewStatsStore() *StatsStore {
	return &StatsStore{
		data: make(map[string]*APIStat),
	}
}

func (s *StatsStore) Record(feature, scenario, path, method string, cacheHit bool, latencyMs float64) {
	if s == nil {
		return
	}
	key := fmt.Sprintf("%s:%s:%s:%s", feature, scenario, method, path)
	s.mu.Lock()
	e, ok := s.data[key]
	if !ok {
		e = &APIStat{Feature: feature, Scenario: scenario, Path: path, Method: method}
		s.data[key] = e
	}
	e.Hits++
	if cacheHit {
		e.CacheHits++
	}
	e.TotalLatMs += latencyMs
	s.mu.Unlock()
}

// Snapshot returns a copy of all stats sorted by hits descending.
func (s *StatsStore) Snapshot() []APIStat {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]APIStat, 0, len(s.data))
	for _, e := range s.data {
		out = append(out, *e)
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		return out[i].Hits > out[j].Hits
	})
	return out
}

// Reset clears all statistics data.
func (s *StatsStore) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.data = make(map[string]*APIStat)
	s.mu.Unlock()
	slog.Info("Stats reset")
}

// StartResetWorker starts a goroutine that resets stats every hour.
func (s *StatsStore) StartResetWorker(ctx context.Context) {
	if s == nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		slog.Info("Stats reset worker started")

		for {
			select {
			case <-ticker.C:
				s.Reset()
			case <-workerCtx.Done():
				slog.Info("Stats reset worker stopped")
				return
			}
		}
	}()
}

// StopResetWorker stops the reset worker.
func (s *StatsStore) StopResetWorker() {
	if s == nil || s.cancel == nil {
		return
	}
	s.cancel()
}
