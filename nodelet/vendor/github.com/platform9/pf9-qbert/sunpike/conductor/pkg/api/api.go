package api

import (
	"google.golang.org/grpc"
)

type Client interface {
	ConductorClient
	ConductorAdminClient
}

type Server interface {
	ConductorServer
	ConductorAdminServer
}

type client struct {
	ConductorClient
	ConductorAdminClient
}

func NewClient(conn *grpc.ClientConn) Client {
	return &client{
		ConductorClient:      NewConductorClient(conn),
		ConductorAdminClient: NewConductorAdminClient(conn),
	}
}
