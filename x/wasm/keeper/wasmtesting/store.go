package wasmtesting

import (
	storetypes "cosmossdk.io/store/types"
)

// MockCommitMultiStore mock with a CacheMultiStore to capture commits
type MockCommitMultiStore struct {
	storetypes.CommitMultiStore
	Committed []bool
}

func (m *MockCommitMultiStore) CacheMultiStore() storetypes.CacheMultiStore {
	m.Committed = append(m.Committed, false)
	return &mockCMS{m, &m.Committed[len(m.Committed)-1]}
}

type mockCMS struct {
	storetypes.CommitMultiStore
	committed *bool
}

func (m *mockCMS) Write() {
	*m.committed = true
}

// TODO: This is new function for evm. Need to check if it impact to other feature?
func (m *mockCMS) Copy() storetypes.CacheMultiStore {
	return m.CacheMultiStore()
}
