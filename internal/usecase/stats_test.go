package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStatsStore(t *testing.T) {
	s := NewStatsStore()
	assert.NotNil(t, s)
	assert.Empty(t, s.Snapshot())
}

func TestRecord_FirstHit(t *testing.T) {
	s := NewStatsStore()
	s.Record("feat", "scen", "/api/foo", "GET", false, 10.0)
	snap := s.Snapshot()
	assert.Len(t, snap, 1)
	assert.Equal(t, int64(1), snap[0].Hits)
	assert.Equal(t, int64(0), snap[0].CacheHits)
	assert.Equal(t, 10.0, snap[0].TotalLatMs)
}

func TestRecord_CacheHit(t *testing.T) {
	s := NewStatsStore()
	s.Record("feat", "scen", "/api/foo", "GET", true, 5.0)
	s.Record("feat", "scen", "/api/foo", "GET", false, 15.0)
	snap := s.Snapshot()
	assert.Equal(t, int64(2), snap[0].Hits)
	assert.Equal(t, int64(1), snap[0].CacheHits)
	assert.Equal(t, 20.0, snap[0].TotalLatMs)
}

func TestRecord_MultipleAPIs(t *testing.T) {
	s := NewStatsStore()
	s.Record("feat", "scen", "/api/a", "GET", false, 10.0)
	s.Record("feat", "scen", "/api/b", "POST", false, 20.0)
	s.Record("feat", "scen", "/api/a", "GET", true, 5.0)
	snap := s.Snapshot()
	assert.Len(t, snap, 2)
	// sorted by hits desc: /api/a has 2 hits, /api/b has 1
	assert.Equal(t, "/api/a", snap[0].Path)
	assert.Equal(t, int64(2), snap[0].Hits)
}

func TestRecord_NilSafe(t *testing.T) {
	var s *StatsStore
	assert.NotPanics(t, func() { s.Record("f", "s", "/p", "GET", false, 1.0) })
	assert.Nil(t, s.Snapshot())
}

func TestSnapshot_SortedByHitsDesc(t *testing.T) {
	s := NewStatsStore()
	s.Record("f", "s", "/low", "GET", false, 1.0)
	s.Record("f", "s", "/high", "GET", false, 1.0)
	s.Record("f", "s", "/high", "GET", false, 1.0)
	s.Record("f", "s", "/high", "GET", false, 1.0)
	snap := s.Snapshot()
	assert.Equal(t, "/high", snap[0].Path)
	assert.Equal(t, int64(3), snap[0].Hits)
}
