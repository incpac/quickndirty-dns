# Quick'n'dirty DNS server

A quick'n'dirty DNS server designed for temporary use

## Compile

Simply download, grab dependencies, and complile.

```
go get github.com/incpac/quickndirty-dns
dep ensure
go build -ldflags "-X main.Version=0.0.1"
```

## Running
```
$ ./qnddns --config ./configuration.yml
```

See `qnddns --help` for all options

## Sample Config

```
results:
  - name:  google.com.
    value: 127.0.0.1
  - name:  amazon.com.
    value: 127.0.0.1
```