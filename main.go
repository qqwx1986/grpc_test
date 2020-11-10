package main

import (
	"context"
	"encoding/json"
	"flag"
	"net"
	"os"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ImplementedGRPCTestServer struct {
	GRPCTestServer
}

var num int64

var mem [][]byte

func (th *ImplementedGRPCTestServer) GetRequest(ctx context.Context, req *GetReq) (*GetRsp, error) {
	atomic.AddInt64(&num, 1)
	logrus.Infof("Cmd %s", req.Cmd)
	switch req.Cmd {
	case "alloc":
		if mem == nil {
			mem = [][]byte{}
		}
		go alloc()
	case "free":
		mem = nil
	case "gc":
		runtime.GC()
	case "print":
		printMem()
	case "free_os":
		logrus.Info("free_os")
		debug.FreeOSMemory()
	}
	return &GetRsp{}, nil
}

var cli *bool
var addr *string
var cnt *int
var single *bool
var cmd *string
var allocNum *int

func alloc() {
	for i := 0; i < *allocNum; i++ {
		mem = append(mem, make([]byte, 10240))
	}
}
func printMem() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b, _ := json.Marshal(m)
	logrus.Infof("%s", string(b))
}
func main() {
	os.Setenv("GODEBUG", "madvdontneed=1")

	cli = flag.Bool("cli", false, "is run as client")
	cmd = flag.String("cmd", "", "cmd")
	allocNum = flag.Int("alloc_num", 102400, "alloc_num")
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
	RegisterGRPCTestServer(s, &ImplementedGRPCTestServer{})
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		logrus.Fatalf("listen err,%s", err.Error())
	}
	go func() {
		t := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-t.C:
				//logrus.Infof("num %d", atomic.LoadInt64(&num))
			}
		}
	}()
	if err := s.Serve(ln); err != nil {
		runtime.GC()
		logrus.Info("Stopped")
		time.Sleep(100 * time.Second)
		logrus.Fatalf("Serve err,%s", err.Error())
	}
	runtime.GC()
	logrus.Info("Stopped")
	time.Sleep(100 * time.Second)
}
func runCli() {
	if len(*cmd) > 0 {
		*cnt = 1
		*single = false
	}
	con, err := grpc.DialContext(context.Background(), *addr, grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		logrus.Fatalf("DialContext err,%s", err.Error())
	}
	c := NewGRPCTestClient(con)
	var w sync.WaitGroup
	for i := 0; i < *cnt; i++ {
		w.Add(1)
		go func() {
			if *single {
				con, err := grpc.DialContext(context.Background(), *addr, grpc.WithBlock(), grpc.WithInsecure())
				if err != nil {
					return
				} else {
					c = NewGRPCTestClient(con)
				}
			}
			_, err := c.GetRequest(context.Background(), &GetReq{Cmd: *cmd})
			if err != nil {
				logrus.Errorf("err %s", err.Error())
			}
			w.Done()
		}()
	}
	w.Wait()
}
