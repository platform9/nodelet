// Code generated by MockGen. DO NOT EDIT.
// Source: ../pkg/utils/container_runtime/container_runtime.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	config "github.com/platform9/nodelet/nodelet/pkg/utils/config"
)

// MockRuntime is a mock of Runtime interface.
type MockRuntime struct {
	ctrl     *gomock.Controller
	recorder *MockRuntimeMockRecorder
}

// MockRuntimeMockRecorder is the mock recorder for MockRuntime.
type MockRuntimeMockRecorder struct {
	mock *MockRuntime
}

// NewMockRuntime creates a new mock instance.
func NewMockRuntime(ctrl *gomock.Controller) *MockRuntime {
	mock := &MockRuntime{ctrl: ctrl}
	mock.recorder = &MockRuntimeMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRuntime) EXPECT() *MockRuntimeMockRecorder {
	return m.recorder
}

// EnsureContainerDestroyed mocks base method.
func (m *MockRuntime) EnsureContainerDestroyed(arg0 context.Context, arg1 config.Config, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureContainerDestroyed", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnsureContainerDestroyed indicates an expected call of EnsureContainerDestroyed.
func (mr *MockRuntimeMockRecorder) EnsureContainerDestroyed(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureContainerDestroyed", reflect.TypeOf((*MockRuntime)(nil).EnsureContainerDestroyed), arg0, arg1, arg2)
}

// EnsureContainerStoppedOrNonExistent mocks base method.
func (m *MockRuntime) EnsureContainerStoppedOrNonExistent(arg0 context.Context, arg1 config.Config, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureContainerStoppedOrNonExistent", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnsureContainerStoppedOrNonExistent indicates an expected call of EnsureContainerStoppedOrNonExistent.
func (mr *MockRuntimeMockRecorder) EnsureContainerStoppedOrNonExistent(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureContainerStoppedOrNonExistent", reflect.TypeOf((*MockRuntime)(nil).EnsureContainerStoppedOrNonExistent), arg0, arg1, arg2)
}

// EnsureFreshContainerRunning mocks base method.
func (m *MockRuntime) EnsureFreshContainerRunning(arg0 context.Context, arg1 config.Config, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EnsureFreshContainerRunning", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// EnsureFreshContainerRunning indicates an expected call of EnsureFreshContainerRunning.
func (mr *MockRuntimeMockRecorder) EnsureFreshContainerRunning(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnsureFreshContainerRunning", reflect.TypeOf((*MockRuntime)(nil).EnsureFreshContainerRunning), arg0, arg1, arg2, arg3)
}

// MockImageUtils is a mock of ImageUtils interface.
type MockImageUtils struct {
	ctrl     *gomock.Controller
	recorder *MockImageUtilsMockRecorder
}

// MockImageUtilsMockRecorder is the mock recorder for MockImageUtils.
type MockImageUtilsMockRecorder struct {
	mock *MockImageUtils
}

// NewMockImageUtils creates a new mock instance.
func NewMockImageUtils(ctrl *gomock.Controller) *MockImageUtils {
	mock := &MockImageUtils{ctrl: ctrl}
	mock.recorder = &MockImageUtilsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockImageUtils) EXPECT() *MockImageUtilsMockRecorder {
	return m.recorder
}

// LoadImagesFromDir mocks base method.
func (m *MockImageUtils) LoadImagesFromDir(arg0 context.Context, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadImagesFromDir", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// LoadImagesFromDir indicates an expected call of LoadImagesFromDir.
func (mr *MockImageUtilsMockRecorder) LoadImagesFromDir(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadImagesFromDir", reflect.TypeOf((*MockImageUtils)(nil).LoadImagesFromDir), arg0, arg1, arg2)
}

// LoadImagesFromFile mocks base method.
func (m *MockImageUtils) LoadImagesFromFile(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LoadImagesFromFile", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// LoadImagesFromFile indicates an expected call of LoadImagesFromFile.
func (mr *MockImageUtilsMockRecorder) LoadImagesFromFile(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LoadImagesFromFile", reflect.TypeOf((*MockImageUtils)(nil).LoadImagesFromFile), arg0, arg1)
}
