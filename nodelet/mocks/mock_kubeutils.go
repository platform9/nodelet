// Code generated by MockGen. DO NOT EDIT.
// Source: ../pkg/utils/kubeutils/kube_utils.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	config "github.com/platform9/nodelet/pkg/utils/config"
	v1 "k8s.io/api/core/v1"
)

// MockUtils is a mock of Utils interface.
type MockUtils struct {
	ctrl     *gomock.Controller
	recorder *MockUtilsMockRecorder
}

// MockUtilsMockRecorder is the mock recorder for MockUtils.
type MockUtilsMockRecorder struct {
	mock *MockUtils
}

// NewMockUtils creates a new mock instance.
func NewMockUtils(ctrl *gomock.Controller) *MockUtils {
	mock := &MockUtils{ctrl: ctrl}
	mock.recorder = &MockUtilsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUtils) EXPECT() *MockUtilsMockRecorder {
	return m.recorder
}

// AddAnnotationsToNode mocks base method.
func (m *MockUtils) AddAnnotationsToNode(arg0 string, arg1 map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddAnnotationsToNode", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddAnnotationsToNode indicates an expected call of AddAnnotationsToNode.
func (mr *MockUtilsMockRecorder) AddAnnotationsToNode(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddAnnotationsToNode", reflect.TypeOf((*MockUtils)(nil).AddAnnotationsToNode), arg0, arg1)
}

// AddLabelsToNode mocks base method.
func (m *MockUtils) AddLabelsToNode(arg0 string, arg1 map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddLabelsToNode", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddLabelsToNode indicates an expected call of AddLabelsToNode.
func (mr *MockUtilsMockRecorder) AddLabelsToNode(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddLabelsToNode", reflect.TypeOf((*MockUtils)(nil).AddLabelsToNode), arg0, arg1)
}

// AddTaintsToNode mocks base method.
func (m *MockUtils) AddTaintsToNode(arg0 string, arg1 []*v1.Taint) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddTaintsToNode", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddTaintsToNode indicates an expected call of AddTaintsToNode.
func (mr *MockUtilsMockRecorder) AddTaintsToNode(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTaintsToNode", reflect.TypeOf((*MockUtils)(nil).AddTaintsToNode), arg0, arg1)
}

// DrainNodeFromApiServer mocks base method.
func (m *MockUtils) DrainNodeFromApiServer(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DrainNodeFromApiServer", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DrainNodeFromApiServer indicates an expected call of DrainNodeFromApiServer.
func (mr *MockUtilsMockRecorder) DrainNodeFromApiServer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DrainNodeFromApiServer", reflect.TypeOf((*MockUtils)(nil).DrainNodeFromApiServer), arg0)
}

// GetNodeFromK8sApi mocks base method.
func (m *MockUtils) GetNodeFromK8sApi(arg0 string) (*v1.Node, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNodeFromK8sApi", arg0)
	ret0, _ := ret[0].(*v1.Node)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNodeFromK8sApi indicates an expected call of GetNodeFromK8sApi.
func (mr *MockUtilsMockRecorder) GetNodeFromK8sApi(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNodeFromK8sApi", reflect.TypeOf((*MockUtils)(nil).GetNodeFromK8sApi), arg0)
}

// KubernetesApiAvailable mocks base method.
func (m *MockUtils) KubernetesApiAvailable(arg0 config.Config) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "KubernetesApiAvailable", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// KubernetesApiAvailable indicates an expected call of KubernetesApiAvailable.
func (mr *MockUtilsMockRecorder) KubernetesApiAvailable(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "KubernetesApiAvailable", reflect.TypeOf((*MockUtils)(nil).KubernetesApiAvailable), arg0)
}

// RemoveAnnotationsFromNode mocks base method.
func (m *MockUtils) RemoveAnnotationsFromNode(arg0 string, arg1 []string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveAnnotationsFromNode", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveAnnotationsFromNode indicates an expected call of RemoveAnnotationsFromNode.
func (mr *MockUtilsMockRecorder) RemoveAnnotationsFromNode(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveAnnotationsFromNode", reflect.TypeOf((*MockUtils)(nil).RemoveAnnotationsFromNode), arg0, arg1)
}

// UncordonNode mocks base method.
func (m *MockUtils) UncordonNode(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UncordonNode", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// UncordonNode indicates an expected call of UncordonNode.
func (mr *MockUtilsMockRecorder) UncordonNode(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UncordonNode", reflect.TypeOf((*MockUtils)(nil).UncordonNode), arg0)
}