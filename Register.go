package clientlib

import (
	"bytes"
	"errors"
	"github.com/adrianleh/WTMP-middleend/command"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var regLock sync.Mutex

func listen(listener net.Listener) {
	conn, err := listener.Accept()
	if err != nil {
		log.Fatalf("listener failed to accept")
	}
	recvConnLock.Lock()
	defer recvConnLock.Unlock()
	recvConn = conn
	recvConnCond.Signal()
}

func Register(name string) error {
	regLock.Lock()
	defer regLock.Unlock()

	if uuid_ != uuid.Nil {
		return errors.New("attempt to register twice, UUID is already set: " + uuid_.String())
	}
	uuid__, uuidErr := uuid.NewRandom()
	if uuidErr != nil {
		return uuidErr
	}
	uuid_ = uuid__

	recvSockFile, fErr := ioutil.TempFile("", "wtmp-recv-socket")
	if fErr != nil {
		return fErr
	}
	sigs := make(chan os.Signal, 1)
	recvSockPath := recvSockFile.Name()
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		_ = os.Remove(recvSockPath)
		os.Exit(0)
	}()

	recvSockListener, lErr := net.Listen("unix", recvSockPath)
	if lErr != nil {
		return lErr
	}
	go listen(recvSockListener)

	size := uint64(4 + len(name) + len(recvSockPath))
	bufInit := make([]byte, 0, 25+size)
	out := bytes.NewBuffer(bufInit)
	if err := writeCommandHeader(command.RegisterCommandId, size, out); err != nil {
		return err
	}
	if err := writeBE(uint32(len(name)), out); err != nil {
		return err
	}
	if _, err := out.WriteString(name); err != nil {
		return err
	}
	if _, err := out.WriteString(recvSockPath); err != nil {
		return err
	}
	outBytes := out.Bytes()
	return sendViaSocket(&outBytes)
}
