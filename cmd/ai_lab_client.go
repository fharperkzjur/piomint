
// Package main implements a client for Greeter service.
package main

import (
	"context"
	"log"
	"time"

	pb "github.com/apulis/bmod/ai-lab-backend/pkg/api"
	"google.golang.org/grpc"
)

const (
	address     = "localhost:5567"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewAILabClient(conn)

	for i:= 0;i<100000;i++{
		_,err := testCreateLab(client,2*time.Second)
		if err != nil{
			log.Printf("failed: %v",err)
		}
	}
}

func testCreateLab(client pb.AILabClient,ts time.Duration) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ts)
	defer cancel()

	data,err := client.CreateLab(ctx,&pb.ReqCreateLab{
		Group:     "",
		Name:      "",
		App:       "",
		Type:      "",
		Classify:  "",
		Creator:   "",
		Namespace: "",
		Tags:      nil,
		Meta:      nil,
	})
	return data,err
}