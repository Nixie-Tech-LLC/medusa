The client side TV application API portion of Medusa's backend 

## API Spec. and Testing
There are 5 endpoints exposed in this module:
```go
    r.GET("/screens", listScreens)
    r.POST("/screens", createScreen)
    r.GET("/screens/:id", getScreen)
    r.PUT("/screens/:id", updateScreen)
    r.DELETE("/screens/:id", deleteScreen)
```
Pretty self explanatory functionality, here is how to test with curl

`Base URL: http://localhost:9000/api/tv`

Once you have signed up and logged in, record your authorization token, it will be necessarily formatted in the header as:
`-H "Authorization: Bearer <your_token>"`

`list` screens
```bash 
curl -X GET http://localhost:9000/api/tv/screens \
  -H "Authorization: Bearer $JWT"
```

`create` screen 
```bash
curl -X POST http://localhost:9000/api/tv/screens \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Lobby Display",
    "location": "First Floor"
}'
```

`get` screen by `id` (untested)
```bash 
curl -X GET http://localhost:9000/api/tv/screens/1 \
  -H "Authorization: Bearer $JWT"
```

`update` screen (untested)
```bash
curl -X PUT http://localhost:9000/api/tv/screens/1 \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Main Hall Display",
    "location": "Second Floor"
}'
```

`delete` screen (untested)
```bash 
curl -X DELETE http://localhost:9000/api/tv/screens/1 \
  -H "Authorization: Bearer $JWT"
```
