package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

const Version = "v1.0.0"

func main() {
	var createSchemas []string
	var createTables []string
	var createViews []string
	var createSequences []string
	var inserts []string
	db, err := sql.Open("postgres", os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var curDB string
	row := db.QueryRow("SELECT current_database()")
	if err := row.Scan(&curDB); err != nil {
		panic(err)
	}

	rows, err := db.Query("SHOW SCHEMAS")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var schema string
		var owner sql.NullString
		if err := rows.Scan(&schema, &owner); err != nil {
			panic(err)
		}
		if owner.Valid && owner.String != "admin" {
			createSchemas = append(createSchemas, fmt.Sprintf("CREATE SCHEMA %s;", schema))
		}
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}

	var tables []string
	var sequences []string

	rows, err = db.Query("SHOW CREATE ALL TABLES")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var create string
		if err := rows.Scan(&create); err != nil {
			panic(err)
		}

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
	if err = rows.Err(); err != nil {
		panic(err)
	}

	for _, table := range tables {
		rows, err = db.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", table))
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		var columns []string
		for rows.Next() {
			var colName, dataType, gen string
			var nullable, hidden bool
			var deflt sql.NullString
			var index interface{}
			if err := rows.Scan(&colName, &dataType, &nullable, &deflt, &gen, &index, &hidden); err != nil {
				panic(err)
			}
			if gen == "" && !hidden {
				columns = append(columns, colName)
			}
		}
		sel := fmt.Sprintf("SELECT \"%s\" FROM %s", strings.Join(columns, "\", \""), table)
		if err = rows.Err(); err != nil {
			panic(err)
		}

		rows, err = db.Query(sel)
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		for rows.Next() {
			values := make([]sql.NullString, len(columns))
			valuePtrs := make([]interface{}, len(values))
			valStr := make([]string, len(values))
			for i := range columns {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				panic(err)
			}

			for idx, value := range values {
				if value.Valid {
					valStr[idx] = "'" + strings.ReplaceAll(value.String, "'", "''") + "'"
				} else {
					valStr[idx] = "NULL"
				}
			}

			inserts = append(inserts, fmt.Sprintf("INSERT INTO %s (\"%s\") VALUES (%s);", table, strings.Join(columns, "\", \""), strings.Join(valStr, ", ")))
		}
		if err = rows.Err(); err != nil {
			panic(err)
		}
	}

	for _, sequence := range sequences {
		var curVal int64
		row := db.QueryRow(fmt.Sprintf("SELECT nextval('%s')", sequence))
		if err := row.Scan(&curVal); err != nil {
			panic(err)
		}
		qry := fmt.Sprintf("SELECT setval('%s', %d, false);", sequence, curVal)
		if _, err := db.Exec(qry); err != nil {
			panic(err)
		}
		inserts = append(inserts, qry)
	}

	for _, list := range [][]string{createSchemas, createTables, createSequences, createViews, inserts} {
		for _, output := range list {
			fmt.Println(output)
		}
	}
}
