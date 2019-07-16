/*
 Copyright 2019 Vimeo Inc.
 Adapted from https://github.com/charithe/gcgrpcpool/blob/master/gcgrpcpool.go

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package galaxycache

import (
	"context"
	"fmt"

	pb "github.com/vimeo/galaxycache/galaxycachepb"
	"google.golang.org/grpc"
)

// GRPCGalaxyCacheServer implements the GalaxyCacheServer
// interface generated by the GalaxyCache pb service
type GRPCGalaxyCacheServer struct {
	universe *Universe
}

// RegisterGRPCServer registers the given grpc.Server with
// a Universe for GetFromPeer calls over RPC
func RegisterGRPCServer(universe *Universe, grpcServer *grpc.Server) {
	pb.RegisterGalaxyCacheServer(grpcServer, &GRPCGalaxyCacheServer{universe: universe})
}

// GetFromPeer implements the generated GalaxyCacheServer
// interface, making an internal Get() after receiving a
// remote call from a peer
func (gp *GRPCGalaxyCacheServer) GetFromPeer(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	group := gp.universe.GetGalaxy(req.Galaxy)
	if group == nil {
		return nil, fmt.Errorf("Unable to find group [%s]", req.Galaxy)
	}

	group.Stats.ServerRequests.Add(1) // keep track of the num of req

	var value ByteCodec
	err := group.Get(ctx, req.Key, &value)
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve [%s]: %v", req, err)
	}

	return &pb.GetResponse{Value: value}, nil
}
