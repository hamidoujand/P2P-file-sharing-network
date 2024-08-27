run-tracker:
	go run tracker/main.go 

tidy:
	go mod tidy 
	go mod vendor   

gen-tracker:
	protoc --go_out=. --go-grpc_out=. --proto_path=proto   proto/tracker.proto

