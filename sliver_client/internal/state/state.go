package state

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/commonpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// TargetKind distinguishes sessions from beacons.
type TargetKind int

const (
	KindSession TargetKind = iota
	KindBeacon
)

// Target is the unified type for dispatch — either a session or a beacon.
type Target struct {
	Kind     TargetKind
	ID       string // full UUID
	ShortID  string // first segment before '-'
	Hostname string
	Address  string
	OS       string
	Arch     string
	Username string
	PID      int32
	Process  string
	// Beacon-specific
	LastCheckin time.Time
	NextCheckin time.Time
	TasksCount  int32
}

// State holds the live view of all sessions and beacons, protected by a mutex.
type State struct {
	mu       sync.RWMutex
	rpc      rpcpb.SliverRPCClient
	refresh  time.Duration
	stopCh   chan struct{}
	sessions []Target
	beacons  []Target
	selected map[string]struct{} // set of target IDs
}

func New(rpc rpcpb.SliverRPCClient, refresh time.Duration) *State {
	return &State{
		rpc:      rpc,
		refresh:  refresh,
		stopCh:   make(chan struct{}),
		selected: make(map[string]struct{}),
	}
}

// StartRefresh polls the server periodically. Run as a goroutine.
func (s *State) StartRefresh() {
	s.poll() // initial fetch
	ticker := time.NewTicker(s.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.poll()
		}
	}
}

func (s *State) Stop() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
}

func (s *State) poll() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch sessions
	sessResp, err := s.rpc.GetSessions(ctx, &commonpb.Empty{})
	if err == nil && sessResp != nil {
		var sessions []Target
		for _, sess := range sessResp.Sessions {
			if sess.IsDead {
				continue
			}
			sessions = append(sessions, Target{
				Kind:     KindSession,
				ID:       sess.ID,
				ShortID:  shortID(sess.ID),
				Hostname: sess.Hostname,
				Address:  stripPort(sess.RemoteAddress),
				OS:       sess.OS,
				Arch:     sess.Arch,
				Username: sess.Username,
				PID:      sess.PID,
				Process:  sess.Filename,
			})
		}
		s.mu.Lock()
		s.sessions = sessions
		s.mu.Unlock()
	}

	// Fetch beacons
	beaconResp, err := s.rpc.GetBeacons(ctx, &commonpb.Empty{})
	if err == nil && beaconResp != nil {
		var beacons []Target
		for _, b := range beaconResp.Beacons {
			beacons = append(beacons, Target{
				Kind:        KindBeacon,
				ID:          b.ID,
				ShortID:     shortID(b.ID),
				Hostname:    b.Hostname,
				Address:     stripPort(b.RemoteAddress),
				OS:          b.OS,
				Arch:        b.Arch,
				Username:    b.Username,
				PID:         b.PID,
				Process:     b.Filename,
				LastCheckin: time.Unix(b.LastCheckin, 0),
				NextCheckin: time.Unix(b.NextCheckin, 0),
				TasksCount:  int32(b.TasksCountCompleted),
			})
		}
		s.mu.Lock()
		s.beacons = beacons
		s.mu.Unlock()
	}
}

// Sessions returns a snapshot of live sessions.
func (s *State) Sessions() []Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Target, len(s.sessions))
	copy(out, s.sessions)
	return out
}

// Beacons returns a snapshot of live beacons.
func (s *State) Beacons() []Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Target, len(s.beacons))
	copy(out, s.beacons)
	return out
}

// AllTargets returns sessions + beacons combined.
func (s *State) AllTargets() []Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Target, 0, len(s.sessions)+len(s.beacons))
	out = append(out, s.sessions...)
	out = append(out, s.beacons...)
	return out
}

// Selected returns targets whose IDs are in the selected set.
func (s *State) Selected() []Target {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Target
	for _, t := range s.sessions {
		if _, ok := s.selected[t.ID]; ok {
			out = append(out, t)
		}
	}
	for _, t := range s.beacons {
		if _, ok := s.selected[t.ID]; ok {
			out = append(out, t)
		}
	}
	return out
}

// ToggleSelected toggles whether a target ID is in the selected set.
func (s *State) ToggleSelected(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.selected[id]; ok {
		delete(s.selected, id)
	} else {
		s.selected[id] = struct{}{}
	}
}

// IsSelected checks if a target ID is selected.
func (s *State) IsSelected(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.selected[id]
	return ok
}

// SelectAll selects all live sessions and beacons.
func (s *State) SelectAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.sessions {
		s.selected[t.ID] = struct{}{}
	}
	for _, t := range s.beacons {
		s.selected[t.ID] = struct{}{}
	}
}

// SelectNone clears the selection.
func (s *State) SelectNone() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selected = make(map[string]struct{})
}

// SelectByOS selects only targets matching the given OS (e.g. "windows", "linux").
func (s *State) SelectByOS(osName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.selected = make(map[string]struct{})
	for _, t := range s.sessions {
		if strings.EqualFold(t.OS, osName) {
			s.selected[t.ID] = struct{}{}
		}
	}
	for _, t := range s.beacons {
		if strings.EqualFold(t.OS, osName) {
			s.selected[t.ID] = struct{}{}
		}
	}
}

// SelectedCount returns how many targets are selected.
func (s *State) SelectedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.selected)
}

func shortID(id string) string {
	parts := strings.SplitN(id, "-", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return id
}

func stripPort(addr string) string {
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		return addr[:idx]
	}
	return addr
}
