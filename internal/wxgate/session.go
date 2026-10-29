package wxgate

import (
	"crypto/rand"
	"encoding/hex"
		"sync"
	"time"
)

type SessionData struct {
	AppID               string        `json:"app_id"`
	OriginalRedirectURI string        `json:"original_redirect_uri"`
	OriginalState       string        `json:"original_state"`
	Scope               string        `json:"scope"`
	UserIP              string        `json:"user_ip"`
	CreatedAt           time.Time     `json:"created_at"`
	Completed           bool          `json:"completed"`
	CompletedAt         *time.Time    `json:"completed_at,omitempty"`
	UserInfo            *UserInfoSafe `json:"user_info,omitempty"`
	Code                string        `json:"code,omitempty"`
	Error               string        `json:"error,omitempty"`
}

type GateSessionManager struct {
	sessions    map[string]*SessionData
	totalCount  int64
	mutex       sync.RWMutex
}

var (
	defaultSessionManager *GateSessionManager
	sessionOnce           sync.Once
)

func GetSessionManager() *GateSessionManager {
	sessionOnce.Do(func() {
		sm := &GateSessionManager{
			sessions: make(map[string]*SessionData),
		}
		go sm.cleanupExpired()
		defaultSessionManager = sm
	})
	return defaultSessionManager
}

func (sm *GateSessionManager) CreateSession(appID, originalRedirectURI, originalState, scope, userIP string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	sessionID := hex.EncodeToString(bytes)

	now := time.Now()
	data := &SessionData{
		AppID:               appID,
		OriginalRedirectURI: originalRedirectURI,
		OriginalState:       originalState,
		Scope:               scope,
		UserIP:              userIP,
		CreatedAt:           now,
	}

	sm.mutex.Lock()
	sm.sessions[sessionID] = data
	sm.totalCount++
	sm.mutex.Unlock()

	return sessionID, nil
}

func (sm *GateSessionManager) GetSession(sessionID string) (*SessionData, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// 24小时有效
	if time.Since(session.CreatedAt) > 24*time.Hour {
		go sm.DeleteSession(sessionID)
		return nil, false
	}

	return session, true
}

func (sm *GateSessionManager) CompleteSession(sessionID string, code string, user *UserInfoSafe, authErr error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return
	}
	now := time.Now()
	session.Completed = true
	session.CompletedAt = &now
	session.Code = code
	session.UserInfo = user
	if authErr != nil {
		session.Error = authErr.Error()
	}
}

func (sm *GateSessionManager) DeleteSession(sessionID string) {
	sm.mutex.Lock()
	delete(sm.sessions, sessionID)
	sm.mutex.Unlock()
}

func (sm *GateSessionManager) cleanupExpired() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.mutex.Lock()
		now := time.Now()
		for id, s := range sm.sessions {
			if now.Sub(s.CreatedAt) > 24*time.Hour {
				delete(sm.sessions, id)
			}
		}
		sm.mutex.Unlock()
	}
}

func (sm *GateSessionManager) GetStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return map[string]interface{}{
		"active_sessions": len(sm.sessions),
		"total_transfers": sm.totalCount,
	}
}
