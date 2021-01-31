# protoc-gen-bmicro


> https://github.com/grpc-ecosystem/grpc-gateway

> protoc --go_out=./gateway --go-grpc_out=./gateway --grpc-gateway_out=logtostderr=true:./gateway -I="." -I="./gateway" -I="$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/v2/third_party/googleapis/" ./test.proto