package engine

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type SQLDestination struct {
	Name             string
	Driver           string
	ConnectionString string
	Table            string `aql: TABLE`
	Tx               *sql.Tx
	columns          []string
	manageTx         bool `aql: "MANAGED_TRANSACTION, optional"`
	db               *sql.DB
	Alias            string
}

const InsertQuery = `INSERT INTO %s (%s) VALUES (%s)`

func (sq *SQLDestination) Columns() []string {
	return sq.columns
}

func (sq *SQLDestination) connect() error {
	var err error
	sq.db, err = SQLDriverManager.DB(sq.Driver, sq.ConnectionString)
	if err != nil {
		return fmt.Errorf("SQL destination: %s", err.Error())
	}
	return nil
}

func (sq *SQLDestination) Ping() error {
	if sq.db == nil {
		err := sq.connect()
		if err != nil {
			return err
		}
	}
	return sq.db.Ping()
}

func (sq *SQLDestination) fatalerr(err error, l Logger, st Stopper) {
	l.Chan() <- Event{
		Level:   Error,
		Source:  sq.Name,
		Time:    time.Now(),
		Message: err.Error(),
	}
	st.Stop()
}

func (sq *SQLDestination) log(l Logger, level LogLevel, msg string) {
	l.Chan() <- Event{
		Time:    time.Now(),
		Source:  sq.Name,
		Level:   level,
		Message: msg,
	}
}

func (sq *SQLDestination) Open(s Stream, l Logger, st Stopper) {
	sq.manageTx = sq.Tx == nil
	if sq.db == nil {
		err := sq.connect()
		if err != nil {
			sq.fatalerr(err, l, st)
			return
		}
	}
	sq.log(l, Info, "SQL destination opened")
	var (
		tx  *sql.Tx
		err error
	)
	if sq.Tx == nil {
		tx, err = sq.db.Begin()
		sq.log(l, Trace, "Initiated transaction")
		if err != nil {
			sq.fatalerr(err, l, st)
			return
		}
	} else {
		tx = sq.Tx
	}
	var (
		stmt *sql.Stmt
	)
	for msg := range s.Chan(sq.Alias) {
		if st.Stopped() {
			sq.log(l, Warning, "SQL destination aborted")
			if !sq.manageTx {
				return
			}
			err := tx.Rollback()
			if err == nil {
				sq.log(l, Info, "Transaction rolled back")
			} else {
				sq.log(l, Error, fmt.Sprintf("Failed to roll back transaction: %v", err))
			}
			return
		}
		sq.log(l, Trace, fmt.Sprintf("Row %v", msg.Data))
		if len(s.Columns()) != len(msg.Data) {
			sq.fatalerr(fmt.Errorf("expected %v columns but got %v", len(s.Columns()), len(msg.Data)), l, st)
			if !sq.manageTx {
				return
			}
			tx.Rollback() //discard error - best effort attempt
			return
		}
		if stmt == nil {
			sq.columns = s.Columns()
			sq.log(l, Trace, fmt.Sprintf("Found columns %v", sq.columns))
			insertQuery := sq.prepare(s, msg.Data)
			stmt, err = tx.Prepare(insertQuery)
			if err != nil {
				sq.fatalerr(err, l, st)
				if !sq.manageTx {
					return
				}
				tx.Rollback()
				return
			}
		}
		_, err := stmt.Exec(msg.Data...)
		if err != nil {
			sq.fatalerr(err, l, st)
			if !sq.manageTx {
				return
			}
			tx.Rollback()
			return
		}
	}
	if !sq.manageTx {
		return
	}
	sq.log(l, Info, "Done - committing transaction")
	err = tx.Commit()
	if err != nil {
		sq.fatalerr(err, l, st)
	}
}

//prepare creates the prepared statement
func (sq *SQLDestination) prepare(s Stream, msg []interface{}) string {
	cols := strings.Join(s.Columns(), ",")
	params := strings.Repeat("?,", len(msg))
	params = params[0 : len(params)-1] //remove trailing comma
	return fmt.Sprintf(InsertQuery, sq.Table, cols, params)
}
