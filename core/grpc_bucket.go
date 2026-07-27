package core

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/melon0826/shaniu/core/storage"
	"github.com/melon0826/shaniu/proto3/srpc"
	"github.com/melon0826/shaniu/utils"
)

type ShaniuService struct {
	srpc.UnsafeShaniuServiceServer
}

func (sg *ShaniuService) BucketWatch(stream srpc.ShaniuService_BucketWatchServer) error {
	var watcher func(old, new, key string) *storage.Final
	var echos sync.Map
	defer func() {
		watcher = nil
	}()
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		if watcher == nil {
			watcher = func(old, new, key string) *storage.Final {
				if watcher == nil {
					return nil
				}
				echo := utils.GenUUID()
				ch := make(chan *storage.Final)
				echos.Store(echo, ch)
				defer echos.Delete(echo)
				stream.Send(&srpc.BucketWatchResponse{
					Echo: echo,
					Old:  old,
					Now:  new,
					Key:  key,
				})
				select {
				case v := <-ch:
					return v
				case <-time.After(time.Second * 5):
				}
				return nil
			}
			storage.Watch(MakeBucket(req.Name), req.Key, watcher, req.PluginId)
		} else {
			echo := req.Echo
			v, ok := echos.Load(echo)
			var fn *storage.Final
			if req.Error != "VOID" {
				fn = &storage.Final{
					Now:     req.Now,
					Message: req.Message,
				}
				if req.Error != "" {
					fn.Error = errors.New(req.Error)
				}
			}
			if ok {
				select {
				case v.(chan *storage.Final) <- fn:
				case <-time.After(time.Millisecond):
				}
			}
		}
	}

}

// Get implements BucketServiceServer.Get.
func (sg *ShaniuService) BucketGet(ctx context.Context, req *srpc.BucketKeyRequest) (*srpc.Default, error) {
	if isBackendVersionStorageKey(req.Name, req.Key) {
		return &srpc.Default{}, nil
	}
	value := MakeBucket(req.Name).GetString(req.Key)
	return &srpc.Default{Value: value}, nil
}

// Set implements BucketServiceServer.Set.
func (sg *ShaniuService) BucketSet(ctx context.Context, req *srpc.BucketSetRequest) (*srpc.BucketSetResponse, error) {
	if isBackendVersionStorageKey(req.Name, req.Key) {
		message := "版本信息由后端维护，不允许在存储中修改"
		return &srpc.BucketSetResponse{Changed: false, Message: message}, errors.New(message)
	}
	message, changed, err := MakeBucket(req.Name).Set(req.Key, req.Value)
	return &srpc.BucketSetResponse{Changed: changed, Message: message}, err
}

// Delete implements BucketServiceServer.Delete.
func (sg *ShaniuService) BucketDelete(ctx context.Context, req *srpc.BucketRequest) (*srpc.Empty, error) {
	err := MakeBucket(req.Name).Delete()
	return &srpc.Empty{}, err
}

// Keys implements BucketServiceServer.Keys.
func (sg *ShaniuService) BucketKeys(ctx context.Context, req *srpc.BucketRequest) (*srpc.BucketKeysResponse, error) {
	keys, err := MakeBucket(req.Name).Keys()
	keys = filterBackendVersionStorageKeys(req.Name, keys)
	return &srpc.BucketKeysResponse{Keys: keys}, err
}

// Len implements BucketServiceServer.Len.
func (sg *ShaniuService) BucketLen(ctx context.Context, req *srpc.BucketRequest) (*srpc.LenResponse, error) {
	keys, err := MakeBucket(req.Name).Keys()
	keys = filterBackendVersionStorageKeys(req.Name, keys)
	return &srpc.LenResponse{Length: int32(len(keys))}, err
}

func (sg *ShaniuService) BucketGetAll(ctx context.Context, req *srpc.BucketRequest) (*srpc.Default, error) {
	var values = map[string]string{}
	MakeBucket(req.Name).Foreach(func(b1, b2 []byte) error {
		if isBackendVersionStorageKey(req.Name, string(b1)) {
			return nil
		}
		values[string(b1)] = string(b2)
		return nil
	})
	return &srpc.Default{Value: string(utils.JsonMarshal(values))}, nil
}

// Buckets implements BucketServiceServer.Buckets.
func (sg *ShaniuService) BucketBuckets(ctx context.Context, req *srpc.Empty) (*srpc.BucketsResponse, error) {
	return &srpc.BucketsResponse{Buckets: MakeBucket("app").Buckets()}, nil
}
