// Code generated by "stringer -type=Operation"; DO NOT EDIT.

package xdiff

import "strconv"

const _Operation_name = "InsertUpdateDeleteInsertSubtreeDeleteSubtree"

var _Operation_index = [...]uint8{0, 6, 12, 18, 31, 44}

func (i Operation) String() string {
	i -= 1
	if i < 0 || i >= Operation(len(_Operation_index)-1) {
		return "Operation(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _Operation_name[_Operation_index[i]:_Operation_index[i+1]]
}