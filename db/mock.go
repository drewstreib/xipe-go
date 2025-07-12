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

func (m *MockDB) DeleteRedirect(code string, ownerID string) error {
	args := m.Called(code, ownerID)
	return args.Error(0)
}

func (m *MockDB) GetCacheSize() int {
	args := m.Called()
	return args.Int(0)
}

// MockS3 is a mock implementation of S3Interface for testing
type MockS3 struct {
	mock.Mock
}

func (m *MockS3) PutObject(key string, data []byte) error {
	args := m.Called(key, data)
	return args.Error(0)
}

func (m *MockS3) GetObject(key string) ([]byte, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}
