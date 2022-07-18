package chord

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"net"
	"net/rpc"
	"time"
)

type network struct {
	serv    *rpc.Server
	lis     net.Listener
	nodePtr *WrapNode
}

func Accept(ser *rpc.Server, lis net.Listener, ptr *ChordNode) {
	for {
		//always run
		conn, tmp_err := lis.Accept()
		select {
		case <-ptr.IsQuit:
			return
		default:
			if tmp_err != nil {
				log.Print("rpc.Serve: accept:", tmp_err.Error())
				return
			}
			go ser.ServeConn(conn)
		}
	}
}

func (this *network) Init(address string, ptr *ChordNode) error {
	this.serv = rpc.NewServer()
	this.nodePtr = new(WrapNode)
	this.nodePtr.node = ptr
	//register rpc service
	tmp_err := this.serv.Register(this.nodePtr)
	if tmp_err != nil {
		log.Errorf("[error] register rpc service error!")
		return tmp_err
	}
	//for tcp listen
	this.lis, tmp_err = net.Listen("tcp", address)
	if tmp_err != nil {
		log.Errorf("[error] tcp error!")
		return tmp_err
	}
	go Accept(this.serv, this.lis, this.nodePtr.node)
	return nil
}

func (this *network) ShutDown() error {
	this.nodePtr.node.IsQuit <- true
	tmp_err := this.lis.Close()
	if tmp_err != nil {
		log.Errorln("ShutDown error")
	}
	return tmp_err
}

func GetClient(addr string) (*rpc.Client, error) {
	var res *rpc.Client
	var err error
	errCh := make(chan error)
	for i := 0; i < 5; i++ {
		go func() {
			res, err = rpc.Dial("tcp", addr)
			errCh <- err
		}()
		select {
		case <-errCh:
			if err == nil {
				return res, err
			} else {
				return nil, err
			}
		case <-time.After(waitTime):
			log.Errorln("In function GetClient time out in" + addr)
			err = errors.New("Time out!")
		}
	}
	return nil, err
}

func RemoteCall(aimNode string, aimFunc string, input interface{}, res interface{}) error {
	if aimNode == "" {
		log.Warningln("<RemoteCall> IP address is nil")
		return errors.New("Null address for RemoteCall")
	}
	c, tmp_err := GetClient(aimNode)
	if tmp_err != nil {
		log.Warningln("<RemoteCall> Fail to dial in ", aimNode, " and error is ", tmp_err)
		return tmp_err
	}
	tmp_err = c.Call(aimFunc, input, res)
	if tmp_err != nil {
		log.Infoln("Can not call function in ", aimNode, " the func is ", aimFunc, tmp_err)
	} else {
		log.Infoln("<RemoteCall> in ", aimNode, " with ", aimFunc, " success!")
	}
	c.Close()
	return tmp_err
}

func CheckOnline(addr string) bool {
	if addr == "" {
		log.Warningln("In checkonline the addr is nil")
		return false
	}
	cli, _ := GetClient(addr)
	if cli != nil {
		defer cli.Close()
		return true
	} else {
		return false
	}
}
