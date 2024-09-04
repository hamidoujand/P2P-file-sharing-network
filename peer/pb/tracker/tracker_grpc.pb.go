// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: tracker.proto

package tracker

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	TrackerService_RegisterPeer_FullMethodName    = "/proto.TrackerService/RegisterPeer"
	TrackerService_UnRegisterPeer_FullMethodName  = "/proto.TrackerService/UnRegisterPeer"
	TrackerService_GetPeers_FullMethodName        = "/proto.TrackerService/GetPeers"
	TrackerService_GetPeersForFile_FullMethodName = "/proto.TrackerService/GetPeersForFile"
	TrackerService_UpdatePeer_FullMethodName      = "/proto.TrackerService/UpdatePeer"
)

// TrackerServiceClient is the client API for TrackerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TrackerServiceClient interface {
	RegisterPeer(ctx context.Context, in *RegisterPeerRequest, opts ...grpc.CallOption) (*RegisterPeerResponse, error)
	UnRegisterPeer(ctx context.Context, in *UnRegisterPeerRequest, opts ...grpc.CallOption) (*UnRegisterPeerResponse, error)
	GetPeers(ctx context.Context, in *GetPeersRequest, opts ...grpc.CallOption) (*GetPeersResponse, error)
	GetPeersForFile(ctx context.Context, in *GetPeersForFileRequest, opts ...grpc.CallOption) (*GetPeersResponse, error)
	UpdatePeer(ctx context.Context, in *UpdatePeerRequest, opts ...grpc.CallOption) (*UpdatePeerResponse, error)
}

type trackerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewTrackerServiceClient(cc grpc.ClientConnInterface) TrackerServiceClient {
	return &trackerServiceClient{cc}
}

func (c *trackerServiceClient) RegisterPeer(ctx context.Context, in *RegisterPeerRequest, opts ...grpc.CallOption) (*RegisterPeerResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RegisterPeerResponse)
	err := c.cc.Invoke(ctx, TrackerService_RegisterPeer_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trackerServiceClient) UnRegisterPeer(ctx context.Context, in *UnRegisterPeerRequest, opts ...grpc.CallOption) (*UnRegisterPeerResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UnRegisterPeerResponse)
	err := c.cc.Invoke(ctx, TrackerService_UnRegisterPeer_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trackerServiceClient) GetPeers(ctx context.Context, in *GetPeersRequest, opts ...grpc.CallOption) (*GetPeersResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetPeersResponse)
	err := c.cc.Invoke(ctx, TrackerService_GetPeers_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trackerServiceClient) GetPeersForFile(ctx context.Context, in *GetPeersForFileRequest, opts ...grpc.CallOption) (*GetPeersResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetPeersResponse)
	err := c.cc.Invoke(ctx, TrackerService_GetPeersForFile_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *trackerServiceClient) UpdatePeer(ctx context.Context, in *UpdatePeerRequest, opts ...grpc.CallOption) (*UpdatePeerResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(UpdatePeerResponse)
	err := c.cc.Invoke(ctx, TrackerService_UpdatePeer_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TrackerServiceServer is the server API for TrackerService service.
// All implementations must embed UnimplementedTrackerServiceServer
// for forward compatibility.
type TrackerServiceServer interface {
	RegisterPeer(context.Context, *RegisterPeerRequest) (*RegisterPeerResponse, error)
	UnRegisterPeer(context.Context, *UnRegisterPeerRequest) (*UnRegisterPeerResponse, error)
	GetPeers(context.Context, *GetPeersRequest) (*GetPeersResponse, error)
	GetPeersForFile(context.Context, *GetPeersForFileRequest) (*GetPeersResponse, error)
	UpdatePeer(context.Context, *UpdatePeerRequest) (*UpdatePeerResponse, error)
	mustEmbedUnimplementedTrackerServiceServer()
}

// UnimplementedTrackerServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedTrackerServiceServer struct{}

func (UnimplementedTrackerServiceServer) RegisterPeer(context.Context, *RegisterPeerRequest) (*RegisterPeerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterPeer not implemented")
}
func (UnimplementedTrackerServiceServer) UnRegisterPeer(context.Context, *UnRegisterPeerRequest) (*UnRegisterPeerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UnRegisterPeer not implemented")
}
func (UnimplementedTrackerServiceServer) GetPeers(context.Context, *GetPeersRequest) (*GetPeersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPeers not implemented")
}
func (UnimplementedTrackerServiceServer) GetPeersForFile(context.Context, *GetPeersForFileRequest) (*GetPeersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPeersForFile not implemented")
}
func (UnimplementedTrackerServiceServer) UpdatePeer(context.Context, *UpdatePeerRequest) (*UpdatePeerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdatePeer not implemented")
}
func (UnimplementedTrackerServiceServer) mustEmbedUnimplementedTrackerServiceServer() {}
func (UnimplementedTrackerServiceServer) testEmbeddedByValue()                        {}

// UnsafeTrackerServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TrackerServiceServer will
// result in compilation errors.
type UnsafeTrackerServiceServer interface {
	mustEmbedUnimplementedTrackerServiceServer()
}

func RegisterTrackerServiceServer(s grpc.ServiceRegistrar, srv TrackerServiceServer) {
	// If the following call pancis, it indicates UnimplementedTrackerServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&TrackerService_ServiceDesc, srv)
}

func _TrackerService_RegisterPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterPeerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrackerServiceServer).RegisterPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TrackerService_RegisterPeer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrackerServiceServer).RegisterPeer(ctx, req.(*RegisterPeerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TrackerService_UnRegisterPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UnRegisterPeerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrackerServiceServer).UnRegisterPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TrackerService_UnRegisterPeer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrackerServiceServer).UnRegisterPeer(ctx, req.(*UnRegisterPeerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TrackerService_GetPeers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPeersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrackerServiceServer).GetPeers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TrackerService_GetPeers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrackerServiceServer).GetPeers(ctx, req.(*GetPeersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TrackerService_GetPeersForFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPeersForFileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrackerServiceServer).GetPeersForFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TrackerService_GetPeersForFile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrackerServiceServer).GetPeersForFile(ctx, req.(*GetPeersForFileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TrackerService_UpdatePeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdatePeerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TrackerServiceServer).UpdatePeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TrackerService_UpdatePeer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TrackerServiceServer).UpdatePeer(ctx, req.(*UpdatePeerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// TrackerService_ServiceDesc is the grpc.ServiceDesc for TrackerService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TrackerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.TrackerService",
	HandlerType: (*TrackerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RegisterPeer",
			Handler:    _TrackerService_RegisterPeer_Handler,
		},
		{
			MethodName: "UnRegisterPeer",
			Handler:    _TrackerService_UnRegisterPeer_Handler,
		},
		{
			MethodName: "GetPeers",
			Handler:    _TrackerService_GetPeers_Handler,
		},
		{
			MethodName: "GetPeersForFile",
			Handler:    _TrackerService_GetPeersForFile_Handler,
		},
		{
			MethodName: "UpdatePeer",
			Handler:    _TrackerService_UpdatePeer_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "tracker.proto",
}
