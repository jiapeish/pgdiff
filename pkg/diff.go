package pkg

import (
	"github.com/jiapeish/pgdiff/pgutil"
)

// Schema is a database definition (table, column, constraint, indes, role, etc) that can be
// added, dropped, or changed to match another database.
type Schema interface {
	Compare(schema interface{}) int
	Add()
	Drop()
	Change(schema interface{})
	NextRow() bool
}

var DbInfo1 pgutil.DbInfo
var DbInfo2 pgutil.DbInfo

/*
 * This is a generic diff function that compares tables, columns, indexes, roles, grants, etc.
 * Different behaviors are specified the Schema implementations
 */
func DoDiff(db1 Schema, db2 Schema) {

	more1 := db1.NextRow()
	more2 := db2.NextRow()
	for more1 || more2 {
		compareVal := db1.Compare(db2)
		if compareVal == 0 {
			// table and column match, look for non-identifying changes
			db1.Change(db2)
			more1 = db1.NextRow()
			more2 = db2.NextRow()
		} else if compareVal < 0 {
			// db2 is missing a value that db1 has
			if more1 {
				db1.Add()
				more1 = db1.NextRow()
			} else {
				// db1 is at the end
				db2.Drop()
				more2 = db2.NextRow()
			}
		} else if compareVal > 0 {
			// db2 has an extra column that we don't want
			if more2 {
				db2.Drop()
				more2 = db2.NextRow()
			} else {
				// db2 is at the end
				db1.Add()
				more1 = db1.NextRow()
			}
		}
	}
}
