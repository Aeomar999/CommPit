package storetest

import (
	"testing"

	"github.com/Aeomar999/CommPit/core"
)

func TestMemStoreConformance(t *testing.T) {
	RunStoreTests(t, func() (core.Store, func()) {
		s := NewMemStore()
		return s, func() {}
	})
}
