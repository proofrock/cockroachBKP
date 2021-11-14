package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

const Version = "v0.1.0"

func ckErr(err error) {
	if err != nil {
		panic(err)
	}
}

type Closable interface {
	Close() error
}

func close(thing Closable) {
	ckErr(thing.Close())
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", r)
		}
	}()

	var createSchemas []string
	var createTables []string
	var createViews []string
	var createSequences []string
	var inserts []string

	db, err := sql.Open("postgres", os.Args[1])
	ckErr(err)
	defer close(db)

	var curDB string
	row := db.QueryRow("SELECT current_database()")
	err = row.Scan(&curDB)
	ckErr(err)

	rows, err := db.Query("SHOW SCHEMAS")
	ckErr(err)
	defer close(rows)
	for rows.Next() {
		var schema string
		var owner sql.NullString
		err = rows.Scan(&schema, &owner)
		ckErr(err)
		if owner.Valid && owner.String != "admin" {
			createSchemas = append(createSchemas, fmt.Sprintf("CREATE SCHEMA %s;", schema))
		}
	}
	err = rows.Err()
	ckErr(err)

	var tables []string
	var sequences []string

	rows, err = db.Query("SHOW CREATE ALL TABLES")
	ckErr(err)
	defer close(rows)
	for rows.Next() {
		var create string
		err = rows.Scan(&create)
		ckErr(err)

		elements := strings.Split(create, " ")
		if elements[1] == "TABLE" {
			// if strings.Contains(create, "GENERATED ALWAYS AS IDENTITY") {
			// 	panic("GENERATED ALWAYS AS IDENTITY is not supported. Aborting.")
			// }
			tables = append(tables, elements[2])
			createTables = append(createTables, create)
		} else if elements[1] == "SEQUENCE" {
			// FIXME restore the sequence value, and deal with autoincrements
			sequences = append(sequences, elements[2])
			createSequences = append(createSequences, strings.ReplaceAll(create, "CREATE SEQUENCE", "CREATE SEQUENCE IF NOT EXISTS"))
		} else if elements[1] == "VIEW" {
			createViews = append(createViews, strings.ReplaceAll(create, curDB+".", ""))
		}
	}
	err = rows.Err()
	ckErr(err)

	for _, table := range tables {
		rows, err = db.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", table))
		ckErr(err)
		defer close(rows)
		var columns []string
		for rows.Next() {
			var colName, dataType, gen string
			var nullable, hidden bool
			var deflt sql.NullString
			var index interface{}
			err = rows.Scan(&colName, &dataType, &nullable, &deflt, &gen, &index, &hidden)
			ckErr(err)
			if gen == "" && !hidden {
				columns = append(columns, colName)
			}
		}
		sel := fmt.Sprintf("SELECT \"%s\" FROM %s", strings.Join(columns, "\", \""), table)
		err = rows.Err()
		ckErr(err)

		rows, err = db.Query(sel)
		ckErr(err)
		defer close(rows)
		for rows.Next() {
			values := make([]sql.NullString, len(columns))
			valuePtrs := make([]interface{}, len(values))
			valStr := make([]string, len(values))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			err = rows.Scan(valuePtrs...)
			ckErr(err)

			for idx, value := range values {
				if value.Valid {
					valStr[idx] = "'" + strings.ReplaceAll(value.String, "'", "''") + "'"
				} else {
					valStr[idx] = "NULL"
				}
			}

			inserts = append(inserts, fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s);", table, strings.Join(columns, "\", \""), strings.Join(valStr, ", ")))
		}
		err = rows.Err()
		ckErr(err)
	}

	for _, sequence := range sequences {
		var curVal int64
		row := db.QueryRow(fmt.Sprintf("SELECT nextval('%s')", sequence))
		err = row.Scan(&curVal)
		ckErr(err)
		qry := fmt.Sprintf("SELECT setval('%s', %d, false);", sequence, curVal)
		_, err = db.Exec(qry)
		ckErr(err)
		inserts = append(inserts, qry)
	}

	for _, list := range [][]string{createSchemas, createTables, createSequences, createViews, inserts} {
		for _, output := range list {
			fmt.Println(output)
		}
	}
}
