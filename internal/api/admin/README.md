The client side webapp admin platform API portion of Medusa.
## API Spec. and Testing

There are 4 endpoints exposed in this module:
```go
    r.POST("/auth/signup", userSignup)
    r.POST("/auth/login", userLogin)
    r.GET("/auth/current_profile", getCurrentProfile)
    r.PATCH("/auth/current_profile", updateCurrentProfile)
```
Pretty self-explanatory functionality. Here's how to test them with curl.

`Base URL: http://localhost:9000/api/admin`

Once you sign up and log in, the API will return a token you must include in the header of any authenticated request:

`-H "Authorization: Bearer <your_token>"`

`signup` create a new admin account

```bash
curl -X POST http://localhost:9000/api/admin/auth/signup \
  -H "Content-Type: application/json" \
  -d '{
    "email": "testadmin@example.com",
    "password": "testpassword",
    "name": "Test Admin"
}'
```

Response:
```json
{
  "token": "<jwt_token>"
}
```

`login` get a JWT token for existing admin
```bash 
curl -X POST http://localhost:9000/api/admin/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "testadmin@example.com",
    "password": "testpassword"
}'
```

Response:
```json 
{
  "token": "<jwt_token>"
}
```

get current profile (untested)
```bash 
curl -X GET http://localhost:9000/api/admin/auth/current_profile \
  -H "Authorization: Bearer $JWT"
```


Response:
```json 
{
  "id": 4,
  "email": "testadmin@example.com",
  "name": "Test Admin",
  "created_at": "2025-06-14T01:23:45Z",
  "updated_at": "2025-06-14T01:23:45Z"
}
```


update current profile (email or name) (untested)
```bash 
curl -X PATCH http://localhost:9000/api/admin/auth/current_profile \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newemail@example.com",
    "name": "Updated Admin"
}'
```


Response:
```json 
{
  "id": 4,
  "email": "newemail@example.com",
  "name": "Updated Admin",
  "created_at": "2025-06-14T01:23:45Z",
  "updated_at": "2025-06-14T01:25:00Z"
}
```


