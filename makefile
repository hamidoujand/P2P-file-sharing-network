run-tracker:
	go run tracker/main.go 

run-pee1:
	go run peer1/main.go

tidy:
	go mod tidy 
	go mod vendor   

gen:
	protoc --go_out=. --go-grpc_out=. --proto_path=proto   proto/tracker.proto
	protoc --go_out=./peer1 --go-grpc_out=./peer1 --proto_path=proto proto/peer.proto
	protoc --go_out=./peer2 --go-grpc_out=./peer2 --proto_path=proto proto/peer.proto




# path for google protobuf messages, in case "protoc" did not find them add this flag.
#--proto_path=/usr/local/include/google/protobuf
#--proto_path=/usr/local/include/google/protobuf