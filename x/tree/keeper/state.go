package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/tree/types"
)

// ======================== Project CRUD ========================

// SetProject stores a project and updates indexes.
func (k Keeper) SetProject(ctx sdk.Context, project types.ProductProject) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&project)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.ProjectKey(project.Id), bz); err != nil {
		panic(err)
	}

	// Update founder index
	if project.Founder != "" {
		if err := store.Set(types.ProjectFounderIndexKey(project.Founder, project.Id), []byte{0x01}); err != nil {
			panic(err)
		}
	}
	// Update domain index
	if project.KnowledgeDomain != "" {
		if err := store.Set(types.ProjectDomainIndexKey(project.KnowledgeDomain, project.Id), []byte{0x01}); err != nil {
			panic(err)
		}
	}
}

// GetProject retrieves a project by ID.
func (k Keeper) GetProject(ctx sdk.Context, projectId string) (types.ProductProject, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ProjectKey(projectId))
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return types.ProductProject{}, false
	}
	var project types.ProductProject
	if err := proto.Unmarshal(bz, &project); err != nil {
		panic(err)
	}
	return project, true
}

// DeleteProject removes a project and its indexes.
func (k Keeper) DeleteProject(ctx sdk.Context, project types.ProductProject) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.ProjectKey(project.Id)); err != nil {
		panic(err)
	}

	if project.Founder != "" {
		if err := store.Delete(types.ProjectFounderIndexKey(project.Founder, project.Id)); err != nil {
			panic(err)
		}
	}
	if project.KnowledgeDomain != "" {
		if err := store.Delete(types.ProjectDomainIndexKey(project.KnowledgeDomain, project.Id)); err != nil {
			panic(err)
		}
	}
}

// IterateProjects iterates all projects. Return true from cb to stop.
func (k Keeper) IterateProjects(ctx sdk.Context, cb func(types.ProductProject) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ProjectKeyPrefix, prefixEndBytes(types.ProjectKeyPrefix))
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var project types.ProductProject
		if err := proto.Unmarshal(iter.Value(), &project); err != nil {
			panic(err)
		}
		if cb(project) {
			break
		}
	}
}

// GetProjectsByFounder returns all projects for a given founder address.
func (k Keeper) GetProjectsByFounder(ctx sdk.Context, founder string) []types.ProductProject {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.ProjectFounderIndexPrefix, []byte(founder+"/")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	var projects []types.ProductProject
	for ; iter.Valid(); iter.Next() {
		// Extract projectId from index key: prefix | founder/ | projectId
		key := iter.Key()
		projectId := string(key[len(prefix):])
		project, found := k.GetProject(ctx, projectId)
		if found {
			projects = append(projects, project)
		}
	}
	return projects
}

// GetNextProjectId returns and increments the project ID counter.
func (k Keeper) GetNextProjectId(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.Uint64ToBytes(0x20))
	if err != nil {
		panic(err)
	}
	var counter uint64
	if bz != nil {
		counter = types.BytesToUint64(bz)
	}
	counter++
	if err := store.Set(types.Uint64ToBytes(0x20), types.Uint64ToBytes(counter)); err != nil {
		panic(err)
	}
	return counter
}

// ======================== Task CRUD ========================

// SetTask stores a task and updates indexes.
func (k Keeper) SetTask(ctx sdk.Context, task types.ProjectTask) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&task)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.TaskKey(task.Id), bz); err != nil {
		panic(err)
	}

	// Update project index
	if task.ProjectId != "" {
		if err := store.Set(types.TaskProjectIndexKey(task.ProjectId, task.Id), []byte{0x01}); err != nil {
			panic(err)
		}
	}
	// Update assignee index
	if task.Assignee != "" {
		if err := store.Set(types.TaskAssigneeIndexKey(task.Assignee, task.Id), []byte{0x01}); err != nil {
			panic(err)
		}
	}
}

// GetTask retrieves a task by ID.
func (k Keeper) GetTask(ctx sdk.Context, taskId string) (types.ProjectTask, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.TaskKey(taskId))
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return types.ProjectTask{}, false
	}
	var task types.ProjectTask
	if err := proto.Unmarshal(bz, &task); err != nil {
		panic(err)
	}
	return task, true
}

// DeleteTask removes a task and its indexes.
func (k Keeper) DeleteTask(ctx sdk.Context, task types.ProjectTask) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.TaskKey(task.Id)); err != nil {
		panic(err)
	}

	if task.ProjectId != "" {
		if err := store.Delete(types.TaskProjectIndexKey(task.ProjectId, task.Id)); err != nil {
			panic(err)
		}
	}
	if task.Assignee != "" {
		if err := store.Delete(types.TaskAssigneeIndexKey(task.Assignee, task.Id)); err != nil {
			panic(err)
		}
	}
}

// IterateTasks iterates all tasks.
func (k Keeper) IterateTasks(ctx sdk.Context, cb func(types.ProjectTask) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.TaskKeyPrefix, prefixEndBytes(types.TaskKeyPrefix))
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var task types.ProjectTask
		if err := proto.Unmarshal(iter.Value(), &task); err != nil {
			panic(err)
		}
		if cb(task) {
			break
		}
	}
}

// GetTasksByProject returns all tasks for a given project.
func (k Keeper) GetTasksByProject(ctx sdk.Context, projectId string) []types.ProjectTask {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.TaskProjectIndexPrefix, []byte(projectId+"/")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	var tasks []types.ProjectTask
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		taskId := string(key[len(prefix):])
		task, found := k.GetTask(ctx, taskId)
		if found {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetNextTaskId returns and increments the task ID counter.
func (k Keeper) GetNextTaskId(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.Uint64ToBytes(0x21))
	if err != nil {
		panic(err)
	}
	var counter uint64
	if bz != nil {
		counter = types.BytesToUint64(bz)
	}
	counter++
	if err := store.Set(types.Uint64ToBytes(0x21), types.Uint64ToBytes(counter)); err != nil {
		panic(err)
	}
	return counter
}

// ======================== Service CRUD ========================

// SetService stores a service and updates indexes.
func (k Keeper) SetService(ctx sdk.Context, service types.ServiceLeaf) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&service)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.ServiceKey(service.Id), bz); err != nil {
		panic(err)
	}

	if service.ProjectId != "" {
		if err := store.Set(types.ServiceProjectIndexKey(service.ProjectId, service.Id), []byte{0x01}); err != nil {
			panic(err)
		}
	}
}

// GetService retrieves a service by ID.
func (k Keeper) GetService(ctx sdk.Context, serviceId string) (types.ServiceLeaf, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ServiceKey(serviceId))
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return types.ServiceLeaf{}, false
	}
	var service types.ServiceLeaf
	if err := proto.Unmarshal(bz, &service); err != nil {
		panic(err)
	}
	return service, true
}

// IterateServices iterates all services.
func (k Keeper) IterateServices(ctx sdk.Context, cb func(types.ServiceLeaf) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ServiceKeyPrefix, prefixEndBytes(types.ServiceKeyPrefix))
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var service types.ServiceLeaf
		if err := proto.Unmarshal(iter.Value(), &service); err != nil {
			panic(err)
		}
		if cb(service) {
			break
		}
	}
}

// GetNextServiceId returns and increments the service ID counter.
func (k Keeper) GetNextServiceId(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.Uint64ToBytes(0x22))
	if err != nil {
		panic(err)
	}
	var counter uint64
	if bz != nil {
		counter = types.BytesToUint64(bz)
	}
	counter++
	if err := store.Set(types.Uint64ToBytes(0x22), types.Uint64ToBytes(counter)); err != nil {
		panic(err)
	}
	return counter
}

// ======================== Seed CRUD ========================

// SetSeed stores an opportunity seed and updates indexes.
func (k Keeper) SetSeed(ctx sdk.Context, seed types.OpportunitySeed) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&seed)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.SeedKey(seed.Id), bz); err != nil {
		panic(err)
	}

	if seed.KnowledgeDomain != "" {
		if err := store.Set(types.SeedDomainIndexKey(seed.KnowledgeDomain, seed.Id), []byte{0x01}); err != nil {
			panic(err)
		}
	}
}

// GetSeed retrieves a seed by ID.
func (k Keeper) GetSeed(ctx sdk.Context, seedId string) (types.OpportunitySeed, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.SeedKey(seedId))
	if err != nil {
		panic(err)
	}
	if bz == nil {
		return types.OpportunitySeed{}, false
	}
	var seed types.OpportunitySeed
	if err := proto.Unmarshal(bz, &seed); err != nil {
		panic(err)
	}
	return seed, true
}

// IterateSeeds iterates all seeds.
func (k Keeper) IterateSeeds(ctx sdk.Context, cb func(types.OpportunitySeed) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.SeedKeyPrefix, prefixEndBytes(types.SeedKeyPrefix))
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var seed types.OpportunitySeed
		if err := proto.Unmarshal(iter.Value(), &seed); err != nil {
			panic(err)
		}
		if cb(seed) {
			break
		}
	}
}

// GetNextSeedId returns and increments the seed ID counter.
func (k Keeper) GetNextSeedId(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.Uint64ToBytes(0x23))
	if err != nil {
		panic(err)
	}
	var counter uint64
	if bz != nil {
		counter = types.BytesToUint64(bz)
	}
	counter++
	if err := store.Set(types.Uint64ToBytes(0x23), types.Uint64ToBytes(counter)); err != nil {
		panic(err)
	}
	return counter
}

// GetExpiredSeeds returns all seeds past their expiry block.
func (k Keeper) GetExpiredSeeds(ctx sdk.Context, currentBlock uint64) []types.OpportunitySeed {
	var expired []types.OpportunitySeed
	k.IterateSeeds(ctx, func(s types.OpportunitySeed) bool {
		if s.Status != string(types.SeedExpired) && s.ExpiresAtBlock > 0 && currentBlock >= s.ExpiresAtBlock {
			expired = append(expired, s)
		}
		return false
	})
	return expired
}

// ======================== Params ========================

// SetParams stores module params.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&params)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(err)
	}
}

// GetParams retrieves module params.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil {
		return *types.DefaultParams()
	}
	if bz == nil {
		return *types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return *types.DefaultParams()
	}
	return params
}

// ======================== Phase Transition Validation ========================

// ValidPhaseTransitions maps each phase to its valid next phases.
var ValidPhaseTransitions = map[types.ProjectPhase][]types.ProjectPhase{
	types.PhaseSeed:     {types.PhaseSprout, types.PhaseWithered},
	types.PhaseSprout:   {types.PhaseGrowing, types.PhaseWithered},
	types.PhaseGrowing:  {types.PhaseMature, types.PhaseDormant, types.PhaseWithered},
	types.PhaseMature:   {types.PhaseFruiting, types.PhaseDormant, types.PhaseWithered},
	types.PhaseFruiting: {types.PhaseSeeding, types.PhaseDormant, types.PhaseWithered},
	types.PhaseSeeding:  {types.PhaseMature, types.PhaseDormant, types.PhaseWithered},
	types.PhaseDormant:  {types.PhaseGrowing, types.PhaseMature, types.PhaseFruiting, types.PhaseWithered},
	types.PhaseWithered: {},
}

// IsValidPhaseTransition checks if a phase transition is allowed.
func IsValidPhaseTransition(from, to types.ProjectPhase) bool {
	validTargets, ok := ValidPhaseTransitions[from]
	if !ok {
		return false
	}
	for _, target := range validTargets {
		if target == to {
			return true
		}
	}
	return false
}

// IsContributor checks if an address is a contributor to the project.
func IsContributor(project types.ProductProject, addr string) bool {
	for _, c := range project.Contributors {
		if c.Did == addr {
			return true
		}
	}
	return false
}

// IsFounderOrContributor checks if an address is the founder or a contributor.
func IsFounderOrContributor(project types.ProductProject, addr string) bool {
	if project.Founder == addr {
		return true
	}
	return IsContributor(project, addr)
}
