# protoc-gen-bmicro

> A protobuf v3 beego(v1) x go-micro(v2) code generator.

## Usage

### Installation
```bash
brew install protoc
```

```bash
git clone https://github.com/shellbreaker/protoc-gen-bmicro.git
cd protoc-gen-bmicro/cmd/protoc-genbmicro
go install
```

### Protobuf Coding
```protobuf
syntax = "proto3";
package test;
option go_package = ".;gateway";

message Request {
  string query = 1 [(gogoproto.moretags) = "form:\"query\""]; //use form tag to get query/form value
}

message Response {
  int64 code = 2;
  string msg = 4;
}

service Test {
  // @router /api/v1/resource [get]
  rpc Get(Request) returns (Response) {}
  // @router /api/v1/resource [post]
  rpc Post(Request) returns (Response) {}
}
```

### Generate Code
```bash
protoc -I="." -I="$GOPATH/src/github.com/gogo/protobuf/gogoproto" --gofast_out=/path/to/your/dir \
--micro_out=/path/to/your/dir --bmicro_out=/path/to/your/dir /your/protobuf.proto
```

### Register Gateway
```go
func main() {
    srv := micro.NewService(micro.Name("go.rpc.test"))
    srv.Init()
    cli := srv.NewTestService("go.rpc.test", srv.Client())
    
    pb.RegisterTestGateway(cli)
}
```