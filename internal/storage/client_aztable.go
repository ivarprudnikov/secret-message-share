package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func cleanStr(input string) string {
	return nonAlphanumericRegex.ReplaceAllString(input, "")
}

type tableClient struct {
	instance *aztables.Client
}

func NewTableClient(accountName string, tableName string) (StorageClient, error) {
	if accountName == "" {
		return nil, errors.New("storage account name is required")
	}
	if tableName == "" {
		return nil, errors.New("table name is required")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("could not obtain default azure credential: %w", err)
	}
	serviceURL := fmt.Sprintf("https://%s.table.core.windows.net/%s", accountName, tableName)
	client, err := aztables.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create tables client: %w", err)
	}
	return &tableClient{instance: client}, nil
}

func (tc *tableClient) ListAll(ctx context.Context, partitionKey string) ([][]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	filter := fmt.Sprintf("PartitionKey eq %s", cleanStr(partitionKey))
	pager := tc.instance.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Filter: &filter,
	})
	var all [][]byte
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			return all, fmt.Errorf("failed to list entries: %w", err)
		}
		all = append(all, response.Entities...)
	}

	return all, nil
}

func (tc *tableClient) GetOne(ctx context.Context, partitionKey string, rowKey string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	response, err := tc.instance.GetEntity(ctx, partitionKey, rowKey, nil)
	if err != nil {
		var azErr *azcore.ResponseError
		if errors.As(err, &azErr) && azErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get entry: %w", err)
	}
	return response.Value, nil
}

func (tc *tableClient) SaveOne(ctx context.Context, partitionKey string, rowKey string, item []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}
	_, err := tc.instance.AddEntity(ctx, item, nil)
	if err != nil {
		return fmt.Errorf("failed to save entry: %w", err)
	}
	return nil
}
