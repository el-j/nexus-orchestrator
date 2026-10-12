package repo_sqlite

import (
	"database/sql"
	"encoding/json"
)

// affected returns the number of rows a statement changed. The SQLite driver
// reports this from the statement itself and never fails, so there is no error
// path to handle.
func affected(res sql.Result) int64 {
	n, _ := res.RowsAffected()
	return n
}

// marshal encodes a slice of strings, a map of strings or a plain struct. Those
// cannot fail to encode, so there is no error to return.
func marshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// marshalTags encodes a task's tags, storing "[]" (not "null") when there are none.
func marshalTags(tags []string) []byte {
	if len(tags) == 0 {
		return []byte("[]")
	}
	return marshal(tags)
}
