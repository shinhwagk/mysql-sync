package main

type MysqlOperation interface {
	OperationType() string
	GetTimestamp() uint32
}

type MysqlOperationBinLogPos struct {
	ServerID  uint32
	Event     string
	File      string
	Pos       uint32
	Timestamp uint32
}

func (op MysqlOperationBinLogPos) OperationType() string {
	return "MysqlOperationBinLogPos"
}

func (op MysqlOperationBinLogPos) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationDDLTable struct {
	SchemaContext string
	Database      string
	Table         string
	Query         string
	Timestamp     uint32
}

func (op MysqlOperationDDLTable) OperationType() string {
	return "MysqlOperationDDLTable"
}

func (op MysqlOperationDDLTable) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationDDLDatabase struct {
	Database  string
	Query     string
	Timestamp uint32
}

func (op MysqlOperationDDLDatabase) OperationType() string {
	return "MysqlOperationDDLDatabase"
}

func (op MysqlOperationDDLDatabase) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationDCLUser struct {
	SchemaContext string
	Query         string
	Timestamp     uint32
}

func (op MysqlOperationDCLUser) OperationType() string {
	return "MysqlOperationDCLUser"
}

func (op MysqlOperationDCLUser) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationDMLColumn struct {
	ColumnName       string
	ColumnType       byte
	ColumnValue      interface{}
	ColumnValueIsNil bool
}

type MysqlOperationDMLInsert struct {
	Database   string
	Table      string
	Columns    []MysqlOperationDMLColumn
	PrimaryKey []uint64
	Timestamp  uint32
}

func (op MysqlOperationDMLInsert) OperationType() string {
	return "MysqlOperationDMLInsert"
}

func (op MysqlOperationDMLInsert) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationDMLDelete struct {
	Database   string
	Table      string
	Columns    []MysqlOperationDMLColumn
	PrimaryKey []uint64
	Timestamp  uint32
}

func (op MysqlOperationDMLDelete) OperationType() string {
	return "MysqlOperationDMLDelete"
}

func (op MysqlOperationDMLDelete) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationDMLUpdate struct {
	Database      string
	Table         string
	AfterColumns  []MysqlOperationDMLColumn
	BeforeColumns []MysqlOperationDMLColumn
	PrimaryKey    []uint64
	Timestamp     uint32
}

func (op MysqlOperationDMLUpdate) OperationType() string {
	return "MysqlOperationDMLUpdate"
}

func (op MysqlOperationDMLUpdate) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationXid struct {
	Timestamp uint32
}

func (op MysqlOperationXid) OperationType() string {
	return "MysqlOperationXid"
}

func (op MysqlOperationXid) GetTimestamp() uint32 {
	return op.Timestamp

}

type MysqlOperationGTID struct {
	LastCommitted int64
	ServerID      uint32
	Timestamp     uint32
	ServerUUID    string
	TrxID         int64
	// GtidNext      string
}

func (op MysqlOperationGTID) OperationType() string {
	return "MysqlOperationGTID"
}

func (op MysqlOperationGTID) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationBegin struct {
	ServerID  uint32
	Timestamp uint32
}

func (op MysqlOperationBegin) OperationType() string {
	return "MysqlOperationBegin"
}

func (op MysqlOperationBegin) GetTimestamp() uint32 {
	return op.Timestamp
}

type MysqlOperationHeartbeat struct {
	Timestamp uint32
}

func (op MysqlOperationHeartbeat) OperationType() string {
	return "MysqlOperationHeartbeat"
}

func (op MysqlOperationHeartbeat) GetTimestamp() uint32 {
	return op.Timestamp
}
