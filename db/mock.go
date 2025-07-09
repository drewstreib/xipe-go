package db

import (
	"github.com/stretchr/testify/mock"
)

type MockDB struct {
	mock.Mock
}

func (m *MockDB) PutURL(key, url string) error {
	args := m.Called(key, url)
	return args.Error(0)
}

func (m *MockDB) GetURL(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}