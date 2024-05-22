package toolbox

import (
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools

	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Errorf("wrong length. wanted=%d, got=%d", 10, len(s))
	}

}
