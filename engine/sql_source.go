package engine

import (
	"database/sql"
	_ "github.com/denisenkom/go-mssqldb" //Microsoft SQL Server driver
	//_ "github.com/go-sql-driver/mysql"   //MySQL (4.1+), MariaDB, Percona Server, Google CloudSQL or Sphinx (2.2.3+)
	_ "github.com/lib/pq"                //Postgres
	_ "github.com/mattn/go-sqlite3"      //SQLite driver
	"time"
	"fmt"
)

type SQLSource struct {
	Name             string
	Driver           string
	ConnectionString string
	Query            string
	QueryParameters  []interface{}
	columns          []string
	db               *sql.DB
}

func (sq *SQLSource) Columns() []string {
	return sq.columns
}

func (sq *SQLSource) connect() error {
	var err error
	sq.db, err = sql.Open(sq.Driver, sq.ConnectionString)
	if err != nil {
		return fmt.Errorf("SQL destination: %s", err.Error())
	}
	return nil
}

func (sq *SQLSource) Ping() error {
	if sq.db == nil {
		err := sq.connect()
		if err != nil {
			return err
		}
	}
	return sq.db.Ping()
}

func (sq *SQLSource) fatalerr(err error, s Stream, l Logger) {
	l.Chan() <- Event{
		Level:   Error,
		Source:  sq.Name,
		Time:    time.Now(),
		Message: err.Error(),
	}
	close(s.Chan())
}

func (sq *SQLSource) Open(s Stream, l Logger, st Stopper) {
	if sq.db == nil {
		err := sq.connect()
		if err != nil {
			sq.fatalerr(err, s, l)
			return
		}
	}
	r, err := sq.db.Query(sq.Query, sq.QueryParameters...)
	if err != nil {
		sq.fatalerr(err, s, l)
		return
	}
	defer r.Close()
	cols, err := r.Columns()
	if err != nil {
		sq.fatalerr(err, s, l)
		return
	}
	sq.columns = cols
	s.SetColumns(cols)
	l.Chan() <- Event{
		Source:  sq.Name,
		Level:   Trace,
		Time:    time.Now(),
		Message: "SQL source opened",
	}
	for r.Next() {
		if st.Stopped() {
			close(s.Chan())
			return
		}
		rr := make([]interface{}, len(cols))
		rrp := makeRowPointers(rr)
		err := r.Scan(rrp...)
		rr = convertRow(rr)
		if err != nil {
			sq.fatalerr(err, s, l)
			return
		}
		s.Chan() <- rr
	}
	close(s.Chan())
}

//makeRowPointers creates a slice that points to elements of another slice. The point is that rows.Scan() requires
//the destination types to be pointers but we want interface{} types
func makeRowPointers(row []interface{}) []interface{} {
	ret := make([]interface{}, len(row), len(row))
	for i := range row {
		ret[i] = &row[i]
	}
	return ret
}

func convertRow(raw []interface{}) []interface{} {
	ret := make([]interface{}, len(raw))
	for i := range raw {
		switch v := raw[i].(type) {
		case int64:
			ret[i] = int(v)
		case []uint8:
			ret[i] = string(v)
		default:
			ret[i] = v
		}
	}
	return ret
}
