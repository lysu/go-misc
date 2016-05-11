// Code generated by protoc-gen-go.
// source: hi.proto
// DO NOT EDIT!

/*
Package hi is a generated protocol buffer package.

It is generated from these files:
	hi.proto

It has these top-level messages:
	Req
	Resp
*/
package protos

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
const _ = proto.ProtoPackageIsVersion1

type Req struct {
	Name string `protobuf:"bytes,1,opt,name=name" json:"name,omitempty"`
}

func (m *Req) Reset()                    { *m = Req{} }
func (m *Req) String() string            { return proto.CompactTextString(m) }
func (*Req) ProtoMessage()               {}
func (*Req) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type Resp struct {
	Message string `protobuf:"bytes,1,opt,name=message" json:"message,omitempty"`
}

func (m *Resp) Reset()                    { *m = Resp{} }
func (m *Resp) String() string            { return proto.CompactTextString(m) }
func (*Resp) ProtoMessage()               {}
func (*Resp) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func init() {
	proto.RegisterType((*Req)(nil), "hi.Req")
	proto.RegisterType((*Resp)(nil), "hi.Resp")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion2

// Client API for Hier service

type HierClient interface {
	SayHi(ctx context.Context, in *Req, opts ...grpc.CallOption) (*Resp, error)
}

type hierClient struct {
	cc *grpc.ClientConn
}

func NewHierClient(cc *grpc.ClientConn) HierClient {
	return &hierClient{cc}
}

func (c *hierClient) SayHi(ctx context.Context, in *Req, opts ...grpc.CallOption) (*Resp, error) {
	out := new(Resp)
	err := grpc.Invoke(ctx, "/hi.Hier/SayHi", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Hier service

type HierServer interface {
	SayHi(context.Context, *Req) (*Resp, error)
}

func RegisterHierServer(s *grpc.Server, srv HierServer) {
	s.RegisterService(&_Hier_serviceDesc, srv)
}

func _Hier_SayHi_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Req)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HierServer).SayHi(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/hi.Hier/SayHi",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HierServer).SayHi(ctx, req.(*Req))
	}
	return interceptor(ctx, in, info, handler)
}

var _Hier_serviceDesc = grpc.ServiceDesc{
	ServiceName: "hi.Hier",
	HandlerType: (*HierServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SayHi",
			Handler:    _Hier_SayHi_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

var fileDescriptor0 = []byte{
	// 135 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0xe2, 0xe2, 0xc8, 0xc8, 0xd4, 0x2b,
	0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0xca, 0xc8, 0x54, 0x92, 0xe4, 0x62, 0x0e, 0x4a, 0x2d, 0x14,
	0x12, 0xe2, 0x62, 0xc9, 0x4b, 0xcc, 0x4d, 0x95, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x0c, 0x02, 0xb3,
	0x95, 0x14, 0xb8, 0x58, 0x82, 0x52, 0x8b, 0x0b, 0x84, 0x24, 0xb8, 0xd8, 0x73, 0x53, 0x8b, 0x8b,
	0x13, 0xd3, 0x61, 0xd2, 0x30, 0xae, 0x91, 0x0a, 0x17, 0x8b, 0x47, 0x66, 0x6a, 0x91, 0x90, 0x0c,
	0x17, 0x6b, 0x70, 0x62, 0xa5, 0x47, 0xa6, 0x10, 0xbb, 0x1e, 0xd0, 0x70, 0xa0, 0x79, 0x52, 0x1c,
	0x10, 0x46, 0x71, 0x81, 0x12, 0x83, 0x13, 0x3f, 0x17, 0x6f, 0x49, 0x6a, 0x71, 0x89, 0x5e, 0x51,
	0x7e, 0x52, 0x26, 0x88, 0x91, 0xc4, 0x06, 0xb6, 0xde, 0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0x63,
	0x5f, 0xe8, 0x65, 0x8a, 0x00, 0x00, 0x00,
}