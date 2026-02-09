package types

import "encoding/binary"

const (
	// ModuleName defines the module name.
	ModuleName = "tree"

	// StoreKey defines the primary module store key.
	StoreKey = ModuleName

	// RouterKey defines the routing key.
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key.
	MemStoreKey = "mem_" + ModuleName

	// QuerierRoute defines the querier route.
	QuerierRoute = ModuleName
)

// Store key prefixes.
var (
	ProjectKeyPrefix  = []byte{0x01}
	ServiceKeyPrefix  = []byte{0x02}
	TaskKeyPrefix     = []byte{0x03}
	SeedKeyPrefix     = []byte{0x04}
	ParamsKey         = []byte{0x05}

	// Index prefixes for lookups.
	ProjectFounderIndexPrefix = []byte{0x10} // 0x10 | founderAddr | projectId -> nil
	ProjectDomainIndexPrefix  = []byte{0x11} // 0x11 | domain | projectId -> nil
	TaskProjectIndexPrefix    = []byte{0x12} // 0x12 | projectId | taskId -> nil
	TaskAssigneeIndexPrefix   = []byte{0x13} // 0x13 | assigneeAddr | taskId -> nil
	ServiceProjectIndexPrefix = []byte{0x14} // 0x14 | projectId | serviceId -> nil
	SeedDomainIndexPrefix     = []byte{0x15} // 0x15 | domain | seedId -> nil

	// Counters
	projectCounterKey = []byte{0x20}
	taskCounterKey    = []byte{0x21}
	serviceCounterKey = []byte{0x22}
	seedCounterKey    = []byte{0x23}
)

// ProjectKey returns the store key for a project.
func ProjectKey(projectId string) []byte {
	return append(ProjectKeyPrefix, []byte(projectId)...)
}

// TaskKey returns the store key for a task.
func TaskKey(taskId string) []byte {
	return append(TaskKeyPrefix, []byte(taskId)...)
}

// ServiceKey returns the store key for a service.
func ServiceKey(serviceId string) []byte {
	return append(ServiceKeyPrefix, []byte(serviceId)...)
}

// SeedKey returns the store key for an opportunity seed.
func SeedKey(seedId string) []byte {
	return append(SeedKeyPrefix, []byte(seedId)...)
}

// ProjectFounderIndexKey returns the index key for a project by founder.
func ProjectFounderIndexKey(founder, projectId string) []byte {
	key := make([]byte, 0, len(ProjectFounderIndexPrefix)+len(founder)+1+len(projectId))
	key = append(key, ProjectFounderIndexPrefix...)
	key = append(key, []byte(founder)...)
	key = append(key, '/')
	key = append(key, []byte(projectId)...)
	return key
}

// ProjectDomainIndexKey returns the index key for a project by domain.
func ProjectDomainIndexKey(domain, projectId string) []byte {
	key := make([]byte, 0, len(ProjectDomainIndexPrefix)+len(domain)+1+len(projectId))
	key = append(key, ProjectDomainIndexPrefix...)
	key = append(key, []byte(domain)...)
	key = append(key, '/')
	key = append(key, []byte(projectId)...)
	return key
}

// TaskProjectIndexKey returns the index key for a task by project.
func TaskProjectIndexKey(projectId, taskId string) []byte {
	key := make([]byte, 0, len(TaskProjectIndexPrefix)+len(projectId)+1+len(taskId))
	key = append(key, TaskProjectIndexPrefix...)
	key = append(key, []byte(projectId)...)
	key = append(key, '/')
	key = append(key, []byte(taskId)...)
	return key
}

// TaskAssigneeIndexKey returns the index key for a task by assignee.
func TaskAssigneeIndexKey(assignee, taskId string) []byte {
	key := make([]byte, 0, len(TaskAssigneeIndexPrefix)+len(assignee)+1+len(taskId))
	key = append(key, TaskAssigneeIndexPrefix...)
	key = append(key, []byte(assignee)...)
	key = append(key, '/')
	key = append(key, []byte(taskId)...)
	return key
}

// ServiceProjectIndexKey returns the index key for a service by project.
func ServiceProjectIndexKey(projectId, serviceId string) []byte {
	key := make([]byte, 0, len(ServiceProjectIndexPrefix)+len(projectId)+1+len(serviceId))
	key = append(key, ServiceProjectIndexPrefix...)
	key = append(key, []byte(projectId)...)
	key = append(key, '/')
	key = append(key, []byte(serviceId)...)
	return key
}

// SeedDomainIndexKey returns the index key for a seed by domain.
func SeedDomainIndexKey(domain, seedId string) []byte {
	key := make([]byte, 0, len(SeedDomainIndexPrefix)+len(domain)+1+len(seedId))
	key = append(key, SeedDomainIndexPrefix...)
	key = append(key, []byte(domain)...)
	key = append(key, '/')
	key = append(key, []byte(seedId)...)
	return key
}

// Uint64ToBytes converts uint64 to big-endian bytes.
func Uint64ToBytes(val uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, val)
	return bz
}

// BytesToUint64 converts big-endian bytes to uint64.
func BytesToUint64(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
