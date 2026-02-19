package metrics

import (
	"context"
	"log/slog"
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"golinks/internal/db"
)

var (
	keywordLookupDesc = prometheus.NewDesc(
		"golinks_keyword_lookups_total",
		"Total keyword lookup count by outcome",
		[]string{"keyword", "outcome"},
		nil,
	)
)

// KeywordCollector is a custom Prometheus collector that reads keyword lookup
// counts from the database on each scrape.
type KeywordCollector struct {
	db *db.DB
}

// Describe sends the metric descriptor to the channel.
func (c *KeywordCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- keywordLookupDesc
}

// Collect queries the database for all keyword lookups and emits them as counters.
func (c *KeywordCollector) Collect(ch chan<- prometheus.Metric) {
	lookups, err := c.db.GetAllKeywordLookups(context.Background())
	if err != nil {
		slog.Error("failed to collect keyword lookup metrics", "error", err)
		return
	}
	for _, l := range lookups {
		ch <- prometheus.MustNewConstMetric(
			keywordLookupDesc,
			prometheus.CounterValue,
			float64(l.Count),
			l.Keyword,
			l.Outcome,
		)
	}
}

// Recorder provides async keyword lookup recording.
type Recorder struct {
	db *db.DB
}

var (
	recorder     *Recorder
	recorderOnce sync.Once
)

// Init registers the custom collector and initializes the recorder.
// Must be called once at startup.
func Init(database *db.DB) {
	recorderOnce.Do(func() {
		recorder = &Recorder{db: database}
		prometheus.MustRegister(&KeywordCollector{db: database})
	})
}

// RecordKeywordLookup asynchronously records a keyword lookup outcome.
func RecordKeywordLookup(keyword, outcome string) {
	if recorder == nil {
		return
	}
	go func() {
		if err := recorder.db.IncrementKeywordLookup(context.Background(), keyword, outcome); err != nil {
			slog.Error("failed to record keyword lookup", "keyword", keyword, "outcome", outcome, "error", err)
		}
	}()
}
