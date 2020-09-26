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

### Change Database Cache From In-Memory To Redis
```go
adapter := adapter.NewRedis(g.Redis())
g.DB().GetCache().SetAdapter(adapter)
```
