package sqlxx

import (
	"database/sql"
	"reflect"

	"github.com/proofrock/cockroach_bkp/util"
)

type Scannable interface {
	Scan(...interface{}) error
}

type DB struct {
	Wrapped sql.DB
}

func Wrap(db *sql.DB) *DB {
	return &DB{*db}
}

func (db DB) QRow(qry string, params []interface{}, outputs ...interface{}) (found bool) {
	row := db.Wrapped.QueryRow(qry, params...)
	err := row.Scan(outputs...)
	if err == sql.ErrNoRows {
		return false
	}
	util.CkErr(err)
	return true
}

func (db DB) QRowString(qry string, params ...interface{}) (res string, found bool) {
	var ret string
	row := db.Wrapped.QueryRow(qry, params...)
	err := row.Scan(&ret)
	if err == sql.ErrNoRows {
		return "", false
	}
	util.CkErr(err)
	return ret, true
}

func (db DB) QRowInt(qry string, params ...interface{}) (res int, found bool) {
	var ret int
	row := db.Wrapped.QueryRow(qry, params...)
	err := row.Scan(&ret)
	if err == sql.ErrNoRows {
		return 0, false
	}
	util.CkErr(err)
	return ret, true
}

func (db DB) QRowInt64(qry string, params ...interface{}) (res int64, found bool) {
	var ret int64
	row := db.Wrapped.QueryRow(qry, params...)
	err := row.Scan(&ret)
	if err == sql.ErrNoRows {
		return 0, false
	}
	util.CkErr(err)
	return ret, true
}

func (db DB) QRowStruct(qry string, dest interface{}, params ...interface{}) (found bool) {
	numOfFields := reflect.TypeOf(dest).Elem().NumField()
	interfaces := make([]interface{}, numOfFields)
	for i := 0; i < numOfFields; i++ {
		interfaces[i] = reflect.ValueOf(dest).Elem().Field(i).Addr().Interface()
	}
	err := db.Wrapped.QueryRow(qry, params...).Scan(interfaces...)
	if err == sql.ErrNoRows {
		return false
	}
	util.CkErr(err)
	return true
}

func (db DB) QRows(qry string, lambda func(row Scannable) (stop bool, err error), params ...interface{}) (found int) {
	rows, err := db.Wrapped.Query(qry, params...)
	util.CkErr(err)
	defer util.Close(rows)
	for rows.Next() {
		found++
		stop, err := lambda(rows)
		util.CkErr(err)
		if stop {
			break
		}
	}
	util.CkErr(rows.Err())
	return
}

func (db DB) QRowsAsStrings(qry string, params ...interface{}) []string {
	rows, err := db.Wrapped.Query(qry, params...)
	util.CkErr(err)
	defer util.Close(rows)
	var ret []string
	for rows.Next() {
		var item string
		util.CkErr(rows.Scan(&item))
		ret = append(ret, item)
	}
	util.CkErr(rows.Err())
	return ret
}

func (db DB) QExec(qry string, params ...interface{}) {
	_, err := db.Wrapped.Exec(qry, params...)
	util.CkErr(err)
}
