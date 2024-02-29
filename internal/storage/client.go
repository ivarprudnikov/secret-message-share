package storage

import (
	"golang.org/x/net/context"
)

type StorageClient interface {
	ListAll(ctx context.Context, partitionKey string) ([][]byte, error)
	GetOne(ctx context.Context, partitionKey string, rowKey string) ([]byte, error)
	SaveOne(ctx context.Context, partitionKey string, rowKey string, item []byte) error
}
