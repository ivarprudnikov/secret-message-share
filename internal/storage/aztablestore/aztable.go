package aztablestore

import (
	"errors"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
)

// Use default function credentials and use it for the table client
// the expectation is that the function identity has access to the table
func getTableClient(tableName string) (*aztables.Client, error) {
	accountName, ok := os.LookupEnv("AZURE_STORAGE_ACCOUNT")
	if !ok {
		return nil, errors.New("AZURE_STORAGE_ACCOUNT environment variable not found")
	}
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}
	serviceURL := fmt.Sprintf("https://%s.table.core.windows.net/%s", accountName, tableName)
	client, err := aztables.NewClient(serviceURL, cred, nil)
	if err != nil {
		return nil, err
	}
	return client, nil
}
