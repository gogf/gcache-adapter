# gcache-adapter
Adapters for package gcache.

## Usage

### Normal Cache

```go
cache := gcache.New()
adapter := adapter.NewRedis(g.Redis())
cache.SetAdapter(adapter)
```

### Database Cache
```go
adapter := adapter.NewRedis(g.Redis())
g.DB().GetCache().SetAdapter(adapter)
```
