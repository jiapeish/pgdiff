//
// Copyright (c) 2017 Jon Carlson.  All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.
//

package grant

import (
	"bytes"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/jiapeish/pgdiff/pgutil"
	"github.com/jiapeish/pgdiff/pkg"
)

var (
	grantAttributeSqlTemplate = initGrantAttributeSqlTemplate()
)

// Initializes the Sql template
func initGrantAttributeSqlTemplate() *template.Template {
	sql := `
-- Attribute/Column ACL only
SELECT
  n.nspname AS schema_name
  , {{ if eq $.DbSchema "*" }}n.nspname::text || '.' || {{ end }}c.relkind::text  || '.' || c.relname::text || '.' || a.attname AS compare_name
  , CASE c.relkind
    WHEN 'r' THEN 'TABLE'
    WHEN 'v' THEN 'VIEW'
    WHEN 'f' THEN 'FOREIGN TABLE'
    END as type
  , c.relname AS relationship_name
  , a.attname AS attribute_name
  , a.attacl  AS attribute_acl
FROM pg_catalog.pg_class c
LEFT JOIN pg_catalog.pg_namespace n ON (n.oid = c.relnamespace)
INNER JOIN (SELECT attname, unnest(attacl) AS attacl, attrelid
           FROM pg_catalog.pg_attribute
           WHERE NOT attisdropped AND attacl IS NOT NULL)
      AS a ON (a.attrelid = c.oid)
WHERE c.relkind IN ('r', 'v', 'f')
--AND pg_catalog.pg_table_is_visible(c.oid)
{{ if eq $.DbSchema "*" }}
AND n.nspname NOT LIKE 'pg_%'
AND n.nspname <> 'information_schema'
{{ else }}
AND n.nspname = '{{ $.DbSchema }}'
{{ end }};
`

	t := template.New("GrantAttributeSqlTmpl")
	template.Must(t.Parse(sql))
	return t
}

// ==================================
// GrantAttributeRows definition
// ==================================

// GrantAttributeRows is a sortable slice of string maps
type GrantAttributeRows []map[string]string

func (slice GrantAttributeRows) Len() int {
	return len(slice)
}

func (slice GrantAttributeRows) Less(i, j int) bool {
	if slice[i]["compare_name"] != slice[j]["compare_name"] {
		return slice[i]["compare_name"] < slice[j]["compare_name"]
	}

	// Only compare the role part of the ACL
	// Not yet sure if this is absolutely necessary
	// (or if we could just compare the entire ACL string)
	role1, _ := parseAcl(slice[i]["attribute_acl"])
	role2, _ := parseAcl(slice[j]["attribute_acl"])
	if role1 != role2 {
		return role1 < role2
	}

	return false
}

func (slice GrantAttributeRows) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

// ==================================
// GrantAttributeSchema definition
// (implements Schema -- defined in pgdiff.go)
// ==================================

// GrantAttributeSchema holds a slice of rows from one of the databases as well as
// a reference to the current row of data we're viewing.
type GrantAttributeSchema struct {
	rows   GrantAttributeRows
	rowNum int
	done   bool
}

// get returns the value from the current row for the given key
func (c *GrantAttributeSchema) get(key string) string {
	if c.rowNum >= len(c.rows) {
		return ""
	}
	return c.rows[c.rowNum][key]
}

// get returns the current row for the given key
func (c *GrantAttributeSchema) getRow() map[string]string {
	if c.rowNum >= len(c.rows) {
		return make(map[string]string)
	}
	return c.rows[c.rowNum]
}

// NextRow increments the rowNum and tells you whether or not there are more
func (c *GrantAttributeSchema) NextRow() bool {
	if c.rowNum >= len(c.rows)-1 {
		c.done = true
	}
	c.rowNum = c.rowNum + 1
	return !c.done
}

// Compare tells you, in one pass, whether or not the first row matches, is less than,
// or greater than the second row.
func (c *GrantAttributeSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*GrantAttributeSchema)
	if !ok {
		fmt.Println("Error!!!, Compare needs a GrantAttributeSchema instance", c2)
		return +999
	}

	val := pgutil.CompareStrings(c.get("compare_name"), c2.get("compare_name"))
	if val != 0 {
		return val
	}

	role1, _ := parseAcl(c.get("attribute_acl"))
	role2, _ := parseAcl(c2.get("attribute_acl"))
	val = pgutil.CompareStrings(role1, role2)
	return val
}

// Add prints SQL to add the grant
func (c *GrantAttributeSchema) Add() {
	schema := pkg.DbInfo2.DbSchema
	if schema == "*" {
		schema = c.get("schema_name")
	}

	role, grants := parseGrants(c.get("attribute_acl"))
	fmt.Printf("GRANT %s (%s) ON %s.%s TO %s; -- Add\n", strings.Join(grants, ", "), c.get("attribute_name"), schema, c.get("relationship_name"), role)
}

// Drop prints SQL to drop the grant
func (c *GrantAttributeSchema) Drop() {
	role, grants := parseGrants(c.get("attribute_acl"))
	fmt.Printf("REVOKE %s (%s) ON %s.%s FROM %s; -- Drop\n", strings.Join(grants, ", "), c.get("attribute_name"), c.get("schema_name"), c.get("relationship_name"), role)
}

// Change handles the case where the relationship and column match, but the grant does not
func (c *GrantAttributeSchema) Change(obj interface{}) {
	c2, ok := obj.(*GrantAttributeSchema)
	if !ok {
		fmt.Println("-- Error!!!, Change needs a GrantAttributeSchema instance", c2)
	}

	role, grants1 := parseGrants(c.get("attribute_acl"))
	_, grants2 := parseGrants(c2.get("attribute_acl"))

	// Find grants in the first db that are not in the second
	// (for this relationship and owner)
	var grantList []string
	for _, g := range grants1 {
		if !pgutil.ContainsString(grants2, g) {
			grantList = append(grantList, g)
		}
	}
	if len(grantList) > 0 {
		fmt.Printf("GRANT %s (%s) ON %s.%s TO %s; -- Change\n", strings.Join(grantList, ", "),
			c.get("attribute_name"), c2.get("schema_name"), c.get("relationship_name"), role)
	}

	// Find grants in the second db that are not in the first
	// (for this relationship and owner)
	var revokeList []string
	for _, g := range grants2 {
		if !pgutil.ContainsString(grants1, g) {
			revokeList = append(revokeList, g)
		}
	}
	if len(revokeList) > 0 {
		fmt.Printf("REVOKE %s (%s) ON %s.%s FROM %s; -- Change\n", strings.Join(revokeList, ", "), c.get("attribute_name"), c2.get("schema_name"), c.get("relationship_name"), role)
	}

	//fmt.Printf("--1 rel:%s, relAcl:%s, col:%s, colAcl:%s\n", c.get("attribute_name"), c.get("attribute_acl"), c.get("attribute_name"), c.get("attribute_acl"))
	//fmt.Printf("--2 rel:%s, relAcl:%s, col:%s, colAcl:%s\n", c2.get("attribute_name"), c2.get("attribute_acl"), c2.get("attribute_name"), c2.get("attribute_acl"))
}

// ==================================
// Functions
// ==================================

// compareGrantAttributes outputs SQL to make the granted permissions match between DBs or schemas
func CompareGrantAttributes(conn1 *sql.DB, conn2 *sql.DB) {

	buf1 := new(bytes.Buffer)
	grantAttributeSqlTemplate.Execute(buf1, pkg.DbInfo1)

	buf2 := new(bytes.Buffer)
	grantAttributeSqlTemplate.Execute(buf2, pkg.DbInfo2)

	rowChan1, _ := pgutil.QueryStrings(conn1, buf1.String())
	rowChan2, _ := pgutil.QueryStrings(conn2, buf2.String())

	rows1 := make(GrantAttributeRows, 0)
	for row := range rowChan1 {
		rows1 = append(rows1, row)
	}
	sort.Sort(rows1)
	//for _, row := range rows1 {
	//fmt.Printf("--1b compare:%s, col:%s, colAcl:%s\n", row["compare_name"], row["attribute_name"], row["attribute_acl"])
	//}

	rows2 := make(GrantAttributeRows, 0)
	for row := range rowChan2 {
		rows2 = append(rows2, row)
	}
	sort.Sort(rows2)
	//for _, row := range rows2 {
	//fmt.Printf("--2b compare:%s, col:%s, colAcl:%s\n", row["compare_name"], row["attribute_name"], row["attribute_acl"])
	//}

	// We have to explicitly type this as Schema here for some unknown reason
	var schema1 pkg.Schema = &GrantAttributeSchema{rows: rows1, rowNum: -1}
	var schema2 pkg.Schema = &GrantAttributeSchema{rows: rows2, rowNum: -1}

	pkg.DoDiff(schema1, schema2)
}
