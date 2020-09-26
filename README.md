# gcache-adapter
Adapters for package gcache.

## Requirements

```shell script
gf version >= v1.14.0 
```
> Or using the `master` branch.



## Installation

```shell script
go get -u github.com/gogf/gcache-adapter
```

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
