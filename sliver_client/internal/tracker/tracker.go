package tracker

import (
	"context"
	"sync"
	"time"

	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/bishopfox/sliver/protobuf/rpcpb"
)

// TaskState represents the lifecycle of a beacon task.
type TaskState string

const (
	TaskPending   TaskState = "pending"
	TaskSent      TaskState = "sent"
	TaskCompleted TaskState = "completed"
	TaskFailed    TaskState = "failed"
)

// TaskInfo holds metadata and results for a tracked beacon task.
type TaskInfo struct {
	TaskID      string
	BeaconID    string
	Description string
	State       TaskState
	SubmittedAt time.Time
	CompletedAt time.Time
	Output      string
}

// Tracker polls the Sliver server for beacon task status updates.
type Tracker struct {
	mu     sync.RWMutex
	rpc    rpcpb.SliverRPCClient
	tasks  map[string]*TaskInfo
	stopCh chan struct{}
}

func New(rpc rpcpb.SliverRPCClient) *Tracker {
	return &Tracker{
		rpc:    rpc,
		tasks:  make(map[string]*TaskInfo),
		stopCh: make(chan struct{}),
	}
}

// Register adds a new task to track.
func (t *Tracker) Register(taskID, beaconID, description string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tasks[taskID] = &TaskInfo{
		TaskID:      taskID,
		BeaconID:    beaconID,
		Description: description,
		State:       TaskPending,
		SubmittedAt: time.Now(),
	}
}

// Start begins background polling. Run as a goroutine.
func (t *Tracker) Start() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.pollAll()
		}
	}
}

func (t *Tracker) Stop() {
	select {
	case <-t.stopCh:
	default:
		close(t.stopCh)
	}
}

// CheckAll performs an on-demand poll of all pending/sent tasks. Returns updated tasks.
func (t *Tracker) CheckAll() []TaskInfo {
	t.pollAll()
	return t.PendingTasks()
}

// PendingTasks returns all tasks that are not yet completed.
func (t *Tracker) PendingTasks() []TaskInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var out []TaskInfo
	for _, ti := range t.tasks {
		if ti.State == TaskPending || ti.State == TaskSent {
			out = append(out, *ti)
		}
	}
	return out
}

// AllTasks returns a snapshot of all tracked tasks.
func (t *Tracker) AllTasks() []TaskInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]TaskInfo, 0, len(t.tasks))
	for _, ti := range t.tasks {
		out = append(out, *ti)
	}
	return out
}

// ClearCompleted removes all completed/failed tasks from tracking.
func (t *Tracker) ClearCompleted() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, ti := range t.tasks {
		if ti.State == TaskCompleted || ti.State == TaskFailed {
			delete(t.tasks, id)
		}
	}
}

// CompletedCount returns how many tasks are complete or failed.
func (t *Tracker) CompletedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	count := 0
	for _, ti := range t.tasks {
		if ti.State == TaskCompleted || ti.State == TaskFailed {
			count++
		}
	}
	return count
}

// TotalCount returns total tracked tasks.
func (t *Tracker) TotalCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.tasks)
}

func (t *Tracker) pollAll() {
	t.mu.RLock()
	// Collect beacon IDs that have pending tasks
	beaconIDs := make(map[string]struct{})
	for _, ti := range t.tasks {
		if ti.State == TaskPending || ti.State == TaskSent {
			beaconIDs[ti.BeaconID] = struct{}{}
		}
	}
	t.mu.RUnlock()

	if len(beaconIDs) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for beaconID := range beaconIDs {
		tasksResp, err := t.rpc.GetBeaconTasks(ctx, &clientpb.Beacon{ID: beaconID})
		if err != nil {
			continue
		}
		if tasksResp == nil {
			continue
		}

		t.mu.Lock()
		for _, bt := range tasksResp.Tasks {
			ti, exists := t.tasks[bt.ID]
			if !exists {
				continue
			}

			switch bt.State {
			case "completed":
				ti.State = TaskCompleted
				ti.CompletedAt = time.Unix(bt.CompletedAt, 0)
				// Try to fetch the task content for output
				t.mu.Unlock()
				content, err := t.rpc.GetBeaconTaskContent(ctx, &clientpb.BeaconTask{ID: bt.ID})
				t.mu.Lock()
				if err == nil && content != nil {
					ti.Output = string(content.Response)
				}
			case "sent":
				if ti.State == TaskPending {
					ti.State = TaskSent
				}
			case "canceled":
				ti.State = TaskFailed
				ti.Output = "task was canceled"
			}
		}
		t.mu.Unlock()
	}
}
