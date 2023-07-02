/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"

	expp "sample-grpc-app/experiment"
)

var (
	port = flag.Int("port", 50051, "The server port")
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

type fakeListener struct {
	actualListener net.Listener
}

type fakeConn struct {
	actualConn net.Conn
}

func (f fakeConn) Read(b []byte) (n int, err error) {
	n, err = f.actualConn.Read(b)
	fmt.Printf("Read initiated from the client with data \n%v\n\n", string(b))
	return n, err
}

func (f fakeConn) Write(b []byte) (n int, err error) {
	fmt.Printf("Writing data to client \n%v\n\n", b)
	buf := bytes.NewBuffer(b)
	framer := http2.NewFramer(buf, buf)
	for {
		n := expp.HeaderFields(framer)
		if n != 1 {
			break
		}
	}
	n, err = f.actualConn.Write(b)
	if err != nil {
		fmt.Printf("Error encountered while writing data to client: %v\n", err)
	}
	return n, err
}

func (f fakeConn) Close() error {
	fmt.Println("Closing client connection....")
	return f.actualConn.Close()
}

func (f fakeConn) LocalAddr() net.Addr {
	return f.actualConn.LocalAddr()
}

func (f fakeConn) RemoteAddr() net.Addr {
	return f.actualConn.RemoteAddr()
}

func (f fakeConn) SetDeadline(t time.Time) error {
	return f.actualConn.SetDeadline(t)
}

func (f fakeConn) SetReadDeadline(t time.Time) error {
	return f.actualConn.SetReadDeadline(t)
}

func (f fakeConn) SetWriteDeadline(t time.Time) error {
	return f.actualConn.SetWriteDeadline(t)
}

func (f fakeListener) Accept() (net.Conn, error) {
	netConn, err := f.actualListener.Accept()
	return fakeConn{actualConn: netConn}, err
}

func (f fakeListener) Addr() net.Addr {
	return f.actualListener.Addr()
}
func (f fakeListener) Close() error {
	return f.actualListener.Close()
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
