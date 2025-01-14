// SPDX-License-Identifier: GPL-3.0-or-later

package postgres

import (
	"fmt"
	"strings"

	"github.com/netdata/go.d.plugin/agent/module"
)

const (
	prioConnectionsUtilization = module.Priority + iota
	prioConnectionsUsage
	prioCheckpoints
	prioCheckpointTime
	prioBGWriterBuffersAllocated
	prioBGWriterBuffersWritten
	prioBGWriterMaxWrittenClean
	prioBGWriterBackedFsync
	prioDBTransactions
	prioDBConnectionsUtilization
	prioDBConnections
	prioDBBufferCache
	prioDBReadOperations
	prioDBWriteOperations
	prioDBConflicts
	prioDBConflictsStat
	prioDBDeadlocks
	prioDBLocksHeld
	prioDBLocksAwaited
	prioDBTempFiles
	prioDBTempFilesData
	prioDBSize
)

var baseCharts = module.Charts{
	serverConnectionsUtilizationChart.Copy(),
	serverConnectionsUsageChart.Copy(),
	checkpointsChart.Copy(),
	checkpointWriteChart.Copy(),
	bgWriterBuffersWrittenChart.Copy(),
	bgWriterBuffersAllocChart.Copy(),
	bgWriterMaxWrittenCleanChart.Copy(),
	bgWriterBuffersBackendFsyncChart.Copy(),
}

var (
	serverConnectionsUtilizationChart = module.Chart{
		ID:       "connections_utilization",
		Title:    "Connections utilization",
		Units:    "percentage",
		Fam:      "connections",
		Ctx:      "postgres.connections_utilization",
		Priority: prioConnectionsUtilization,
		Dims: module.Dims{
			{ID: "server_connections_utilization", Name: "used"},
		},
	}
	serverConnectionsUsageChart = module.Chart{
		ID:       "connections_usage",
		Title:    "Connections usage",
		Units:    "connections",
		Fam:      "connections",
		Ctx:      "postgres.connections_usage",
		Priority: prioConnectionsUsage,
		Type:     module.Stacked,
		Dims: module.Dims{
			{ID: "server_connections_available", Name: "available"},
			{ID: "server_connections_used", Name: "used"},
		},
	}

	checkpointsChart = module.Chart{
		ID:       "checkpoints",
		Title:    "Checkpoints",
		Units:    "checkpoints/s",
		Fam:      "checkpointer",
		Ctx:      "postgres.checkpoints",
		Priority: prioCheckpoints,
		Type:     module.Stacked,
		Dims: module.Dims{
			{ID: "checkpoints_timed", Name: "scheduled", Algo: module.Incremental},
			{ID: "checkpoints_req", Name: "requested", Algo: module.Incremental},
		},
	}
	// TODO: should be seconds, also it is units/s when using incremental...
	checkpointWriteChart = module.Chart{
		ID:       "checkpoint_time",
		Title:    "Checkpoint time",
		Units:    "milliseconds",
		Fam:      "checkpointer",
		Ctx:      "postgres.checkpoint_time",
		Priority: prioCheckpointTime,
		Dims: module.Dims{
			{ID: "checkpoint_write_time", Name: "write", Algo: module.Incremental},
			{ID: "checkpoint_sync_time", Name: "sync", Algo: module.Incremental},
		},
	}

	bgWriterBuffersAllocChart = module.Chart{
		ID:       "bgwriter_buffers_alloc",
		Title:    "Background writer buffers allocated",
		Units:    "B/s",
		Fam:      "background writer",
		Ctx:      "postgres.bgwriter_buffers_alloc",
		Priority: prioBGWriterBuffersAllocated,
		Dims: module.Dims{
			{ID: "buffers_alloc", Name: "allocated", Algo: module.Incremental},
		},
	}
	bgWriterBuffersWrittenChart = module.Chart{
		ID:       "bgwriter_buffers_written",
		Title:    "Background writer buffers written",
		Units:    "B/s",
		Fam:      "background writer",
		Ctx:      "postgres.bgwriter_buffers_written",
		Priority: prioBGWriterBuffersWritten,
		Type:     module.Area,
		Dims: module.Dims{
			{ID: "buffers_checkpoint", Name: "checkpoint", Algo: module.Incremental},
			{ID: "buffers_backend", Name: "backend", Algo: module.Incremental},
			{ID: "buffers_clean", Name: "clean", Algo: module.Incremental},
		},
	}
	bgWriterMaxWrittenCleanChart = module.Chart{
		ID:       "bgwriter_maxwritten_clean",
		Title:    "Background writer cleaning scan stops",
		Units:    "events/s",
		Fam:      "background writer",
		Ctx:      "postgres.bgwriter_maxwritten_clean",
		Priority: prioBGWriterMaxWrittenClean,
		Dims: module.Dims{
			{ID: "maxwritten_clean", Name: "maxwritten", Algo: module.Incremental},
		},
	}
	bgWriterBuffersBackendFsyncChart = module.Chart{
		ID:       "bgwriter_buffers_backend_fsync",
		Title:    "Backend fsync",
		Units:    "operations/s",
		Fam:      "background writer",
		Ctx:      "postgres.bgwriter_buffers_backend_fsync",
		Priority: prioBGWriterBackedFsync,
		Dims: module.Dims{
			{ID: "buffers_backend_fsync", Name: "fsync", Algo: module.Incremental},
		},
	}
)

var (
	dbChartsTmpl = module.Charts{
		dbTransactionsChartTmpl.Copy(),
		dbConnectionsUtilizationChartTmpl.Copy(),
		dbConnectionsChartTmpl.Copy(),
		dbBufferCacheChartTmpl.Copy(),
		dbReadOpsChartTmpl.Copy(),
		dbWriteOpsChartTmpl.Copy(),
		dbConflictsChartTmpl.Copy(),
		dbConflictsStatChartTmpl.Copy(),
		dbDeadlocksChartTmpl.Copy(),
		dbLocksHeldChartTmpl.Copy(),
		dbLocksAwaitedChartTmpl.Copy(),
		dbTempFilesChartTmpl.Copy(),
		dbTempFilesDataChartTmpl.Copy(),
		dbSizeChartTmpl.Copy(),
	}
	dbTransactionsChartTmpl = module.Chart{
		ID:       "db_%s_transactions",
		Title:    "Database transactions",
		Units:    "transactions/s",
		Fam:      "db transactions",
		Ctx:      "postgres.db_transactions",
		Priority: prioDBTransactions,
		Dims: module.Dims{
			{ID: "db_%s_xact_commit", Name: "committed", Algo: module.Incremental},
			{ID: "db_%s_xact_rollback", Name: "rollback", Algo: module.Incremental},
		},
	}
	dbConnectionsUtilizationChartTmpl = module.Chart{
		ID:       "db_%s_connections_utilization",
		Title:    "Database connections utilization withing limits",
		Units:    "percentage",
		Fam:      "db connections",
		Ctx:      "postgres.db_connections_utilization",
		Priority: prioDBConnectionsUtilization,
		Dims: module.Dims{
			{ID: "db_%s_numbackends_utilization", Name: "used"},
		},
	}
	dbConnectionsChartTmpl = module.Chart{
		ID:       "db_%s_connections",
		Title:    "Database connections",
		Units:    "connections",
		Fam:      "db connections",
		Ctx:      "postgres.db_connections",
		Priority: prioDBConnections,
		Dims: module.Dims{
			{ID: "db_%s_numbackends", Name: "connections"},
		},
	}
	dbBufferCacheChartTmpl = module.Chart{
		ID:       "db_%s_buffer_cache",
		Title:    "Database buffer cache",
		Units:    "blocks/s",
		Fam:      "db buffer cache",
		Ctx:      "postgres.db_buffer_cache",
		Priority: prioDBBufferCache,
		Type:     module.Area,
		Dims: module.Dims{
			{ID: "db_%s_blks_hit", Name: "hit", Algo: module.Incremental},
			{ID: "db_%s_blks_read", Name: "miss", Algo: module.Incremental},
		},
	}
	dbReadOpsChartTmpl = module.Chart{
		ID:       "db_%s_read_operations",
		Title:    "Database read operations",
		Units:    "rows/s",
		Fam:      "db operations",
		Ctx:      "postgres.db_read_operations",
		Priority: prioDBReadOperations,
		Dims: module.Dims{
			{ID: "db_%s_tup_returned", Name: "returned", Algo: module.Incremental},
			{ID: "db_%s_tup_fetched", Name: "fetched", Algo: module.Incremental},
		},
	}
	dbWriteOpsChartTmpl = module.Chart{
		ID:       "db_%s_write_operations",
		Title:    "Database write operations",
		Units:    "rows/s",
		Fam:      "db operations",
		Ctx:      "postgres.db_write_operations",
		Priority: prioDBWriteOperations,
		Dims: module.Dims{
			{ID: "db_%s_tup_inserted", Name: "inserted", Algo: module.Incremental},
			{ID: "db_%s_tup_deleted", Name: "deleted", Algo: module.Incremental},
			{ID: "db_%s_tup_updated", Name: "updated", Algo: module.Incremental},
		},
	}
	dbConflictsChartTmpl = module.Chart{
		ID:       "db_%s_conflicts",
		Title:    "Database canceled queries",
		Units:    "queries/s",
		Fam:      "db operations",
		Ctx:      "postgres.db_conflicts",
		Priority: prioDBConflicts,
		Dims: module.Dims{
			{ID: "db_%s_conflicts", Name: "conflicts", Algo: module.Incremental},
		},
	}
	dbConflictsStatChartTmpl = module.Chart{
		ID:       "db_%s_conflicts_stat",
		Title:    "Database canceled queries by reason",
		Units:    "queries/s",
		Fam:      "db operations",
		Ctx:      "postgres.db_conflicts_stat",
		Priority: prioDBConflictsStat,
		Dims: module.Dims{
			{ID: "db_%s_confl_tablespace", Name: "tablespace", Algo: module.Incremental},
			{ID: "db_%s_confl_lock", Name: "lock", Algo: module.Incremental},
			{ID: "db_%s_confl_snapshot", Name: "snapshot", Algo: module.Incremental},
			{ID: "db_%s_confl_bufferpin", Name: "bufferpin", Algo: module.Incremental},
			{ID: "db_%s_confl_deadlock", Name: "deadlock", Algo: module.Incremental},
		},
	}
	dbDeadlocksChartTmpl = module.Chart{
		ID:       "db_%s_deadlocks",
		Title:    "Database deadlocks",
		Units:    "deadlocks/s",
		Fam:      "db deadlocks",
		Ctx:      "postgres.db_deadlocks",
		Priority: prioDBDeadlocks,
		Dims: module.Dims{
			{ID: "db_%s_deadlocks", Name: "deadlocks", Algo: module.Incremental},
		},
	}
	dbLocksHeldChartTmpl = module.Chart{
		ID:       "db_%s_locks_held",
		Title:    "Database locks held",
		Units:    "locks",
		Fam:      "db locks",
		Ctx:      "postgres.db_locks_held",
		Priority: prioDBLocksHeld,
		Type:     module.Stacked,
		Dims: module.Dims{
			{ID: "db_%s_lock_mode_AccessShareLock_held", Name: "access_share"},
			{ID: "db_%s_lock_mode_RowShareLock_held", Name: "row_share"},
			{ID: "db_%s_lock_mode_RowExclusiveLock_held", Name: "row_exclusive"},
			{ID: "db_%s_lock_mode_ShareUpdateExclusiveLock_held", Name: "share_update"},
			{ID: "db_%s_lock_mode_ShareLock_held", Name: "share"},
			{ID: "db_%s_lock_mode_ShareRowExclusiveLock_held", Name: "share_row_exclusive"},
			{ID: "db_%s_lock_mode_ExclusiveLock_held", Name: "exclusive"},
			{ID: "db_%s_lock_mode_AccessExclusiveLock_held", Name: "access_exclusive"},
		},
	}
	dbLocksAwaitedChartTmpl = module.Chart{
		ID:       "db_%s_locks_awaited",
		Title:    "Database locks awaited",
		Units:    "locks",
		Fam:      "db locks",
		Ctx:      "postgres.db_locks_awaited",
		Priority: prioDBLocksAwaited,
		Type:     module.Stacked,
		Dims: module.Dims{
			{ID: "db_%s_lock_mode_AccessShareLock_awaited", Name: "access_share"},
			{ID: "db_%s_lock_mode_RowShareLock_awaited", Name: "row_share"},
			{ID: "db_%s_lock_mode_RowExclusiveLock_awaited", Name: "row_exclusive"},
			{ID: "db_%s_lock_mode_ShareUpdateExclusiveLock_awaited", Name: "share_update"},
			{ID: "db_%s_lock_mode_ShareLock_awaited", Name: "share"},
			{ID: "db_%s_lock_mode_ShareRowExclusiveLock_awaited", Name: "share_row_exclusive"},
			{ID: "db_%s_lock_mode_ExclusiveLock_awaited", Name: "exclusive"},
			{ID: "db_%s_lock_mode_AccessExclusiveLock_awaited", Name: "access_exclusive"},
		},
	}
	dbTempFilesChartTmpl = module.Chart{
		ID:       "db_%s_temp_files",
		Title:    "Database temporary files written to disk",
		Units:    "files/s",
		Fam:      "db temp files",
		Ctx:      "postgres.db_temp_files",
		Priority: prioDBTempFiles,
		Dims: module.Dims{
			{ID: "db_%s_temp_files", Name: "written", Algo: module.Incremental},
		},
	}
	dbTempFilesDataChartTmpl = module.Chart{
		ID:       "db_%s_temp_files_data",
		Title:    "Database temporary files data written to disk",
		Units:    "B/s",
		Fam:      "db temp files",
		Ctx:      "postgres.db_temp_files_data",
		Priority: prioDBTempFilesData,
		Dims: module.Dims{
			{ID: "db_%s_temp_bytes", Name: "written", Algo: module.Incremental},
		},
	}
	dbSizeChartTmpl = module.Chart{
		ID:       "db_%s_size",
		Title:    "Database size",
		Units:    "B",
		Fam:      "db size",
		Ctx:      "postgres.db_size",
		Priority: prioDBSize,
		Dims: module.Dims{
			{ID: "db_%s_size", Name: "size"},
		},
	}
)

func newDatabaseCharts(dbname string) *module.Charts {
	charts := dbChartsTmpl.Copy()
	for _, c := range *charts {
		c.ID = fmt.Sprintf(c.ID, dbname)
		c.Labels = []module.Label{
			{Key: "database", Value: dbname},
		}
		for _, d := range c.Dims {
			d.ID = fmt.Sprintf(d.ID, dbname)
		}
	}
	return charts
}

func (p *Postgres) addNewDatabaseCharts(dbname string) {
	charts := newDatabaseCharts(dbname)
	if err := p.Charts().Add(*charts...); err != nil {
		p.Warning(err)
	}
}

func (p *Postgres) removeDatabaseCharts(dbname string) {
	prefix := fmt.Sprintf("db_%s_", dbname)
	for _, c := range *p.Charts() {
		if strings.HasPrefix(c.ID, prefix) {
			c.MarkRemove()
			c.MarkNotCreated()
		}
	}
}
