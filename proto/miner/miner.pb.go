// Code generated by protoc-gen-go. DO NOT EDIT.
// source: miner.proto

/*
Package miner is a generated protocol buffer package.

It is generated from these files:
	miner.proto

It has these top-level messages:
	PingRequest
	PingReply
	InfoRequest
	InfoReply
	HandshakeRequest
	HandshakeReply
	StartRequest
	StartReply
	StopRequest
	StopReply
*/
package miner

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
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type PingRequest struct {
}

func (m *PingRequest) Reset()                    { *m = PingRequest{} }
func (m *PingRequest) String() string            { return proto.CompactTextString(m) }
func (*PingRequest) ProtoMessage()               {}
func (*PingRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type PingReply struct {
	Status string `protobuf:"bytes,1,opt,name=status" json:"status,omitempty"`
}

func (m *PingReply) Reset()                    { *m = PingReply{} }
func (m *PingReply) String() string            { return proto.CompactTextString(m) }
func (*PingReply) ProtoMessage()               {}
func (*PingReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *PingReply) GetStatus() string {
	if m != nil {
		return m.Status
	}
	return ""
}

type InfoRequest struct {
}

func (m *InfoRequest) Reset()                    { *m = InfoRequest{} }
func (m *InfoRequest) String() string            { return proto.CompactTextString(m) }
func (*InfoRequest) ProtoMessage()               {}
func (*InfoRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

type InfoReply struct {
	Stats map[string]*InfoReplyStats `protobuf:"bytes,1,rep,name=Stats" json:"Stats,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *InfoReply) Reset()                    { *m = InfoReply{} }
func (m *InfoReply) String() string            { return proto.CompactTextString(m) }
func (*InfoReply) ProtoMessage()               {}
func (*InfoReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *InfoReply) GetStats() map[string]*InfoReplyStats {
	if m != nil {
		return m.Stats
	}
	return nil
}

type InfoReplyStats struct {
	CPU    *InfoReplyStatsCpu    `protobuf:"bytes,1,opt,name=CPU" json:"CPU,omitempty"`
	Memory *InfoReplyStatsMemory `protobuf:"bytes,2,opt,name=Memory" json:"Memory,omitempty"`
}

func (m *InfoReplyStats) Reset()                    { *m = InfoReplyStats{} }
func (m *InfoReplyStats) String() string            { return proto.CompactTextString(m) }
func (*InfoReplyStats) ProtoMessage()               {}
func (*InfoReplyStats) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3, 0} }

func (m *InfoReplyStats) GetCPU() *InfoReplyStatsCpu {
	if m != nil {
		return m.CPU
	}
	return nil
}

func (m *InfoReplyStats) GetMemory() *InfoReplyStatsMemory {
	if m != nil {
		return m.Memory
	}
	return nil
}

type InfoReplyStatsCpu struct {
	TotalUsage uint64 `protobuf:"varint,1,opt,name=totalUsage" json:"totalUsage,omitempty"`
}

func (m *InfoReplyStatsCpu) Reset()                    { *m = InfoReplyStatsCpu{} }
func (m *InfoReplyStatsCpu) String() string            { return proto.CompactTextString(m) }
func (*InfoReplyStatsCpu) ProtoMessage()               {}
func (*InfoReplyStatsCpu) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3, 0, 0} }

func (m *InfoReplyStatsCpu) GetTotalUsage() uint64 {
	if m != nil {
		return m.TotalUsage
	}
	return 0
}

type InfoReplyStatsMemory struct {
	MaxUsage uint64 `protobuf:"varint,1,opt,name=maxUsage" json:"maxUsage,omitempty"`
}

func (m *InfoReplyStatsMemory) Reset()                    { *m = InfoReplyStatsMemory{} }
func (m *InfoReplyStatsMemory) String() string            { return proto.CompactTextString(m) }
func (*InfoReplyStatsMemory) ProtoMessage()               {}
func (*InfoReplyStatsMemory) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3, 0, 1} }

func (m *InfoReplyStatsMemory) GetMaxUsage() uint64 {
	if m != nil {
		return m.MaxUsage
	}
	return 0
}

type HandshakeRequest struct {
	Hub string `protobuf:"bytes,1,opt,name=hub" json:"hub,omitempty"`
}

func (m *HandshakeRequest) Reset()                    { *m = HandshakeRequest{} }
func (m *HandshakeRequest) String() string            { return proto.CompactTextString(m) }
func (*HandshakeRequest) ProtoMessage()               {}
func (*HandshakeRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *HandshakeRequest) GetHub() string {
	if m != nil {
		return m.Hub
	}
	return ""
}

type HandshakeReply struct {
	Miner string `protobuf:"bytes,1,opt,name=miner" json:"miner,omitempty"`
}

func (m *HandshakeReply) Reset()                    { *m = HandshakeReply{} }
func (m *HandshakeReply) String() string            { return proto.CompactTextString(m) }
func (*HandshakeReply) ProtoMessage()               {}
func (*HandshakeReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *HandshakeReply) GetMiner() string {
	if m != nil {
		return m.Miner
	}
	return ""
}

type StartRequest struct {
	Id       string `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Registry string `protobuf:"bytes,2,opt,name=registry" json:"registry,omitempty"`
	Image    string `protobuf:"bytes,3,opt,name=image" json:"image,omitempty"`
}

func (m *StartRequest) Reset()                    { *m = StartRequest{} }
func (m *StartRequest) String() string            { return proto.CompactTextString(m) }
func (*StartRequest) ProtoMessage()               {}
func (*StartRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

func (m *StartRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *StartRequest) GetRegistry() string {
	if m != nil {
		return m.Registry
	}
	return ""
}

func (m *StartRequest) GetImage() string {
	if m != nil {
		return m.Image
	}
	return ""
}

type StartReply struct {
	Container string                     `protobuf:"bytes,1,opt,name=container" json:"container,omitempty"`
	Ports     map[string]*StartReplyPort `protobuf:"bytes,2,rep,name=Ports" json:"Ports,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *StartReply) Reset()                    { *m = StartReply{} }
func (m *StartReply) String() string            { return proto.CompactTextString(m) }
func (*StartReply) ProtoMessage()               {}
func (*StartReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

func (m *StartReply) GetContainer() string {
	if m != nil {
		return m.Container
	}
	return ""
}

func (m *StartReply) GetPorts() map[string]*StartReplyPort {
	if m != nil {
		return m.Ports
	}
	return nil
}

type StartReplyPort struct {
	IP   string `protobuf:"bytes,1,opt,name=IP" json:"IP,omitempty"`
	Port string `protobuf:"bytes,2,opt,name=port" json:"port,omitempty"`
}

func (m *StartReplyPort) Reset()                    { *m = StartReplyPort{} }
func (m *StartReplyPort) String() string            { return proto.CompactTextString(m) }
func (*StartReplyPort) ProtoMessage()               {}
func (*StartReplyPort) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7, 0} }

func (m *StartReplyPort) GetIP() string {
	if m != nil {
		return m.IP
	}
	return ""
}

func (m *StartReplyPort) GetPort() string {
	if m != nil {
		return m.Port
	}
	return ""
}

type StopRequest struct {
	Id string `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
}

func (m *StopRequest) Reset()                    { *m = StopRequest{} }
func (m *StopRequest) String() string            { return proto.CompactTextString(m) }
func (*StopRequest) ProtoMessage()               {}
func (*StopRequest) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{8} }

func (m *StopRequest) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

type StopReply struct {
}

func (m *StopReply) Reset()                    { *m = StopReply{} }
func (m *StopReply) String() string            { return proto.CompactTextString(m) }
func (*StopReply) ProtoMessage()               {}
func (*StopReply) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{9} }

func init() {
	proto.RegisterType((*PingRequest)(nil), "miner.PingRequest")
	proto.RegisterType((*PingReply)(nil), "miner.PingReply")
	proto.RegisterType((*InfoRequest)(nil), "miner.InfoRequest")
	proto.RegisterType((*InfoReply)(nil), "miner.InfoReply")
	proto.RegisterType((*InfoReplyStats)(nil), "miner.InfoReply.stats")
	proto.RegisterType((*InfoReplyStatsCpu)(nil), "miner.InfoReply.stats.cpu")
	proto.RegisterType((*InfoReplyStatsMemory)(nil), "miner.InfoReply.stats.memory")
	proto.RegisterType((*HandshakeRequest)(nil), "miner.HandshakeRequest")
	proto.RegisterType((*HandshakeReply)(nil), "miner.HandshakeReply")
	proto.RegisterType((*StartRequest)(nil), "miner.StartRequest")
	proto.RegisterType((*StartReply)(nil), "miner.StartReply")
	proto.RegisterType((*StartReplyPort)(nil), "miner.StartReply.port")
	proto.RegisterType((*StopRequest)(nil), "miner.StopRequest")
	proto.RegisterType((*StopReply)(nil), "miner.StopReply")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Miner service

type MinerClient interface {
	Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingReply, error)
	Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoReply, error)
	Handshake(ctx context.Context, in *HandshakeRequest, opts ...grpc.CallOption) (*HandshakeReply, error)
	Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*StartReply, error)
	Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*StopReply, error)
}

type minerClient struct {
	cc *grpc.ClientConn
}

func NewMinerClient(cc *grpc.ClientConn) MinerClient {
	return &minerClient{cc}
}

func (c *minerClient) Ping(ctx context.Context, in *PingRequest, opts ...grpc.CallOption) (*PingReply, error) {
	out := new(PingReply)
	err := grpc.Invoke(ctx, "/miner.Miner/Ping", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *minerClient) Info(ctx context.Context, in *InfoRequest, opts ...grpc.CallOption) (*InfoReply, error) {
	out := new(InfoReply)
	err := grpc.Invoke(ctx, "/miner.Miner/Info", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *minerClient) Handshake(ctx context.Context, in *HandshakeRequest, opts ...grpc.CallOption) (*HandshakeReply, error) {
	out := new(HandshakeReply)
	err := grpc.Invoke(ctx, "/miner.Miner/Handshake", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *minerClient) Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*StartReply, error) {
	out := new(StartReply)
	err := grpc.Invoke(ctx, "/miner.Miner/Start", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *minerClient) Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*StopReply, error) {
	out := new(StopReply)
	err := grpc.Invoke(ctx, "/miner.Miner/Stop", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Miner service

type MinerServer interface {
	Ping(context.Context, *PingRequest) (*PingReply, error)
	Info(context.Context, *InfoRequest) (*InfoReply, error)
	Handshake(context.Context, *HandshakeRequest) (*HandshakeReply, error)
	Start(context.Context, *StartRequest) (*StartReply, error)
	Stop(context.Context, *StopRequest) (*StopReply, error)
}

func RegisterMinerServer(s *grpc.Server, srv MinerServer) {
	s.RegisterService(&_Miner_serviceDesc, srv)
}

func _Miner_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MinerServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/miner.Miner/Ping",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MinerServer).Ping(ctx, req.(*PingRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Miner_Info_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MinerServer).Info(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/miner.Miner/Info",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MinerServer).Info(ctx, req.(*InfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Miner_Handshake_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(HandshakeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MinerServer).Handshake(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/miner.Miner/Handshake",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MinerServer).Handshake(ctx, req.(*HandshakeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Miner_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MinerServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/miner.Miner/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MinerServer).Start(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Miner_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MinerServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/miner.Miner/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MinerServer).Stop(ctx, req.(*StopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Miner_serviceDesc = grpc.ServiceDesc{
	ServiceName: "miner.Miner",
	HandlerType: (*MinerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _Miner_Ping_Handler,
		},
		{
			MethodName: "Info",
			Handler:    _Miner_Info_Handler,
		},
		{
			MethodName: "Handshake",
			Handler:    _Miner_Handshake_Handler,
		},
		{
			MethodName: "Start",
			Handler:    _Miner_Start_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _Miner_Stop_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "miner.proto",
}

func init() { proto.RegisterFile("miner.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 518 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x54, 0xe1, 0x8a, 0xd3, 0x40,
	0x10, 0x6e, 0x93, 0xa6, 0x98, 0x89, 0x1e, 0x75, 0xd5, 0xb3, 0xc4, 0x3b, 0x91, 0x78, 0xca, 0x21,
	0x47, 0xc0, 0x8a, 0x20, 0x82, 0xbf, 0x44, 0xb0, 0x3f, 0x0e, 0x42, 0xca, 0x3d, 0xc0, 0xb6, 0x5d,
	0xdb, 0x70, 0x49, 0x36, 0x6e, 0x36, 0x62, 0x1e, 0xc1, 0x47, 0xf2, 0x1d, 0x7c, 0x04, 0x1f, 0x46,
	0x66, 0x77, 0xbb, 0xdd, 0xb3, 0x1e, 0xf8, 0x2f, 0xb3, 0xfb, 0x7d, 0xdf, 0xcc, 0x7c, 0x3b, 0x13,
	0x88, 0xaa, 0xa2, 0x66, 0x22, 0x6d, 0x04, 0x97, 0x9c, 0x04, 0x2a, 0x48, 0xee, 0x41, 0x94, 0x15,
	0xf5, 0x26, 0x67, 0x5f, 0x3b, 0xd6, 0xca, 0xe4, 0x39, 0x84, 0x3a, 0x6c, 0xca, 0x9e, 0x1c, 0xc3,
	0xb8, 0x95, 0x54, 0x76, 0xed, 0x74, 0xf8, 0x6c, 0x78, 0x1e, 0xe6, 0x26, 0x42, 0xce, 0xbc, 0xfe,
	0xc2, 0x77, 0x9c, 0x5f, 0x1e, 0x84, 0x3a, 0x46, 0xd2, 0x6b, 0x08, 0x16, 0x92, 0x4a, 0xe4, 0xf8,
	0xe7, 0xd1, 0xec, 0x49, 0xaa, 0x93, 0x5a, 0x40, 0xaa, 0x6e, 0x3f, 0xd5, 0x52, 0xf4, 0xb9, 0x46,
	0xc6, 0x3f, 0x87, 0x10, 0xa0, 0x74, 0x4b, 0x2e, 0xc0, 0xff, 0x98, 0x5d, 0xa9, 0x74, 0xd1, 0x2c,
	0x3e, 0xa0, 0x2a, 0x50, 0xba, 0x6a, 0xba, 0x1c, 0x61, 0xe4, 0x2d, 0x8c, 0x2f, 0x59, 0xc5, 0x45,
	0x3f, 0xf5, 0x14, 0xe1, 0xf4, 0x16, 0x42, 0xa5, 0x40, 0xb9, 0x01, 0xc7, 0x2f, 0xc0, 0x5f, 0x35,
	0x1d, 0x79, 0x0a, 0x20, 0xb9, 0xa4, 0xe5, 0x55, 0x4b, 0x37, 0x4c, 0xa5, 0x1c, 0xe5, 0xce, 0x49,
	0x7c, 0x06, 0x63, 0x4d, 0x24, 0x31, 0xdc, 0xa9, 0xe8, 0x77, 0x17, 0x67, 0xe3, 0x38, 0x03, 0xd8,
	0x37, 0x44, 0x26, 0xe0, 0x5f, 0xb3, 0xde, 0xd8, 0x85, 0x9f, 0xe4, 0x02, 0x82, 0x6f, 0xb4, 0xec,
	0x98, 0x29, 0xf1, 0xf8, 0xdf, 0x25, 0xe6, 0x1a, 0xf4, 0xde, 0x7b, 0x37, 0x4c, 0xce, 0x60, 0xf2,
	0x99, 0xd6, 0xeb, 0x76, 0x4b, 0xaf, 0x99, 0xb1, 0x18, 0x75, 0xb7, 0xdd, 0x72, 0xa7, 0xbb, 0xed,
	0x96, 0xc9, 0x4b, 0x38, 0x72, 0x50, 0x68, 0xfc, 0x43, 0xd0, 0x4f, 0x6a, 0x50, 0xe6, 0x7d, 0x33,
	0xb8, 0xbb, 0x90, 0x54, 0xc8, 0x9d, 0xd2, 0x11, 0x78, 0xc5, 0xda, 0x40, 0xbc, 0x62, 0x8d, 0xbd,
	0x09, 0xb6, 0x29, 0x5a, 0x69, 0x5c, 0x0c, 0x73, 0x1b, 0xa3, 0x62, 0x51, 0x61, 0xd3, 0xbe, 0x56,
	0x54, 0x41, 0xf2, 0x7b, 0xa8, 0x5a, 0x46, 0x49, 0x4c, 0x7b, 0x02, 0xe1, 0x8a, 0xd7, 0x92, 0x3a,
	0xa9, 0xf7, 0x07, 0x64, 0x06, 0x41, 0xc6, 0x85, 0x6c, 0xa7, 0x9e, 0x9a, 0x86, 0x13, 0xd3, 0xfe,
	0x9e, 0x9f, 0xaa, 0x6b, 0x33, 0x0e, 0xea, 0x3b, 0x7e, 0x05, 0xa3, 0x86, 0x0b, 0x55, 0xea, 0x3c,
	0xdb, 0x95, 0x3a, 0xcf, 0x08, 0xd1, 0xe7, 0xa6, 0x4c, 0xf5, 0x8d, 0xf6, 0xef, 0x05, 0xfe, 0xdf,
	0x7e, 0x27, 0x3f, 0xca, 0xb8, 0xf6, 0x9f, 0x42, 0xb4, 0x90, 0xbc, 0xb9, 0xc5, 0xaf, 0x24, 0x82,
	0x50, 0x5f, 0x37, 0x65, 0x3f, 0xfb, 0xe1, 0x41, 0x70, 0xa9, 0xfa, 0x4c, 0x61, 0x84, 0x7b, 0x43,
	0x88, 0x49, 0xe0, 0xec, 0x54, 0x3c, 0xb9, 0x71, 0xd6, 0x94, 0x7d, 0x32, 0x40, 0x3c, 0x8e, 0x80,
	0xc5, 0x3b, 0xfb, 0x64, 0xf1, 0x76, 0x46, 0x92, 0x01, 0xf9, 0x00, 0xa1, 0x7d, 0x6e, 0xf2, 0xd8,
	0x00, 0xfe, 0x1e, 0x93, 0xf8, 0xd1, 0xe1, 0x85, 0xa6, 0xeb, 0xa5, 0x14, 0x92, 0x3c, 0xb8, 0x69,
	0x80, 0xa6, 0xdd, 0x3f, 0x70, 0x45, 0x57, 0x88, 0x8d, 0xda, 0x0a, 0x1d, 0x53, 0x6c, 0x85, 0xd6,
	0x89, 0x64, 0xb0, 0x1c, 0xab, 0xdf, 0xca, 0x9b, 0x3f, 0x01, 0x00, 0x00, 0xff, 0xff, 0x5a, 0x40,
	0x6c, 0x26, 0x65, 0x04, 0x00, 0x00,
}
