all: service.pb.go raftadmin_grpc.pb.go 
	    @echo "Done"
.PHONY: all

service.pb.go: service.proto
	protoc service.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

raftadmin_grpc.pb.go: raftadmin.proto
	protoc raftadmin.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative

clean:
	rm -f service.pb.go
	rm -f raftadmin_grpc.pb.go raftadmin.pb.go