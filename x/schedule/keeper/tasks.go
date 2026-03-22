package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/schedule/types"
)

// Task type constants.
const (
	TaskTypeGuardianMaintenance uint32 = 1
	TaskTypeCreditRenewal       uint32 = 2
	TaskTypeCustom              uint32 = 3
)

// ScheduledTask represents a task registered by an agent home.
type ScheduledTask struct {
	TaskID         string `json:"task_id"`
	HomeID         string `json:"home_id"`
	TaskType       uint32 `json:"task_type"`
	ScheduledHeight uint64 `json:"scheduled_height"`
	IntervalBlocks uint64 `json:"interval_blocks"` // 0 = one-shot, >0 = recurring
	Payload        []byte `json:"payload,omitempty"`
	CreatedAt      uint64 `json:"created_at"`
}

// ---------- Key Construction ----------

// taskKey returns the primary store key: TaskKeyPrefix + homeID + "/" + taskID
func taskKey(homeID, taskID string) []byte {
	return append(types.TaskKeyPrefix, []byte(homeID+"/"+taskID)...)
}

// taskByHomePrefix returns the prefix for iterating all tasks of a home.
func taskByHomePrefix(homeID string) []byte {
	return append(types.TaskKeyPrefix, []byte(homeID+"/")...)
}

// taskHeightIndexKey returns the height-based index key for due-task lookup.
// Format: TaskByHeightIndexPrefix + uint64BE(height) + "/" + homeID + "/" + taskID
func taskHeightIndexKey(height uint64, homeID, taskID string) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height)
	key := append(types.TaskByHeightIndexPrefix, buf...)
	key = append(key, []byte("/"+homeID+"/"+taskID)...)
	return key
}

// taskHeightEndKey returns the exclusive upper bound for iterating tasks due at or before height.
func taskHeightEndKey(height uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, height+1)
	return append(types.TaskByHeightIndexPrefix, buf...)
}

// ---------- RegisterTask ----------

// RegisterTask registers a new scheduled task for a home.
// Returns the generated taskID.
func (k Keeper) RegisterTask(
	ctx sdk.Context,
	homeID string,
	taskType uint32,
	scheduledHeight uint64,
	intervalBlocks uint64,
	payload []byte,
) (string, error) {
	if homeID == "" {
		return "", fmt.Errorf("homeID cannot be empty")
	}
	if scheduledHeight == 0 {
		return "", fmt.Errorf("scheduledHeight must be positive")
	}

	// Generate deterministic taskID: homeID[:8] + hex(height)
	prefix := homeID
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}
	taskID := fmt.Sprintf("%s-%x", prefix, scheduledHeight)

	// Ensure uniqueness — if collision, append sequence
	store := k.storeService.OpenKVStore(ctx)
	key := taskKey(homeID, taskID)
	existing, err := store.Get(key)
	if err == nil && existing != nil {
		seq := k.NextSequence(ctx)
		taskID = fmt.Sprintf("%s-%x-%d", prefix, scheduledHeight, seq)
		key = taskKey(homeID, taskID)
	}

	task := ScheduledTask{
		TaskID:          taskID,
		HomeID:          homeID,
		TaskType:        taskType,
		ScheduledHeight: scheduledHeight,
		IntervalBlocks:  intervalBlocks,
		Payload:         payload,
		CreatedAt:       uint64(ctx.BlockHeight()),
	}

	bz, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}

	if err := store.Set(key, bz); err != nil {
		return "", fmt.Errorf("failed to store task: %w", err)
	}

	// Height index
	idxKey := taskHeightIndexKey(scheduledHeight, homeID, taskID)
	if err := store.Set(idxKey, []byte(taskID)); err != nil {
		return "", fmt.Errorf("failed to store height index: %w", err)
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("task_registered",
			sdk.NewAttribute("home_id", homeID),
			sdk.NewAttribute("task_id", taskID),
			sdk.NewAttribute("task_type", fmt.Sprintf("%d", taskType)),
			sdk.NewAttribute("scheduled_height", fmt.Sprintf("%d", scheduledHeight)),
		),
	})

	return taskID, nil
}

// ---------- GetTask ----------

// GetTask retrieves a task by homeID and taskID.
func (k Keeper) GetTask(ctx sdk.Context, homeID, taskID string) (*ScheduledTask, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(taskKey(homeID, taskID))
	if err != nil || bz == nil {
		return nil, false
	}
	var task ScheduledTask
	if err := json.Unmarshal(bz, &task); err != nil {
		return nil, false
	}
	return &task, true
}

// ---------- ListTasks ----------

// ListTasks returns all tasks for a given homeID.
func (k Keeper) ListTasks(ctx sdk.Context, homeID string) []*ScheduledTask {
	store := k.storeService.OpenKVStore(ctx)
	prefix := taskByHomePrefix(homeID)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var tasks []*ScheduledTask
	for ; iter.Valid(); iter.Next() {
		var task ScheduledTask
		if err := json.Unmarshal(iter.Value(), &task); err != nil {
			continue
		}
		tasks = append(tasks, &task)
	}
	return tasks
}

// ---------- CancelTask ----------

// CancelTask removes a task. Returns false if not found.
func (k Keeper) CancelTask(ctx sdk.Context, homeID, taskID string) bool {
	task, found := k.GetTask(ctx, homeID, taskID)
	if !found {
		return false
	}

	store := k.storeService.OpenKVStore(ctx)

	// Remove primary key
	_ = store.Delete(taskKey(homeID, taskID))

	// Remove height index
	_ = store.Delete(taskHeightIndexKey(task.ScheduledHeight, homeID, taskID))

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent("task_cancelled",
			sdk.NewAttribute("home_id", homeID),
			sdk.NewAttribute("task_id", taskID),
		),
	})

	return true
}

// ---------- ProcessDueTasks ----------

// ProcessDueTasks finds and returns all tasks due at or before currentHeight.
// One-shot tasks (IntervalBlocks == 0) are deleted after collection.
// Recurring tasks are rescheduled to currentHeight + IntervalBlocks.
func (k Keeper) ProcessDueTasks(ctx sdk.Context, currentHeight uint64) []ScheduledTask {
	store := k.storeService.OpenKVStore(ctx)

	iter, err := store.Iterator(types.TaskByHeightIndexPrefix, taskHeightEndKey(currentHeight))
	if err != nil {
		return nil
	}

	// Collect index entries first to avoid mutating store during iteration.
	type indexEntry struct {
		idxKey []byte
		homeID string
		taskID string
	}
	var entries []indexEntry
	for ; iter.Valid(); iter.Next() {
		// Parse homeID and taskID from the index key.
		// Key format: prefix(1) + height(8) + "/" + homeID + "/" + taskID
		raw := iter.Key()
		prefixLen := len(types.TaskByHeightIndexPrefix)
		if len(raw) <= prefixLen+8+1 { // prefix + height + at least "/"
			continue
		}
		remainder := string(raw[prefixLen+8+1:]) // skip prefix + height + "/"
		// remainder = homeID + "/" + taskID
		var hID, tID string
		for i := 0; i < len(remainder); i++ {
			if remainder[i] == '/' {
				hID = remainder[:i]
				tID = remainder[i+1:]
				break
			}
		}
		if hID == "" || tID == "" {
			continue
		}
		idxKeyCopy := make([]byte, len(raw))
		copy(idxKeyCopy, raw)
		entries = append(entries, indexEntry{idxKey: idxKeyCopy, homeID: hID, taskID: tID})
	}
	iter.Close()

	var dueTasks []ScheduledTask
	for _, e := range entries {
		task, found := k.GetTask(ctx, e.homeID, e.taskID)
		if !found {
			// Stale index — clean up
			_ = store.Delete(e.idxKey)
			continue
		}

		dueTasks = append(dueTasks, *task)

		// Remove old height index
		_ = store.Delete(e.idxKey)

		if task.IntervalBlocks == 0 {
			// One-shot: delete the task
			_ = store.Delete(taskKey(e.homeID, e.taskID))
		} else {
			// Recurring: reschedule
			task.ScheduledHeight = currentHeight + task.IntervalBlocks
			bz, err := json.Marshal(task)
			if err != nil {
				continue
			}
			_ = store.Set(taskKey(e.homeID, e.taskID), bz)
			_ = store.Set(taskHeightIndexKey(task.ScheduledHeight, e.homeID, e.taskID), []byte(e.taskID))
		}
	}

	if len(dueTasks) > 0 {
		ctx.EventManager().EmitEvents(sdk.Events{
			sdk.NewEvent("tasks_processed",
				sdk.NewAttribute("count", fmt.Sprintf("%d", len(dueTasks))),
				sdk.NewAttribute("height", fmt.Sprintf("%d", currentHeight)),
			),
		})
	}

	return dueTasks
}
