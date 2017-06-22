package reiner

import (
	"database/sql"
	"fmt"
	"strings"

	// The MySQL driver.
	_ "github.com/go-sql-driver/mysql"
)

type tableName string

type function struct {
	query  string
	values []interface{}
}
type condition struct {
	column    interface{}
	operator  string
	connector string
	values    []interface{}
}

type join struct {
	table      string
	typ        string
	condition  string
	conditions []condition
}

// Wrapper represents a database connection.
type Wrapper struct {
	db                 *DB
	isSubQuery         bool
	query              string
	alias              string
	tableName          []string
	queryOptions       []string
	destination        interface{}
	joins              map[tableName]join
	params             []interface{}
	onDuplicateColumns []string
	lastInsertIDColumn string

	//
	Timestamp *Timestamp
	// Count is the count of the results, or the affected rows.
	Count int
	//
	TotalCount int
	//
	PageLimit int
	//
	TotalPage int
	// LasyQuery is last executed query.
	LastQuery string
	//
	LastInsertID int
	//
	LastInsertIDs []int
	//
	LastRows *sql.Rows
	//
	LastRow *sql.Row
}

// New creates a new database connection which provides the MySQL wrapper functions.
// The first data source name is for the master, the rest are for the slaves, which is used for the read/write split.
//     .New("root:root@/master", []string{"root:root@/slave", "root:root@/slave2"})
// Check https://dev.mysql.com/doc/refman/5.7/en/replication-solutions-scaleout.html for more information.
func newWrapper(db *DB) *Wrapper {
	return &Wrapper{db: db}
}

func (w *Wrapper) clean() {
	w.tableName = []string{}
	w.params = []interface{}{}
	w.query = ""
}

func (w *Wrapper) buildPair(data interface{}) {
	//switch v := data.(type) {
	//case *Wrapper:
	//}
}

func (w *Wrapper) bindParams(data []interface{}) {
	for _, v := range data {
		w.bindParam(v)
	}
	return
}

func (w *Wrapper) bindParam(data interface{}) (param string) {
	switch v := data.(type) {
	case *Wrapper:
		if len(v.params) > 0 {
			w.params = append(w.params, v.params...)
		}
	case function:
		if len(v.values) > 0 {
			w.params = append(w.params, v.values...)
		}
	default:
		w.params = append(w.params, data)
	}
	param = w.paramToQuery(data)
	return
}

func (w *Wrapper) paramToQuery(data interface{}) (param string) {
	switch v := data.(type) {
	case *Wrapper:
		param = fmt.Sprintf("(%s)", v.query)
	case function:
		param = v.query
	default:
		param = "?"
	}
	return
}

func (w *Wrapper) buildDuplicate() (query string) {
	if len(w.onDuplicateColumns) == 0 {
		return
	}
	query += "ON DUPLICATE KEY UPDATE "
	if w.lastInsertIDColumn != "" {
		query += fmt.Sprintf("%s=LAST_INSERT_ID(%s), ", w.lastInsertIDColumn, w.lastInsertIDColumn)
	}
	for _, v := range w.onDuplicateColumns {
		query += fmt.Sprintf("%s = VALUE(%s), ", v, v)
	}
	query = trim(query)
	return
}

func (w *Wrapper) buildInsert(operator string, data interface{}) (query string) {
	var columns, values, options string
	if len(w.queryOptions) > 0 {
		options = fmt.Sprintf("%s ", strings.Join(w.queryOptions, ", "))
	}

	switch realData := data.(type) {
	case map[string]interface{}:
		for column, value := range realData {
			columns += fmt.Sprintf("%s, ", column)
			values += fmt.Sprintf("%s, ", w.bindParam(value))
		}
		values = fmt.Sprintf("(%s)", trim(values))

	case []map[string]interface{}:
		for index, single := range realData {
			var currentValues string
			for column, value := range single {
				// Get the column names from the first data set only.
				if index == 0 {
					columns += fmt.Sprintf("%s, ", column)
				}
				currentValues += fmt.Sprintf("%s, ", w.bindParam(value))
			}
			values += fmt.Sprintf("(%s), ", trim(currentValues))
		}
		values = trim(values)
	}
	columns = trim(columns)
	query = fmt.Sprintf("%s %sINTO %s (%s) VALUES %s ", operator, options, w.tableName[0], columns, values)
	return
}

func (w *Wrapper) Table(tableName ...string) *Wrapper {
	w.tableName = tableName
	return w
}

func (w *Wrapper) Insert(data interface{}) (err error) {
	w.query = w.buildInsert("INSERT", data)
	w.query += w.buildDuplicate()
	w.LastQuery = w.query
	w.clean()
	return
}

func (w *Wrapper) InsertMulti(data interface{}) (err error) {
	w.query = w.buildInsert("INSERT", data)
	w.query += w.buildDuplicate()
	w.LastQuery = w.query
	w.clean()
	return
}

func (w *Wrapper) Replace(data interface{}) (err error) {
	w.query = w.buildInsert("REPLACE", data)
	w.LastQuery = w.query
	w.clean()
	return
}

func (w *Wrapper) Func(query string, data ...interface{}) function {
	return function{
		query:  query,
		values: data,
	}
}

func (w *Wrapper) Now(formats ...string) function {
	query := "NOW() "
	unitMap := map[string]string{
		"Y": "YEAR",
		"M": "MONTH",
		"D": "DAY",
		"W": "WEEK",
		"h": "HOUR",
		"m": "MINUTE",
		"s": "SECOND",
	}
	for _, v := range formats {
		operator := string(v[0])
		interval := v[1 : len(v)-1]
		unit := string(v[len(v)-1])
		query += fmt.Sprintf("%s INTERVAL %s %s ", operator, interval, unitMap[unit])
	}
	return function{
		query: strings.TrimSpace(query),
	}
}

func (w *Wrapper) OnDuplicate(columns []string, lastInsertID ...string) *Wrapper {
	w.onDuplicateColumns = columns
	if len(lastInsertID) != 0 {
		w.lastInsertIDColumn = lastInsertID[0]
	}
	return w
}

func (w *Wrapper) Update(data interface{}) (err error) {
	return
}

func (w *Wrapper) Limit(count int) *Wrapper {
	return w
}

func (w *Wrapper) Get(columns ...string) (err error) {
	return
}

func (w *Wrapper) GetOne(columns ...string) (err error) {
	return
}

func (w *Wrapper) GetValue(column string) (err error) {
	return
}

func (w *Wrapper) Paginate(pageCount int) (err error) {
	return
}

func (w *Wrapper) RawQuery(query string, values ...interface{}) (err error) {
	return
}

func (w *Wrapper) RawQueryOne(query string, values ...interface{}) (err error) {
	return
}

func (w *Wrapper) RawQueryValue(query string, values ...interface{}) (err error) {
	return
}

func (w *Wrapper) Having(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) OrHaving(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) Where(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) OrWhere(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) WhereBetween(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) WhereNotBetween(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) WhereIn(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) WhereNotIn(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) WhereNull(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) WhereNotNull(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) Delete() (err error) {
	return
}

func (w *Wrapper) OrderBy(column string, args ...interface{}) *Wrapper { //sorting string, fields ...[]string
	return w
}

func (w *Wrapper) GroupBy(column string) *Wrapper {
	return w
}

func (w *Wrapper) LeftJoin(table interface{}, condition string) *Wrapper {
	return w
}

func (w *Wrapper) RightJoin(table interface{}, condition string) *Wrapper {
	return w
}

func (w *Wrapper) InnerJoin(table interface{}, condition string) *Wrapper {
	return w
}

func (w *Wrapper) NatualJoin(table interface{}, condition string) *Wrapper {
	return w
}

func (w *Wrapper) CrossJoin(table interface{}, condition string) *Wrapper {
	return w
}

func (w *Wrapper) JoinWhere(table string, args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) JoinOrWhere(args ...interface{}) *Wrapper {
	return w
}

func (w *Wrapper) SubQuery(alias ...string) *Wrapper {
	return w
}

func (w *Wrapper) Has() (has bool, err error) {
	return
}

func (w *Wrapper) Disconnect() (err error) {
	return
}

func (w *Wrapper) Ping() (err error) {
	return
}

func (w *Wrapper) Connect() (err error) {
	return
}

func (w *Wrapper) Begin() *Wrapper {
	return w
}

func (w *Wrapper) Rollback() *Wrapper {
	return w
}

func (w *Wrapper) Commit() *Wrapper {
	return w
}

func (w *Wrapper) SetLockMethod() *Wrapper {
	return w
}

func (w *Wrapper) Lock() *Wrapper {
	return w
}

func (w *Wrapper) Unlock() *Wrapper {
	return w
}

func (w *Wrapper) SetQueryOption(options ...string) *Wrapper {
	return w
}

// Migration returns a new table migration struct
// based on the current database connection for the migration functions.
func (w *Wrapper) Migration() *Migration {
	return newMigration(w.db)
}
