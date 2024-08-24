# Middlewares

## CORS
Support 6 access controls `Access-Control-Allow-Origin`, `Access-Control-Allow-Headers`, `Access-Control-Allow-Methods`, `Access-Control-Allow-Credentials`, `Access-Control-Expose-Headers`, `Access-Control-Max-Age`. 

### Usage
Install
```bash
go get -u github.com/gflydev/middleware/cors@v1.0.0
```

Quick usage `main.go`
```go
// Add global middlewares
app.Use(cors.New(cors.Data{
    core.HeaderAccessControlAllowOrigin: cors.AllowedOrigin,
}))
```
### Access controls:
- Access-Control-Allow-Origin: Accept all domains `*` (default)
- Access-Control-Allow-Headers: Accept all header parameters `Authorization, Content-Type, x-requested-with, origin, true-client-ip, X-Correlation-ID` (default)
- Access-Control-Allow-Methods: List supported methods `PUT`, `POST`, `GET`, `DELETE`, `OPTIONS`, `PATCH` (default)
- Access-Control-Allow-Credentials: N/A
- Access-Control-Expose-Headers: N/A
- Access-Control-Max-Age: N/A
