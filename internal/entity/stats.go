package entity

type APIStatDTO struct {
	Feature      string  `json:"feature"`
	Scenario     string  `json:"scenario"`
	Path         string  `json:"path"`
	Method       string  `json:"method"`
	Hits         int64   `json:"hits"`
	CacheHits    int64   `json:"cache_hits"`
	CacheHitRate float64 `json:"cache_hit_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

type StatsResponse struct {
	TotalHits    int64        `json:"total_hits"`
	CacheHitRate float64      `json:"cache_hit_rate"`
	AvgLatencyMs float64      `json:"avg_latency_ms"`
	APIs         []APIStatDTO `json:"apis"`
}
