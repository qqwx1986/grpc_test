Build
```cassandraql
go build 
```
Start Server
```cassandraql
./grpc_test
```
Start Client
```cassandraql
./grpc_test -cli=true -cnt=10000000
```
just watch memory

chg go.mod google.golang.org/grpc v1.27.1
repeated
there no memory leak 