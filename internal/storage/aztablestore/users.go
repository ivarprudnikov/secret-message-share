package aztablestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/ivarprudnikov/secretshare/internal/crypto"
	"github.com/ivarprudnikov/secretshare/internal/storage"
)

type azUserStore struct {
	accountName string
	tableName   string
	salt        string
}

func NewAzUserStore(accountName, tableName, salt string) storage.UserStore {
	return &azUserStore{accountName: accountName, tableName: tableName, salt: salt}
}

func (u *azUserStore) getClient() (*aztables.Client, error) {
	return getTableClient(u.accountName, u.tableName)
}

// aztables does not have a way to query the size of the table
// need to scan through all of the records :(
func (u *azUserStore) CountUsers(ctx context.Context) (int64, error) {
	var count int64 = 0
	client, err := u.getClient()
	if err != nil {
		return count, fmt.Errorf("failed to get aztable client: %w", err)
	}
	keySelector := "PartitionKey"
	metadataFormat := aztables.MetadataFormatNone
	listPager := client.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Select: &keySelector,
		Format: &metadataFormat,
	})
	for listPager.More() {
		response, err := listPager.NextPage(ctx)
		if err != nil {
			return count, fmt.Errorf("next page request failed: %w", err)
		}
		count += int64(len(response.Entities))
	}
	return count, nil
}

func (u *azUserStore) AddUser(ctx context.Context, username string, password string, permissions []string) (*storage.User, error) {
	existing, err := u.GetUser(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if existing != nil {
		return nil, errors.New("username is not available")
	}
	usr, err := storage.NewUser(username, password, permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	marshalled, err := json.Marshal(usr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user: %w", err)
	}
	client, err := u.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get aztable client: %w", err)
	}
	resp, err := client.AddEntity(ctx, marshalled, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}
	var saved *storage.User
	err = json.Unmarshal(resp.Value, &saved)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}
	return saved, nil
}

func (u *azUserStore) GetUser(ctx context.Context, username string) (*storage.User, error) {
	client, err := u.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get aztable client: %w", err)
	}
	resp, err := client.GetEntity(ctx, username, username, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user entity: %w", err)
	}
	if resp.Value == nil {
		return nil, nil
	}
	var user *storage.User
	err = json.Unmarshal(resp.Value, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}
	return user, nil
}

func (u *azUserStore) GetUserWithPass(ctx context.Context, username string, password string) (*storage.User, error) {
	user, err := u.GetUser(ctx, username)
	if err != nil {
		return nil, err
	}
	// even if user is not found evaluate the password
	// this will reduce the effect on time difference
	// at the time of the login check
	hashedPass := "unknown"
	if user != nil {
		hashedPass = user.Password
	}
	if err := crypto.CompareHashToPass(hashedPass, password); err == nil {
		return user, nil
	}
	return nil, nil
}
