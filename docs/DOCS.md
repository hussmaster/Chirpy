# API

### Health

```
GET /api/healthz/
```
Shows OK status if server is alive

### Metrics

```
GET /admin/metrics/
```
Shows number of site visits

### Reset

```
POST /admin/rest
```
Resets site visits. Will only work with env variable is set to 'dev'

### Create User

```
POST /api/users
```
Requires 
```
{
    "email": "email@email.com",
    "password": "password"
}
```
Will return the following
```
{
    "id": "user id",
    "created_at": "created at time",
    "updated_at": "updated at time",
    "email": "email address",
    "is_chirpy_red": "either true or false, default is false"
}

### Login

```
POST /api/login
```
Logs you in and returns multiple things but also an access and refresh token
```
{
    "id": "user_id",
    "created_at": "create at time",
    "updated_at": "updated at time",
    "email": "email address",
    "token": "access token",
    "refresh_token": "refresh token",
    "is_chirpy_red": "either true or false, default is false"
}
```

### Refresh token

```
POST /api/refresh
```
Refreshes access token using your refresh token
Requires token in header `Authorization: Bearer <refreshtoken>`
Returns
```
{
    "token": "new access token"
}

### Revoke Refresh Token

```
POST /api/revoke
```
Revokes refresh token
Requires token in header `Authorization: Bearer <refreshtoken>`
Returns Status code 204 for success

### Update User

```
PUT /api/users
```
Allows authenticated user to update email and password
Requires this in the requestBody
```
{
    "email": "email address",
    "password": "new password"
}
```
Requires access token in header `Authorization: Bearer <accessToken>`
Responds with
```
{
    "id": "user id",
    "created_at": "created at date",
    "updated_at": "new updated date",
    "email": "new email",
    "is_chirpy_red": "true or false"
}

### Post Chirp

```
POST /api/chirps
```
Posts chirp using an account
Requires this in the request body
```
{
    "body": "body of message"
}
```
Requires access token in header `Authorization: Bearer <accessToken>`
Responds with
```
{
    "id": "id of chirp",
    "created_at": "chirp creation date",
    "updated_at": "chirp creation date",
    "body": "body of chirp",
    "user_id": "user id of poster"
}
```

### Get all chirps

```
GET /api/chirps?sort=[desc/asc]?author_id={userID}
```
Can pass in an optional parameter to sort as well as an author ID. By default sorts ascending.
Returns multiples of
```
{
    "id": "id of chirp",
    "created_at": "creation date of chirp",
    "updated_at": "updated time of chirp",
    "body": "body of chirp",
    "user_id": "user id of chirp postee"
}
```

### Get one chirp

```
GET /api/chirps/{chirpID}
```
Returns only one chirp by ID. Requires chirp ID in http Path value
```
{
    "id": "id of chirp",
    "created_at": "creation date of chirp",
    "updated_at": "updated time of chirp",
    "body": "body of chirp",
    "user_id": "user id of chirp postee"
}

### Delete Chirp

```
DELETE /api/chirps/{chirpID}
```
Deletes chirp by ID. 
Requires access token in header `Authorization: Bearer <accessToken>`
Requires chirp ID in http Path value
Returns 204 status code for success.

### Chirpy Red

```
POST /api/polka/webhooks
```
Flips on chirpy red. Fake subscription service.
Requires API key in header `Authorization: ApiKey <apikey>`
Request body is
```
{
    "event": "event string",
    "data": {
        "user_id": "user id for chirpy red enable"
    }
}
```
Returns 204 status code for success.