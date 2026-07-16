package process

import (
	"sync"
	"testing"

	"repwire/internal/domain"
)

// TestExtractEntitiesIsConcurrencySafe guards the alias regex cache.
//
// The worker runs jobs concurrently (WORKER_CONCURRENCY defaults to 8) and every
// handleProcess call lands here, so an unsynchronised cache map is a live data
// race. It shows up as a "concurrent map writes" panic that takes the worker
// down under exactly the load that matters, and never once in a quiet local run.
// Run with -race for this to mean anything.
func TestExtractEntitiesIsConcurrencySafe(t *testing.T) {
	entities := sportsEntities()
	titles := []string{
		"Man Utd sack their manager",
		"Quỷ đỏ sa thải huấn luyện viên",
		"Messi và Real Madrid",
		"Premier League: Man City thắng đậm",
		"Nguyễn Quang Hải trở lại tuyển Việt Nam",
	}

	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			item := &domain.ContentItem{Title: titles[i%len(titles)]}
			if got := ExtractEntities(item, "", entities); len(got) == 0 {
				t.Errorf("no entities for %q", item.Title)
			}
		}(i)
	}
	wg.Wait()
}
