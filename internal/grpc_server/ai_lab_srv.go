/* ******************************************************************************
* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
*
* This program and the accompanying materials are made available under the
* terms of the MIT License, which is available at
* https://www.opensource.org/licenses/MIT
*
* See the NOTICE file distributed with this work for additional
* information regarding copyright ownership.
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
* License for the specific language governing permissions and limitations
* under the License.
*
* SPDX-License-Identifier: MIT
******************************************************************************/
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
