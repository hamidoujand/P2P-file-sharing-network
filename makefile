.PHONY: build tracker peer up


GOLANG  := golang:1.23.0
ALPINE   := alpine:3.20

### Downlaod images 
docker-pull:
	docker pull $(GOLANG) & \
	docker pull $(ALPINE) & \
	wait;

tidy:
	go mod tidy 
	go mod vendor   

gen:
	protoc --go_out=./tracker --go-grpc_out=./tracker --proto_path=proto   proto/tracker.proto
	protoc --go_out=./peer --go-grpc_out=./peer --proto_path=proto proto/peer.proto
	protoc --go_out=./peer --go-grpc_out=./peer --proto_path=proto proto/tracker.proto
	protoc --go_out=./client --go-grpc_out=./client --proto_path=proto proto/peer.proto	
	protoc --go_out=./client --go-grpc_out=./client --proto_path=proto proto/tracker.proto	



# path for google protobuf messages, in case "protoc" did not find them add this flag.
#--proto_path=/usr/local/include/google/protobuf
#--proto_path=/usr/local/include/google/protobuf

download:
	go run client/main.go download -filename=file.txt -peer=127.0.0.0:50052

upload:
	go run client/main.go upload -filename=file.txt -peer=127.0.0.0:50053

get-peers:
	go run client/main.go peers -tracker=127.0.0.0:50051

### Build image
build: tracker peer

tracker:
	docker build -t tracker -f tracker/dockerfile .

peer:
	docker build -t peer -f peer/dockerfile .

up:
	docker-compose up --remove-orphans -d 


down:
	docker-compose  down

logs:
	docker-compose logs -f peer1 peer2