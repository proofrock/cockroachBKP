package main

import (
	"database/sql"
	"reflect"
)

func ckErr(err error) {
	if err != nil {
		panic(err)
	}
}

type Closable interface {
	Close() error
}

func Close(thing Closable) {
	ckErr(thing.Close())
}

type Scannable interface {
	Scan(...interface{}) error
}

func QRow(db *sql.DB, qry string, params []interface{}, outputs ...interface{}) (found bool) {
	row := db.QueryRow(qry, params...)
	err := row.Scan(outputs...)
	if err == sql.ErrNoRows {
		return false
	}
	ckErr(err)
	return true
}

func QRowString(db *sql.DB, qry string, params ...interface{}) (res string, found bool) {
	var ret string
	row := db.QueryRow(qry, params...)
	err := row.Scan(&ret)
	if err == sql.ErrNoRows {
		return "", false
	}
	ckErr(err)
	return ret, true
}

func QRowInt(db *sql.DB, qry string, params ...interface{}) (res int, found bool) {
	var ret int
	row := db.QueryRow(qry, params...)
	err := row.Scan(&ret)
	if err == sql.ErrNoRows {
		return 0, false
	}
	ckErr(err)
	return ret, true
}

func QRowInt64(db *sql.DB, qry string, params ...interface{}) (res int64, found bool) {
	var ret int64
	row := db.QueryRow(qry, params...)
	err := row.Scan(&ret)
	if err == sql.ErrNoRows {
		return 0, false
	}
	ckErr(err)
	return ret, true
}

func QRowStruct(db *sql.DB, qry string, dest interface{}) (found bool) {
	numOfFields := reflect.TypeOf(dest).Elem().NumField()
	interfaces := make([]interface{}, numOfFields)
	for i := 0; i < numOfFields; i++ {
		interfaces[i] = reflect.ValueOf(dest).Elem().Field(i).Addr().Interface()
	}
	err := db.QueryRow(qry).Scan(interfaces...)
	if err == sql.ErrNoRows {
		return false
	}
	ckErr(err)
	return true
}

func QRows(db *sql.DB, qry string, params []interface{}, lambda func(row Scannable) (stop bool, err error)) (found int) {
	rows, err := db.Query(qry, params...)
	ckErr(err)
	defer Close(rows)
	for rows.Next() {
		found++
		stop, err := lambda(rows)
		ckErr(err)
		if stop {
			break
		}
	}
	ckErr(rows.Err())
	return
}

func QRowsAsStrings(db *sql.DB, qry string, params ...interface{}) []string {
	rows, err := db.Query(qry, params...)
	ckErr(err)
	defer Close(rows)
	var ret []string
	for rows.Next() {
		var item string
		ckErr(rows.Scan(&item))
		ret = append(ret, item)
	}
	ckErr(rows.Err())
	return ret
}

func QExec(db *sql.DB, qry string, params []interface{}) {
	_, err := db.Exec(qry, params...)
	ckErr(err)
}
