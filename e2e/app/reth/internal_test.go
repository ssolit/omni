package reth

import "testing"

func TestDefaultGethConfig(t *testing.T) {
	res := defaultRethConfig()
	t.Logf("Result: %+v", res)
	t.FailNow()
}
