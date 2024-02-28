# Secret message sharing platform

[![Build](https://github.com/ivarprudnikov/secret-message-share/actions/workflows/build.yml/badge.svg)](https://github.com/ivarprudnikov/secret-message-share/actions/workflows/build.yml)

Preview live on https://secret-share.azurewebsites.net

Users are able to create an account and store textual content.
This could be links, notes or encoded images.
Once the content is created it can be shared with other internet
users through a unique URL. The visitors to the URL will need 
to enter a PIN to get the content.

## Development

- Test: `go test ./...`
- Run server: `go run .`
- Run server in an Azure function locally: `./scripts/run.sh`
- Build and deploy to Azure: `./scripts/azure.fn.deploy.sh`
- Create required Azure infrastructure: `./scripts/azure.infra.create.sh`

## Models

```
 User { username password(hashed) created_at }
   |
  /|\
Message { username content(encrypted) digest pin(hashed) attempt created_at }
```
