// Copyright 2023 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package importinto

import (
	"sync"

	"github.com/pingcap/tidb/br/pkg/lightning/backend"
	"github.com/pingcap/tidb/br/pkg/lightning/mydump"
	"github.com/pingcap/tidb/br/pkg/lightning/verification"
	"github.com/pingcap/tidb/domain/infosync"
	"github.com/pingcap/tidb/executor/asyncloaddata"
	"github.com/pingcap/tidb/executor/importer"
)

// TaskStep of IMPORT INTO.
const (
	// Import we sort source data and ingest it into TiKV in this step.
	Import int64 = 1
)

// TaskMeta is the task of IMPORT INTO.
// All the field should be serializable.
type TaskMeta struct {
	// IMPORT INTO job id.
	JobID  int64
	Plan   importer.Plan
	Stmt   string
	Result Result
	// eligible instances to run this task, we run on all instances if it's empty.
	// we only need this when run IMPORT INTO without distributed option now, i.e.
	// running on the instance that initiate the IMPORT INTO.
	EligibleInstances []*infosync.ServerInfo
	// the file chunks to import, when import from server file, we need to pass those
	// files to the framework dispatcher which might run on another instance.
	// we use a map from engine ID to chunks since we need support split_file for CSV,
	// so need to split them into engines before passing to dispatcher.
	ChunkMap map[int32][]Chunk
}

// SubtaskMeta is the subtask of IMPORT INTO.
// Dispatcher will split the task into subtasks(FileInfos -> Chunks)
// All the field should be serializable.
type SubtaskMeta struct {
	Plan importer.Plan
	// this is the engine ID, not the id in tidb_background_subtask table.
	ID       int32
	Chunks   []Chunk
	Checksum Checksum
	Result   Result
}

// SharedVars is the shared variables between subtask and minimal tasks.
// This is because subtasks cannot directly obtain the results of the minimal subtask.
// All the fields should be concurrent safe.
type SharedVars struct {
	TableImporter *importer.TableImporter
	DataEngine    *backend.OpenedEngine
	IndexEngine   *backend.OpenedEngine
	Progress      *asyncloaddata.Progress

	mu       sync.Mutex
	Checksum *verification.KVChecksum
}

// MinimalTaskMeta is the minimal task of IMPORT INTO.
// Scheduler will split the subtask into minimal tasks(Chunks -> Chunk)
type MinimalTaskMeta struct {
	Plan       importer.Plan
	Chunk      Chunk
	SharedVars *SharedVars
}

// IsMinimalTask implements the MinimalTask interface.
func (MinimalTaskMeta) IsMinimalTask() {}

// Chunk records the chunk information.
type Chunk struct {
	Path         string
	Offset       int64
	EndOffset    int64
	PrevRowIDMax int64
	RowIDMax     int64
	Type         mydump.SourceType
	Compression  mydump.Compression
	Timestamp    int64
}

// Checksum records the checksum information.
type Checksum struct {
	Sum  uint64
	KVs  uint64
	Size uint64
}

// Result records the metrics information.
// This portion of the code may be implemented uniformly in the framework in the future.
type Result struct {
	ReadRowCnt   uint64
	LoadedRowCnt uint64
	ColSizeMap   map[int64]int64
}
