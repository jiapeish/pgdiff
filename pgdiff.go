//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	flag "github.com/jiapeish/pgdiff/pflag"

	_ "github.com/lib/pq"

	"github.com/jiapeish/pgdiff/grant"
	"github.com/jiapeish/pgdiff/pkg"
)

const (
	version = "0.9.3"
)

var (
	args       []string
	schemaType string
)

/*
 * Initialize anything needed later
 */
func init() {
}

/*
 * Do the main logic
 */
func main() {

	var helpPtr = flag.BoolP("help", "?", false, "print help information")
	var versionPtr = flag.BoolP("version", "V", false, "print version information")

	pkg.DbInfo1, pkg.DbInfo2 = pkg.ParseFlags()

	// Remaining args:
	args = flag.Args()

	if *helpPtr {
		usage()
	}

	if *versionPtr {
		fmt.Fprintf(os.Stderr, "%s - version %s\n", os.Args[0], version)
		fmt.Fprintln(os.Stderr, "Copyright (c) 2017 Jon Carlson.  All rights reserved.")
		fmt.Fprintln(os.Stderr, "Use of this source code is governed by the MIT license")
		fmt.Fprintln(os.Stderr, "that can be found here: http://opensource.org/licenses/MIT")
		os.Exit(1)
	}

	if len(args) == 0 {
		fmt.Println("The required first argument is SchemaType: SCHEMA, ROLE, SEQUENCE, TABLE, VIEW, MATVIEW, COLUMN, INDEX, FOREIGN_KEY, OWNER, GRANT_RELATIONSHIP, GRANT_ATTRIBUTE")
		os.Exit(1)
	}

	// Verify schemas
	schemas := pkg.DbInfo1.DbSchema + pkg.DbInfo2.DbSchema
	if schemas != "**" && strings.Contains(schemas, "*") {
		fmt.Println("If one schema is an asterisk, both must be.")
		os.Exit(1)
	}

	schemaType = strings.ToUpper(args[0])
	fmt.Println("-- schemaType:", schemaType)

	fmt.Println("-- db1:", pkg.DbInfo1)
	fmt.Println("-- db2:", pkg.DbInfo2)
	fmt.Println("-- Run the following SQL against db2:")

	conn1, err := pkg.DbInfo1.Open()
	check("opening database 1", err)

	conn2, err := pkg.DbInfo2.Open()
	check("opening database 2", err)

	// This section needs to be improved so that you do not need to choose the type
	// of alter statements to generate.  Rather, all should be generated in the
	// proper order.
	if schemaType == "ALL" {
		if pkg.DbInfo1.DbSchema == "*" {
			pkg.compareSchematas(conn1, conn2)
		}
		pkg.compareSchematas(conn1, conn2)
		pkg.compareRoles(conn1, conn2)
		pkg.compareSequences(conn1, conn2)
		pkg.compareTables(conn1, conn2)
		pkg.compareColumns(conn1, conn2)
		pkg.compareIndexes(conn1, conn2) // includes PK and Unique constraints
		pkg.compareViews(conn1, conn2)
		pkg.compareMatViews(conn1, conn2)
		pkg.compareForeignKeys(conn1, conn2)
		pkg.compareFunctions(conn1, conn2)
		pkg.compareTriggers(conn1, conn2)
		pkg.compareOwners(conn1, conn2)
		grant.compareGrantRelationships(conn1, conn2)
		grant.compareGrantAttributes(conn1, conn2)
	} else if schemaType == "SCHEMA" {
		pkg.compareSchematas(conn1, conn2)
	} else if schemaType == "ROLE" {
		pkg.compareRoles(conn1, conn2)
	} else if schemaType == "SEQUENCE" {
		pkg.compareSequences(conn1, conn2)
	} else if schemaType == "TABLE" {
		pkg.compareTables(conn1, conn2)
	} else if schemaType == "COLUMN" {
		pkg.compareColumns(conn1, conn2)
	} else if schemaType == "TABLE_COLUMN" {
		pkg.compareTableColumns(conn1, conn2)
	} else if schemaType == "INDEX" {
		pkg.compareIndexes(conn1, conn2)
	} else if schemaType == "VIEW" {
		pkg.compareViews(conn1, conn2)
	} else if schemaType == "MATVIEW" {
		pkg.compareMatViews(conn1, conn2)
	} else if schemaType == "FOREIGN_KEY" {
		pkg.compareForeignKeys(conn1, conn2)
	} else if schemaType == "FUNCTION" {
		pkg.compareFunctions(conn1, conn2)
	} else if schemaType == "TRIGGER" {
		pkg.compareTriggers(conn1, conn2)
	} else if schemaType == "OWNER" {
		pkg.compareOwners(conn1, conn2)
	} else if schemaType == "GRANT_RELATIONSHIP" {
		grant.compareGrantRelationships(conn1, conn2)
	} else if schemaType == "GRANT_ATTRIBUTE" {
		grant.compareGrantAttributes(conn1, conn2)
	} else {
		fmt.Println("Not yet handled:", schemaType)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "%s - version %s\n", os.Args[0], version)
	fmt.Fprintf(os.Stderr, "usage: %s [<options>] <schemaType> \n", os.Args[0])
	fmt.Fprintln(os.Stderr, `
Compares the schema between two PostgreSQL databases and generates alter statements 
that can be *manually* run against the second database.

Options:
  -?, --help    : print help information
  -V, --version : print version information
  -v, --verbose : print extra run information
  -U, --user1   : first postgres user 
  -u, --user2   : second postgres user 
  -H, --host1   : first database host.  default is localhost 
  -h, --host2   : second database host. default is localhost 
  -P, --port1   : first port.  default is 5432 
  -p, --port2   : second port. default is 5432 
  -D, --dbname1 : first database name 
  -d, --dbname2 : second database name 
  -S, --schema1 : first schema.  default is all schemas
  -s, --schema2 : second schema. default is all schemas

<schemaTpe> can be: ALL, SCHEMA, ROLE, SEQUENCE, TABLE, TABLE_COLUMN, VIEW, MATVIEW, COLUMN, INDEX, FOREIGN_KEY, OWNER, GRANT_RELATIONSHIP, GRANT_ATTRIBUTE, TRIGGER, FUNCTION`)

	os.Exit(2)
}

func check(msg string, err error) {
	if err != nil {
		log.Fatal("Error "+msg, err)
	}
}
