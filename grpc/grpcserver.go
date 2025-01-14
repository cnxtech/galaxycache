/*
 Copyright 2019 Vimeo Inc.

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

package grpc

import (
	"context"

	gc "github.com/vimeo/galaxycache"
	pb "github.com/vimeo/galaxycache/galaxycachepb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// serviceImpl implements the GalaxyCacheServer
// interface generated by the GalaxyCache pb service
type serviceImpl struct {
	universe *gc.Universe
}

// RegisterGRPCServer registers the given grpc.Server with
// a Universe for GetFromPeer calls over RPC
func RegisterGRPCServer(universe *gc.Universe, grpcServer *grpc.Server) {
	pb.RegisterGalaxyCacheServer(grpcServer, &serviceImpl{universe: universe})
}

// GetFromPeer implements the generated GalaxyCacheServer
// interface, making an internal Get() after receiving a
// remote call from a peer
func (gp *serviceImpl) GetFromPeer(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	galaxy := gp.universe.GetGalaxy(req.Galaxy)
	if galaxy == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Unable to find galaxy [%s]", req.Galaxy)
	}

	galaxy.Stats.ServerRequests.Add(1) // keep track of the num of req

	var value gc.ByteCodec
	err := galaxy.Get(ctx, req.Key, &value)
	if err != nil {
		return nil, status.Errorf(status.Code(err), "Failed to retrieve [%s]: %v", req, err)
	}

	return &pb.GetResponse{Value: value}, nil
}
