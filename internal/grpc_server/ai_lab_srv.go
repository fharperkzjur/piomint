
package grpc_server

import (
	"context"
	pb "github.com/apulis/bmod/ai-lab-backend/pkg/api"
	_ "github.com/pkg/errors"
)

type AILabServerImpl struct{
	pb.UnimplementedAILabServer
}



func (*AILabServerImpl) CreateLab(ctx context.Context, req*pb.ReqCreateLab) (*pb.ReplyHeader, error){

	  return &pb.ReplyHeader{
		  Code:   12345,
		  Msg:    "this is a test request!!!",
		  Detail: "1234567",
	  },nil
}
