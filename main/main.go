package main

import (
	"context"
	"flag"
	"net"
	"runtime"
	"sync"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"github.com/qqwx1986/grpc_test"
)

type ImplementedGRPCTestServer struct {
	grpc_test.GRPCTestServer
}

func (th *ImplementedGRPCTestServer) GetRequest(ctx context.Context, req *grpc_test.GetReq) (*grpc_test.GetRsp, error) {
	return &grpc_test.GetRsp{A: req.A}, nil
}

var cli *bool
var addr *string
var cnt *int
var single *bool

func main() {
	cli = flag.Bool("cli", false, "is run as client")
	addr = flag.String("addr", "localhost:3334", "addr")
	cnt = flag.Int("cnt", 10, "cnt")
	single = flag.Bool("single", false, "single")
	flag.Parse()
	if *cli {
		runCli()
		return
	}
	go func() {
		if err := http.ListenAndServe("0.0.0.0:6060", nil); err != nil {
			logrus.Infof("start pprof failed %s", err.Error())
		}
	}()

	s := grpc.NewServer()
	grpc_test.RegisterGRPCTestServer(s, &ImplementedGRPCTestServer{})
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		logrus.Fatalf("listen err,%s", err.Error())
	}

	if err := s.Serve(ln); err != nil {
		logrus.Info("Stopped")
		time.Sleep(100 * time.Second)
		logrus.Fatalf("Serve err,%s", err.Error())
	}
	runtime.GC()
	logrus.Info("Stopped")
	time.Sleep(100 * time.Second)
}
func runCli() {
	con, err := grpc.DialContext(context.Background(), *addr, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		logrus.Fatalf("DialContext err,%s", err.Error())
	}
	c := grpc_test.NewGRPCTestClient(con)
	var w sync.WaitGroup
	for i := 0; i < *cnt; i++ {
		w.Add(1)
		go func() {
			if *single {
				con, err := grpc.DialContext(context.Background(), *addr, grpc.WithBlock(), grpc.WithInsecure())
				if err != nil {
					return
				} else {
					c = grpc_test.NewGRPCTestClient(con)
				}
			}
			_, err := c.GetRequest(context.Background(), &grpc_test.GetReq{A: "1"})
			if err != nil {
				logrus.Errorf("err %s", err.Error())
			}
			w.Done()
		}()
	}
	w.Wait()
}
