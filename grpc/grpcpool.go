/*
 Copyright 2019 Will Greenberg
 
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

package grpcpool

import (
	"fmt"
	"sync"

	"github.com/vimeo/groupcache"
	"github.com/vimeo/groupcache/consistenthash"
	pb "github.com/vimeo/groupcache/groupcachepb"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const defaultReplicas = 50 // replace with an interface?

// GOOD
type GRPCPool struct {
	self	string
	opts	GRPCPoolOptions
	mu		sync.Mutex
	peers	*consistenthash.Map
	grpcGetters map[string]*grpcGetter
}

// GOOD
type GRPCPoolOptions struct {
	// number of repeated server keys for even consistent hashing among all peers
	Replicas	int

	// defaults to crc32.ChecksumIEEE
	HashFn consistenthash.Hash

	// connection set up configurations for all peers
	PeerDialOptions []grpc.DialOption

	// if true, there will be no TLS
	AllInsecureConnections bool
}

// NOT SURE
// GETTER
type grpcGetter struct {
	address string
	conn    *grpc.ClientConn
}

// GOOD
// NEW GETTER: starts up the connection, throwing an error if it can't be made
func newGRPCGetter(address string, dialOpts ...grpc.DialOption) (*grpcGetter, error) {
	conn, err := grpc.Dial(address, dialOpts...)
	if err != nil {
		fmt.Printf("Failure connecting: [%v]\n", err)
		return nil, fmt.Errorf("Failed to connect to peer at [%s]: %v", address, err)
	}
	fmt.Printf("Success connecting to new Getter at [%s]\n", address)
	return &grpcGetter{address: address, conn: conn}, nil
}

// GOOD
// Initializes a gRPC pool of peers;
// The self argument should be a valid base URL that points to the current server, for example "http://example.net:8000".
func NewGRPCPool(self string, server *grpc.Server) *GRPCPool {
	return NewGRPCOptions(self, server, nil)
}

// GOOD
func NewGRPCOptions(self string, server *grpc.Server, opts *GRPCPoolOptions) *GRPCPool {
	// TODO: figure out a way to ensure GRPCPool hasn't already been made, but don't use a global
	pool := &GRPCPool {
		self:	self,
		grpcGetters: make(map[string]*grpcGetter),
	}

	if opts != nil {
		pool.opts = *opts
	}

	if pool.opts.Replicas == 0 {
		pool.opts.Replicas = defaultReplicas
	}

	// old default: using it to test until I figure out how to set up the security
	if pool.opts.PeerDialOptions == nil {
		pool.opts.PeerDialOptions = []grpc.DialOption{grpc.WithInsecure()}
	}

	if pool.opts.AllInsecureConnections {
		pool.opts.PeerDialOptions = []grpc.DialOption{grpc.WithInsecure()}
	}

	// necessary? Will need to specify credentials
	// if pool.opts.PeerDialOptions == nil {
	// 	pool.opts.PeerDialOptions = []grpc.DialOption{grpc.WithTransportCredentials()}
	// }

	pool.peers = consistenthash.New(pool.opts.Replicas, pool.opts.HashFn)
	groupcache.RegisterPeerPicker(func() groupcache.PeerPicker { return pool })
	pb.RegisterGroupCacheServer(server, pool)
	return pool

}


// NOT SURE
func (gp *GRPCPool) PickPeer(key string) (groupcache.ProtoGetter, bool) {
	gp.mu.Lock()
	defer gp.mu.Unlock()
	if gp.peers.IsEmpty() {
		return nil, false
	}
	if peer := gp.peers.Get(key); peer != gp.self {
		return gp.grpcGetters[peer], true
	}
	return nil, false
}

// NOT SURE
func (gp *GRPCPool) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	group := groupcache.GetGroup(req.Group)
	if group == nil {
		// log.Warnf("Unable to find group [%s]", req.Group)
		return nil, fmt.Errorf("Unable to find group [%s]", req.Group)
	}

	group.Stats.ServerRequests.Add(1)	// keep track of the num of req... was a TODO to remove this?

	var value []byte
	err := group.Get(ctx, req.Key, groupcache.AllocatingByteSliceSink(&value))
	if err != nil {
		// log.WithError(err).Warnf("Failed to retrieve [%s]", req)
		return nil, fmt.Errorf("Failed to retrieve [%s]: %v", req, err)
	}

	return &pb.GetResponse{Value: value}, nil
}

// NOT SURE
func (g *grpcGetter) Get(ctx context.Context, in *pb.GetRequest, out *pb.GetResponse) error {
	client := pb.NewGroupCacheClient(g.conn)
	resp, err := client.Get(context.Background(), &pb.GetRequest{
		Group: in.Group, 
		Key: in.Key})	// passed with an empty Context
	if err != nil {
		return fmt.Errorf("Failed to GET [%s]: %v", in, err)
	}

	out.Value = resp.Value
	return nil
}

// NOT SURE
// SET: sets the peers for the given GRPCPool, can be specified or not (same as HTTPPool); also opens the RPC connections to all new peers
func (gp *GRPCPool) Set(peers ...string) {
	gp.mu.Lock()
	defer gp.mu.Unlock()
	gp.peers = consistenthash.New(gp.opts.Replicas, gp.opts.HashFn)
	tempGetters := make(map[string]*grpcGetter, len(peers))
	for _, peer := range peers {
		fmt.Printf("Peer Address: [%s]\n", peer)
		if getter, exists := gp.grpcGetters[peer]; exists == true {
			tempGetters[peer] = getter
			gp.peers.Add(peer)
			delete(gp.grpcGetters, peer)	// should be able to just set the new getter immediately, no? no need for tempGetters:
			// gp.grpcGetters[peer] = getter
		} else {
			getter, err := newGRPCGetter(peer, gp.opts.PeerDialOptions...)
			if err != nil {
				// TODO: log it
			} else {
				tempGetters[peer] = getter
				gp.peers.Add(peer)
			}
		}
	}

	for p, g := range gp.grpcGetters {
		g.close()
		delete(gp.grpcGetters, p)
	}

	gp.grpcGetters = tempGetters
}

// NOT SURE
func (g *grpcGetter) close() {
	if g.conn != nil {
		g.conn.Close()
	}
}




