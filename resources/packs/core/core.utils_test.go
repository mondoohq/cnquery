package core

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
)

func TestSliceToInterfaceSclice_Strings(t *testing.T) {
	arr := []string{"a", "b", "c"}
	res := SliceToInterfaceSlice(arr)
	assert.IsType(t, []interface{}{}, res)
}

func TestSliceToInterfaceSclice_StringPtrsWithNil(t *testing.T) {
	arr := []*string{ptr.String("a"), nil, ptr.String("c")}
	res := SliceToInterfaceSlice(arr)
	assert.IsType(t, []interface{}{}, res)
	assert.Len(t, res, 2)
}

