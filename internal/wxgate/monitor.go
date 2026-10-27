package wxgate

import (
	"fmt"
	"sync"
	"time"
)

type APIStats struct {
	AppID     string    `json:"app_id"`
	Endpoint  string    `json:"endpoint"`
	Count     int64     `json:"count"`
	Errors    int64     `json:"errors"`
	LastCall  time.Time `json:"last_call"`
	FirstCall time.Time `json:"first_call"`
}

type GateMonitor struct {
	stats map[string]*APIStats
	mutex sync.RWMutex
}

var (
	defaultMonitor *GateMonitor
	monitorOnce    sync.Once
)

func GetMonitor() *GateMonitor {
	monitorOnce.Do(func() {
		defaultMonitor = &GateMonitor{
			stats: make(map[string]*APIStats),
		}
	})
	return defaultMonitor
}

func (m *GateMonitor) RecordCall(appID, endpoint string, isError bool) {
	key := fmt.Sprintf("%s:%s", appID, endpoint)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	if s, exists := m.stats[key]; exists {
		s.Count++
		s.LastCall = now
		if isError {
			s.Errors++
		}
	} else {
		errCount := int64(0)
		if isError {
			errCount = 1
		}
		m.stats[key] = &APIStats{
			AppID:     appID,
			Endpoint:  endpoint,
			Count:     1,
			Errors:    errCount,
			FirstCall: now,
			LastCall:  now,
		}
	}
}

func (m *GateMonitor) GetStats() []APIStats {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	res := make([]APIStats, 0, len(m.stats))
	for _, v := range m.stats {
		res = append(res, *v)
	}
	return res
}

func (m *GateMonitor) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.stats = make(map[string]*APIStats)
}
