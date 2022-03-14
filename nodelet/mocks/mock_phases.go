// Code generated by MockGen. DO NOT EDIT.
// Source: ../pkg/phases/phase_interface.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	config "github.com/platform9/nodelet/nodelet/pkg/utils/config"
	v1alpha1 "github.com/platform9/pf9-qbert/sunpike/apiserver/pkg/apis/sunpike/v1alpha1"
)

// MockPhaseInterface is a mock of PhaseInterface interface.
type MockPhaseInterface struct {
	ctrl     *gomock.Controller
	recorder *MockPhaseInterfaceMockRecorder
}

// MockPhaseInterfaceMockRecorder is the mock recorder for MockPhaseInterface.
type MockPhaseInterfaceMockRecorder struct {
	mock *MockPhaseInterface
}

// NewMockPhaseInterface creates a new mock instance.
func NewMockPhaseInterface(ctrl *gomock.Controller) *MockPhaseInterface {
	mock := &MockPhaseInterface{ctrl: ctrl}
	mock.recorder = &MockPhaseInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPhaseInterface) EXPECT() *MockPhaseInterfaceMockRecorder {
	return m.recorder
}

// GetHostPhase mocks base method.
func (m *MockPhaseInterface) GetHostPhase() v1alpha1.HostPhase {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHostPhase")
	ret0, _ := ret[0].(v1alpha1.HostPhase)
	return ret0
}

// GetHostPhase indicates an expected call of GetHostPhase.
func (mr *MockPhaseInterfaceMockRecorder) GetHostPhase() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHostPhase", reflect.TypeOf((*MockPhaseInterface)(nil).GetHostPhase))
}

// GetOrder mocks base method.
func (m *MockPhaseInterface) GetOrder() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOrder")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetOrder indicates an expected call of GetOrder.
func (mr *MockPhaseInterfaceMockRecorder) GetOrder() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOrder", reflect.TypeOf((*MockPhaseInterface)(nil).GetOrder))
}

// GetPhaseName mocks base method.
func (m *MockPhaseInterface) GetPhaseName() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPhaseName")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetPhaseName indicates an expected call of GetPhaseName.
func (mr *MockPhaseInterfaceMockRecorder) GetPhaseName() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPhaseName", reflect.TypeOf((*MockPhaseInterface)(nil).GetPhaseName))
}

// Start mocks base method.
func (m *MockPhaseInterface) Start(arg0 context.Context, arg1 config.Config) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start.
func (mr *MockPhaseInterfaceMockRecorder) Start(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockPhaseInterface)(nil).Start), arg0, arg1)
}

// Status mocks base method.
func (m *MockPhaseInterface) Status(arg0 context.Context, arg1 config.Config) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Status", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Status indicates an expected call of Status.
func (mr *MockPhaseInterfaceMockRecorder) Status(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Status", reflect.TypeOf((*MockPhaseInterface)(nil).Status), arg0, arg1)
}

// Stop mocks base method.
func (m *MockPhaseInterface) Stop(arg0 context.Context, arg1 config.Config) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockPhaseInterfaceMockRecorder) Stop(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockPhaseInterface)(nil).Stop), arg0, arg1)
}
