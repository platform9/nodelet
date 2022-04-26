// Code generated by MockGen. DO NOT EDIT.
// Source: ../pkg/utils/fileio/fileio.go

// Package mocks is a generated GoMock package.
package mocks

import (
	os "os"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockFileInterface is a mock of FileInterface interface.
type MockFileInterface struct {
	ctrl     *gomock.Controller
	recorder *MockFileInterfaceMockRecorder
}

// MockFileInterfaceMockRecorder is the mock recorder for MockFileInterface.
type MockFileInterfaceMockRecorder struct {
	mock *MockFileInterface
}

// NewMockFileInterface creates a new mock instance.
func NewMockFileInterface(ctrl *gomock.Controller) *MockFileInterface {
	mock := &MockFileInterface{ctrl: ctrl}
	mock.recorder = &MockFileInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFileInterface) EXPECT() *MockFileInterfaceMockRecorder {
	return m.recorder
}

// CopyFile mocks base method.
func (m *MockFileInterface) CopyFile(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CopyFile", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CopyFile indicates an expected call of CopyFile.
func (mr *MockFileInterfaceMockRecorder) CopyFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CopyFile", reflect.TypeOf((*MockFileInterface)(nil).CopyFile), arg0, arg1)
}

// DeleteFile mocks base method.
func (m *MockFileInterface) DeleteFile(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFile", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFile indicates an expected call of DeleteFile.
func (mr *MockFileInterfaceMockRecorder) DeleteFile(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFile", reflect.TypeOf((*MockFileInterface)(nil).DeleteFile), arg0)
}

// GenerateChecksum mocks base method.
func (m *MockFileInterface) GenerateChecksum(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateChecksum", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// GenerateChecksum indicates an expected call of GenerateChecksum.
func (mr *MockFileInterfaceMockRecorder) GenerateChecksum(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateChecksum", reflect.TypeOf((*MockFileInterface)(nil).GenerateChecksum), arg0)
}

// GenerateHashForDir mocks base method.
func (m *MockFileInterface) GenerateHashForDir(arg0 string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateHashForDir", arg0)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateHashForDir indicates an expected call of GenerateHashForDir.
func (mr *MockFileInterfaceMockRecorder) GenerateHashForDir(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateHashForDir", reflect.TypeOf((*MockFileInterface)(nil).GenerateHashForDir), arg0)
}

// GenerateHashForFile mocks base method.
func (m *MockFileInterface) GenerateHashForFile(arg0 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateHashForFile", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateHashForFile indicates an expected call of GenerateHashForFile.
func (mr *MockFileInterfaceMockRecorder) GenerateHashForFile(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateHashForFile", reflect.TypeOf((*MockFileInterface)(nil).GenerateHashForFile), arg0)
}

// GetFileInfo mocks base method.
func (m *MockFileInterface) GetFileInfo(arg0 string) (os.FileInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFileInfo", arg0)
	ret0, _ := ret[0].(os.FileInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFileInfo indicates an expected call of GetFileInfo.
func (mr *MockFileInterfaceMockRecorder) GetFileInfo(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFileInfo", reflect.TypeOf((*MockFileInterface)(nil).GetFileInfo), arg0)
}

// ListFiles mocks base method.
func (m *MockFileInterface) ListFiles(arg0 string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListFiles", arg0)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListFiles indicates an expected call of ListFiles.
func (mr *MockFileInterfaceMockRecorder) ListFiles(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListFiles", reflect.TypeOf((*MockFileInterface)(nil).ListFiles), arg0)
}

// NewYamlFromTemplateYaml mocks base method.
func (m *MockFileInterface) NewYamlFromTemplateYaml(arg0, arg1 string, arg2 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewYamlFromTemplateYaml", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// NewYamlFromTemplateYaml indicates an expected call of NewYamlFromTemplateYaml.
func (mr *MockFileInterfaceMockRecorder) NewYamlFromTemplateYaml(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewYamlFromTemplateYaml", reflect.TypeOf((*MockFileInterface)(nil).NewYamlFromTemplateYaml), arg0, arg1, arg2)
}

// ReadFile mocks base method.
func (m *MockFileInterface) ReadFile(arg0 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFile", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFile indicates an expected call of ReadFile.
func (mr *MockFileInterfaceMockRecorder) ReadFile(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFile", reflect.TypeOf((*MockFileInterface)(nil).ReadFile), arg0)
}

// ReadFileByLine mocks base method.
func (m *MockFileInterface) ReadFileByLine(arg0 string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadFileByLine", arg0)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ReadFileByLine indicates an expected call of ReadFileByLine.
func (mr *MockFileInterfaceMockRecorder) ReadFileByLine(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadFileByLine", reflect.TypeOf((*MockFileInterface)(nil).ReadFileByLine), arg0)
}

// ReadJSONFile mocks base method.
func (m *MockFileInterface) ReadJSONFile(arg0 string, arg1 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReadJSONFile", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// ReadJSONFile indicates an expected call of ReadJSONFile.
func (mr *MockFileInterfaceMockRecorder) ReadJSONFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReadJSONFile", reflect.TypeOf((*MockFileInterface)(nil).ReadJSONFile), arg0, arg1)
}

// RenameAndMoveFile mocks base method.
func (m *MockFileInterface) RenameAndMoveFile(arg0, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RenameAndMoveFile", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// RenameAndMoveFile indicates an expected call of RenameAndMoveFile.
func (mr *MockFileInterfaceMockRecorder) RenameAndMoveFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RenameAndMoveFile", reflect.TypeOf((*MockFileInterface)(nil).RenameAndMoveFile), arg0, arg1)
}

// TouchFile mocks base method.
func (m *MockFileInterface) TouchFile(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TouchFile", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// TouchFile indicates an expected call of TouchFile.
func (mr *MockFileInterfaceMockRecorder) TouchFile(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TouchFile", reflect.TypeOf((*MockFileInterface)(nil).TouchFile), arg0)
}

// VerifyChecksum mocks base method.
func (m *MockFileInterface) VerifyChecksum(arg0 string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VerifyChecksum", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VerifyChecksum indicates an expected call of VerifyChecksum.
func (mr *MockFileInterfaceMockRecorder) VerifyChecksum(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VerifyChecksum", reflect.TypeOf((*MockFileInterface)(nil).VerifyChecksum), arg0)
}

// WriteToFile mocks base method.
func (m *MockFileInterface) WriteToFile(arg0 string, arg1 interface{}, arg2 bool) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteToFile", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteToFile indicates an expected call of WriteToFile.
func (mr *MockFileInterfaceMockRecorder) WriteToFile(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteToFile", reflect.TypeOf((*MockFileInterface)(nil).WriteToFile), arg0, arg1, arg2)
}
