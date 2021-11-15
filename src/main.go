package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/proofrock/cockroach_bkp/util"
	"github.com/proofrock/cockroach_bkp/util/sqlxx"
)

const Version = "v0.1.0"

func walkLists(block func(str string), lists ...[]string) {
	for _, list := range lists {
		for _, output := range list {
			fmt.Println(output)
		}
	}
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
	util.CkErr(err)
	dbxx := sqlxx.Wrap(db)
	defer util.Close(db)

	curDB, _ := dbxx.QRowString("SELECT current_database()")
	curUser, _ := dbxx.QRowString("SELECT current_user")

	dbxx.QRows("SHOW SCHEMAS", func(row sqlxx.Scannable) (stop bool, err error) {
		var schema string
		var owner sql.NullString
		if err = row.Scan(&schema, &owner); err != nil {
			return true, err
		}
		if owner.Valid && owner.String == curUser {
			createSchemas = append(createSchemas, fmt.Sprintf("CREATE SCHEMA %s;", schema))
		}
		return
	})

	var tables []string
	var sequences []string

	dbxx.QRows("SHOW CREATE ALL TABLES", func(row sqlxx.Scannable) (stop bool, err error) {
		var create string
		if err = row.Scan(&create); err != nil {
			return true, err
		}
		elements := strings.Split(create, " ")
		if elements[1] == "TABLE" {
			if strings.Contains(create, "GENERATED ALWAYS AS IDENTITY") {
				return true, errors.New("GENERATED ALWAYS AS IDENTITY is not supported. Aborting.")
			}
			tables = append(tables, elements[2])
			createTables = append(createTables, create)
		} else if elements[1] == "SEQUENCE" {
			// FIXME restore the sequence value, and deal with autoincrements
			sequences = append(sequences, elements[2])
			createSequences = append(createSequences, strings.ReplaceAll(create, "CREATE SEQUENCE", "CREATE SEQUENCE IF NOT EXISTS"))
		} else if elements[1] == "VIEW" {
			createViews = append(createViews, strings.ReplaceAll(create, curDB+".", ""))
		}
		return
	})

	for _, table := range tables {
		var columns []string
		dbxx.QRows(fmt.Sprintf("SHOW COLUMNS FROM %s", table), func(row sqlxx.Scannable) (stop bool, err error) {
			var noop interface{}
			var colName, gen string
			var hidden bool
			if err = row.Scan(&colName, &noop, &noop, &noop, &gen, &noop, &hidden); err != nil {
				return true, err
			}
			if gen == "" && !hidden {
				columns = append(columns, colName)
			}
			return
		})

		sel := fmt.Sprintf("SELECT \"%s\" FROM %s", strings.Join(columns, "\", \""), table)
		dbxx.QRows(sel, func(row sqlxx.Scannable) (stop bool, err error) {
			values := make([]sql.NullString, len(columns))
			valuePtrs := make([]interface{}, len(values))
			valStr := make([]string, len(values))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			if err = row.Scan(valuePtrs...); err != nil {
				return true, err
			}

			for idx, value := range values {
				if value.Valid {
					valStr[idx] = "'" + strings.ReplaceAll(value.String, "'", "''") + "'"
				} else {
					valStr[idx] = "NULL"
				}
			}

			sql := fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s);", table, strings.Join(columns, "\", \""), strings.Join(valStr, ", "))
			inserts = append(inserts, sql)
			return
		})
	}

	for _, sequence := range sequences {
		curVal, _ := dbxx.QRowInt64(fmt.Sprintf("SELECT nextval('%s')", sequence))

		qry := fmt.Sprintf("SELECT setval('%s', %d, false);", sequence, curVal)
		dbxx.QExec(qry)
		inserts = append(inserts, qry)
	}

	walkLists(func(str string) {
		fmt.Println(str)
	}, createSchemas, createTables, createSequences, createViews, inserts)
}
