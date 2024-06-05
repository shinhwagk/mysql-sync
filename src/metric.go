package main

type MetricDestination struct {
	Delay            uint64 `json:"delay"`
	DMLInsertTimes   uint64 `json:"dml_insert_times"`
	DMLUpdateTimes   uint64 `json:"dml_update_times"`
	DMLDeleteTimes   uint64 `json:"dml_delete_times"`
	DDLDatabaseTimes uint64 `json:"ddl_database_times"`
	DDLTableTimes    uint64 `json:"ddl_table_times"`
}

type MetricCanal struct{}

type MetricDirector struct {

	// delay          uint64

	hjdb   *HJDB
	logger *Logger
}

func NewMetric(hjdb *HJDB, logger *Logger) *MetricDirector {
	return &MetricDirector{}
}

func (m MetricDirector) push(name string, md MetricDestination) error {
	return m.hjdb.Update("metric_"+name, md)
}
