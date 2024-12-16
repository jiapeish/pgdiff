//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
//

package main

import (
	"fmt"
	"os"
	"strings"

	flag "github.com/jiapeish/pgdiff/pflag"
	"github.com/jiapeish/pgdiff/pgutil"

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
	pgutil.Check("opening database 1", err)

	conn2, err := pkg.DbInfo2.Open()
	pgutil.Check("opening database 2", err)

	// This section needs to be improved so that you do not need to choose the type
	// of alter statements to generate.  Rather, all should be generated in the
	// proper order.
	if schemaType == "ALL" {
		if pkg.DbInfo1.DbSchema == "*" {
			pkg.CompareSchematas(conn1, conn2)
		}
		pkg.CompareSchematas(conn1, conn2)
		pkg.CompareRoles(conn1, conn2)
		pkg.CompareSequences(conn1, conn2)
		pkg.CompareTables(conn1, conn2)
		pkg.CompareColumns(conn1, conn2)
		pkg.CompareIndexes(conn1, conn2) // includes PK and Unique constraints
		pkg.CompareViews(conn1, conn2)
		pkg.CompareMatViews(conn1, conn2)
		pkg.CompareForeignKeys(conn1, conn2)
		pkg.CompareFunctions(conn1, conn2)
		pkg.CompareTriggers(conn1, conn2)
		pkg.CompareOwners(conn1, conn2)
		grant.CompareGrantRelationships(conn1, conn2)
		grant.CompareGrantAttributes(conn1, conn2)
	} else if schemaType == "SCHEMA" {
		pkg.CompareSchematas(conn1, conn2)
	} else if schemaType == "ROLE" {
		pkg.CompareRoles(conn1, conn2)
	} else if schemaType == "SEQUENCE" {
		pkg.CompareSequences(conn1, conn2)
	} else if schemaType == "TABLE" {
		pkg.CompareTables(conn1, conn2)
	} else if schemaType == "COLUMN" {
		pkg.CompareColumns(conn1, conn2)
	} else if schemaType == "TABLE_COLUMN" {
		pkg.CompareTableColumns(conn1, conn2)
	} else if schemaType == "INDEX" {
		pkg.CompareIndexes(conn1, conn2)
	} else if schemaType == "VIEW" {
		pkg.CompareViews(conn1, conn2)
	} else if schemaType == "MATVIEW" {
		pkg.CompareMatViews(conn1, conn2)
	} else if schemaType == "FOREIGN_KEY" {
		pkg.CompareForeignKeys(conn1, conn2)
	} else if schemaType == "FUNCTION" {
		pkg.CompareFunctions(conn1, conn2)
	} else if schemaType == "TRIGGER" {
		pkg.CompareTriggers(conn1, conn2)
	} else if schemaType == "OWNER" {
		pkg.CompareOwners(conn1, conn2)
	} else if schemaType == "GRANT_RELATIONSHIP" {
		grant.CompareGrantRelationships(conn1, conn2)
	} else if schemaType == "GRANT_ATTRIBUTE" {
		grant.CompareGrantAttributes(conn1, conn2)
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
