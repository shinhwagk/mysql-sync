package main

type MetricDestination struct {
	Delay            uint64 `json:"delay"`
	DMLInsertTimes   uint64 `json:"dml_insert_times"`
	DMLUpdateTimes   uint64 `json:"dml_update_times"`
	DMLDeleteTimes   uint64 `json:"dml_delete_times"`
	DDLDatabaseTimes uint64 `json:"ddl_database_times"`
	DDLTableTimes    uint64 `json:"ddl_table_times"`
}

type MetricReplication struct {
	Delay            uint64 `json:"delay"`
	DMLInsertTimes   uint64 `json:"dml_insert_times"`
	DMLUpdateTimes   uint64 `json:"dml_update_times"`
	DMLDeleteTimes   uint64 `json:"dml_delete_times"`
	DDLDatabaseTimes uint64 `json:"ddl_database_times"`
	DDLTableTimes    uint64 `json:"ddl_table_times"`
}

type MetricDirector struct {
	logger *Logger
	hjdb   *HJDB
}

func NewMetricDirector(logLevel int, hjdb *HJDB) *MetricDirector {
	return &MetricDirector{
		logger: NewLogger(logLevel, "metric"),
		hjdb:   hjdb,
	}
}

func (m MetricDirector) push(name string, md MetricDestination) error {
	return m.hjdb.Update("metric_"+name, md)
}
