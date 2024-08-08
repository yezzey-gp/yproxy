// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/storage/storage.go

// Package mock is a generated GoMock package.
package mock

import (
	io "io"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	storage "github.com/yezzey-gp/yproxy/pkg/storage"
)

// MockStorageReader is a mock of StorageReader interface.
type MockStorageReader struct {
	ctrl     *gomock.Controller
	recorder *MockStorageReaderMockRecorder
}

// MockStorageReaderMockRecorder is the mock recorder for MockStorageReader.
type MockStorageReaderMockRecorder struct {
	mock *MockStorageReader
}

// NewMockStorageReader creates a new mock instance.
func NewMockStorageReader(ctrl *gomock.Controller) *MockStorageReader {
	mock := &MockStorageReader{ctrl: ctrl}
	mock.recorder = &MockStorageReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageReader) EXPECT() *MockStorageReaderMockRecorder {
	return m.recorder
}

// CatFileFromStorage mocks base method.
func (m *MockStorageReader) CatFileFromStorage(name string, offset int64) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CatFileFromStorage", name, offset)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CatFileFromStorage indicates an expected call of CatFileFromStorage.
func (mr *MockStorageReaderMockRecorder) CatFileFromStorage(name, offset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CatFileFromStorage", reflect.TypeOf((*MockStorageReader)(nil).CatFileFromStorage), name, offset)
}

// MockStorageWriter is a mock of StorageWriter interface.
type MockStorageWriter struct {
	ctrl     *gomock.Controller
	recorder *MockStorageWriterMockRecorder
}

// MockStorageWriterMockRecorder is the mock recorder for MockStorageWriter.
type MockStorageWriterMockRecorder struct {
	mock *MockStorageWriter
}

// NewMockStorageWriter creates a new mock instance.
func NewMockStorageWriter(ctrl *gomock.Controller) *MockStorageWriter {
	mock := &MockStorageWriter{ctrl: ctrl}
	mock.recorder = &MockStorageWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageWriter) EXPECT() *MockStorageWriterMockRecorder {
	return m.recorder
}

// PatchFile mocks base method.
func (m *MockStorageWriter) PatchFile(name string, r io.ReadSeeker, startOffset int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PatchFile", name, r, startOffset)
	ret0, _ := ret[0].(error)
	return ret0
}

// PatchFile indicates an expected call of PatchFile.
func (mr *MockStorageWriterMockRecorder) PatchFile(name, r, startOffset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PatchFile", reflect.TypeOf((*MockStorageWriter)(nil).PatchFile), name, r, startOffset)
}

// PutFileToDest mocks base method.
func (m *MockStorageWriter) PutFileToDest(name string, r io.Reader) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PutFileToDest", name, r)
	ret0, _ := ret[0].(error)
	return ret0
}

// PutFileToDest indicates an expected call of PutFileToDest.
func (mr *MockStorageWriterMockRecorder) PutFileToDest(name, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutFileToDest", reflect.TypeOf((*MockStorageWriter)(nil).PutFileToDest), name, r)
}

// MockStorageLister is a mock of StorageLister interface.
type MockStorageLister struct {
	ctrl     *gomock.Controller
	recorder *MockStorageListerMockRecorder
}

// MockStorageListerMockRecorder is the mock recorder for MockStorageLister.
type MockStorageListerMockRecorder struct {
	mock *MockStorageLister
}

// NewMockStorageLister creates a new mock instance.
func NewMockStorageLister(ctrl *gomock.Controller) *MockStorageLister {
	mock := &MockStorageLister{ctrl: ctrl}
	mock.recorder = &MockStorageListerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageLister) EXPECT() *MockStorageListerMockRecorder {
	return m.recorder
}

// ListPath mocks base method.
func (m *MockStorageLister) ListPath(prefix string) ([]*storage.ObjectInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListPath", prefix)
	ret0, _ := ret[0].([]*storage.ObjectInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListPath indicates an expected call of ListPath.
func (mr *MockStorageListerMockRecorder) ListPath(prefix interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListPath", reflect.TypeOf((*MockStorageLister)(nil).ListPath), prefix)
}

// MockStorageMover is a mock of StorageMover interface.
type MockStorageMover struct {
	ctrl     *gomock.Controller
	recorder *MockStorageMoverMockRecorder
}

// MockStorageMoverMockRecorder is the mock recorder for MockStorageMover.
type MockStorageMoverMockRecorder struct {
	mock *MockStorageMover
}

// NewMockStorageMover creates a new mock instance.
func NewMockStorageMover(ctrl *gomock.Controller) *MockStorageMover {
	mock := &MockStorageMover{ctrl: ctrl}
	mock.recorder = &MockStorageMoverMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageMover) EXPECT() *MockStorageMoverMockRecorder {
	return m.recorder
}

// DeleteObject mocks base method.
func (m *MockStorageMover) DeleteObject(key string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteObject", key)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteObject indicates an expected call of DeleteObject.
func (mr *MockStorageMoverMockRecorder) DeleteObject(key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteObject", reflect.TypeOf((*MockStorageMover)(nil).DeleteObject), key)
}

// MoveObject mocks base method.
func (m *MockStorageMover) MoveObject(from, to string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MoveObject", from, to)
	ret0, _ := ret[0].(error)
	return ret0
}

// MoveObject indicates an expected call of MoveObject.
func (mr *MockStorageMoverMockRecorder) MoveObject(from, to interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MoveObject", reflect.TypeOf((*MockStorageMover)(nil).MoveObject), from, to)
}

// MockStorageInteractor is a mock of StorageInteractor interface.
type MockStorageInteractor struct {
	ctrl     *gomock.Controller
	recorder *MockStorageInteractorMockRecorder
}

// MockStorageInteractorMockRecorder is the mock recorder for MockStorageInteractor.
type MockStorageInteractorMockRecorder struct {
	mock *MockStorageInteractor
}

// NewMockStorageInteractor creates a new mock instance.
func NewMockStorageInteractor(ctrl *gomock.Controller) *MockStorageInteractor {
	mock := &MockStorageInteractor{ctrl: ctrl}
	mock.recorder = &MockStorageInteractorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageInteractor) EXPECT() *MockStorageInteractorMockRecorder {
	return m.recorder
}

// CatFileFromStorage mocks base method.
func (m *MockStorageInteractor) CatFileFromStorage(name string, offset int64) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CatFileFromStorage", name, offset)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CatFileFromStorage indicates an expected call of CatFileFromStorage.
func (mr *MockStorageInteractorMockRecorder) CatFileFromStorage(name, offset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CatFileFromStorage", reflect.TypeOf((*MockStorageInteractor)(nil).CatFileFromStorage), name, offset)
}

// DeleteObject mocks base method.
func (m *MockStorageInteractor) DeleteObject(key string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteObject", key)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteObject indicates an expected call of DeleteObject.
func (mr *MockStorageInteractorMockRecorder) DeleteObject(key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteObject", reflect.TypeOf((*MockStorageInteractor)(nil).DeleteObject), key)
}

// ListPath mocks base method.
func (m *MockStorageInteractor) ListPath(prefix string) ([]*storage.ObjectInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListPath", prefix)
	ret0, _ := ret[0].([]*storage.ObjectInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListPath indicates an expected call of ListPath.
func (mr *MockStorageInteractorMockRecorder) ListPath(prefix interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListPath", reflect.TypeOf((*MockStorageInteractor)(nil).ListPath), prefix)
}

// MoveObject mocks base method.
func (m *MockStorageInteractor) MoveObject(from, to string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MoveObject", from, to)
	ret0, _ := ret[0].(error)
	return ret0
}

// MoveObject indicates an expected call of MoveObject.
func (mr *MockStorageInteractorMockRecorder) MoveObject(from, to interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MoveObject", reflect.TypeOf((*MockStorageInteractor)(nil).MoveObject), from, to)
}

// PatchFile mocks base method.
func (m *MockStorageInteractor) PatchFile(name string, r io.ReadSeeker, startOffset int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PatchFile", name, r, startOffset)
	ret0, _ := ret[0].(error)
	return ret0
}

// PatchFile indicates an expected call of PatchFile.
func (mr *MockStorageInteractorMockRecorder) PatchFile(name, r, startOffset interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PatchFile", reflect.TypeOf((*MockStorageInteractor)(nil).PatchFile), name, r, startOffset)
}

// PutFileToDest mocks base method.
func (m *MockStorageInteractor) PutFileToDest(name string, r io.Reader) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PutFileToDest", name, r)
	ret0, _ := ret[0].(error)
	return ret0
}

// PutFileToDest indicates an expected call of PutFileToDest.
func (mr *MockStorageInteractorMockRecorder) PutFileToDest(name, r interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PutFileToDest", reflect.TypeOf((*MockStorageInteractor)(nil).PutFileToDest), name, r)
}