package docker

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
)

// Sample represents CPU and memory stats for a single scrape interval
type Sample struct {
	Timestamp   time.Time
	CPUPercent  float64
	MemoryBytes uint64
}

// ScraperSettings configures the docker stats scraper
type ScraperSettings struct {
	BufferSize int
}

func (s ScraperSettings) withDefaults() ScraperSettings {
	if s.BufferSize == 0 {
		s.BufferSize = 200
	}
	return s
}

// Scraper periodically scrapes Docker container stats
type Scraper struct {
	settings  ScraperSettings
	namespace *Namespace

	mu        sync.RWMutex
	apps      map[string]*appData
	lastError error
}

type appData struct {
	samples []Sample
	head    int
	count   int
}

func NewScraper(ns *Namespace, settings ScraperSettings) *Scraper {
	settings = settings.withDefaults()
	return &Scraper{
		settings:  settings,
		namespace: ns,
		apps:      make(map[string]*appData),
	}
}

// Fetch returns the last n samples for an app, ordered from newest to oldest.
// If fewer than n samples exist, only the available samples are returned.
func (s *Scraper) Fetch(appName string, n int) []Sample {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.apps[appName]
	if !ok {
		return nil
	}

	available := min(n, data.count)
	result := make([]Sample, available)
	for i := range available {
		idx := (data.head - 1 - i + len(data.samples)) % len(data.samples)
		result[i] = data.samples[idx]
	}

	return result
}

func (s *Scraper) LastError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastError
}

func (s *Scraper) Scrape(ctx context.Context) {
	containers, err := s.findAppContainers(ctx)
	if err != nil {
		s.setError(err)
		return
	}

	now := time.Now()
	for appName, containerID := range containers {
		stats, err := s.getContainerStats(ctx, containerID)
		if err != nil {
			continue
		}

		sample := Sample{
			Timestamp:   now,
			CPUPercent:  calculateCPUPercent(stats),
			MemoryBytes: stats.MemoryStats.Usage,
		}
		s.recordSample(appName, sample)
	}

	s.setError(nil)
}

func (s *Scraper) findAppContainers(ctx context.Context) (map[string]string, error) {
	containers, err := s.namespace.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, err
	}

	prefix := s.namespace.name + "-app-"
	result := make(map[string]string)

	for _, c := range containers {
		if c.State != "running" {
			continue
		}
		for _, name := range c.Names {
			name = strings.TrimPrefix(name, "/")
			if strings.HasPrefix(name, prefix) {
				remainder := strings.TrimPrefix(name, prefix)
				lastDash := strings.LastIndex(remainder, "-")
				if lastDash > 0 {
					appName := remainder[:lastDash]
					result[appName] = c.ID
				}
			}
		}
	}

	return result, nil
}

func (s *Scraper) getContainerStats(ctx context.Context, containerID string) (*container.StatsResponse, error) {
	resp, err := s.namespace.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

func (s *Scraper) recordSample(appName string, sample Sample) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, ok := s.apps[appName]
	if !ok {
		data = &appData{
			samples: make([]Sample, s.settings.BufferSize),
		}
		s.apps[appName] = data
	}

	data.samples[data.head] = sample
	data.head = (data.head + 1) % len(data.samples)
	if data.count < len(data.samples) {
		data.count++
	}
}

func (s *Scraper) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastError = err
}

// Helpers

func calculateCPUPercent(stats *container.StatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCPUs) * 100.0
	}
	return 0.0
}
