package main

import "database/sql"

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

func QExec(db *sql.DB, qry string, params []interface{}) {
	_, err := db.Exec(qry, params...)
	ckErr(err)
}
