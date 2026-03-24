module building-services/project-service

go 1.24.4

require (
	building-services v0.0.0-00010101000000-000000000000
	github.com/lib/pq v1.12.0
	github.com/rabbitmq/amqp091-go v1.10.0
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
)

replace building-services => ../
