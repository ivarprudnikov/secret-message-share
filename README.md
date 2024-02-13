# Secret message sharing platform

Users are able to create an account and store textual content.
This could be links, notes or encoded images.
Once the content is created it can be shared with other internet
users through a unique URL. The visitors to the URL will need 
to enter a PIN to get the content.

## Models

```
 User { username password(hashed) created_at }
   |
  /|\
Message { username content(encrypted) digest pin(hashed) attempt created_at }
```