package db

import (
	"github.com/stretchr/testify/mock"
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) PutRedirect(redirect *RedirectRecord) error {
	args := m.Called(redirect)
	return args.Error(0)
}

func (m *MockDB) GetRedirect(code string) (*RedirectRecord, error) {
	args := m.Called(code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RedirectRecord), args.Error(1)
}