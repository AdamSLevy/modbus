package modbus

import (
	"bufio"
	"context"
	"errors"
	"log"
	//"os"
	"os/exec"
	"regexp"
	"sync/atomic"
	"testing"
	"time"
)

var conSettings = [...]ConnectionSettings{
	{Mode: ModeASCII, Baud: 19200, Timeout: 500 * time.Millisecond},
	{Mode: ModeRTU, Baud: 19200, Timeout: 500 * time.Millisecond},
	{Mode: ModeTCP, Host: "localhost:5020", Timeout: 500 * time.Millisecond},
}
var modeName = [...]string{
	"ASCII",
	"RTU",
	"TCP",
}

var queries []Query

func init() {
	queries = make([]Query, 9)
	var err error
	slaveID := byte(1)
	values := make([]uint16, 2)
	if queries[0], err = ReadCoils(slaveID, 0, 1); err != nil {
		log.Println(err)
	}
	if queries[1], err = ReadDiscreteInputs(slaveID, 0, 1); err != nil {
		log.Println(err)
	}
	if queries[2], err = ReadInputRegisters(slaveID, 0, 1); err != nil {
		log.Println(err)
	}
	if queries[3], err = ReadHoldingRegisters(slaveID, 0, 1); err != nil {
		log.Println(err)
	}
	if queries[4], err = WriteSingleCoil(slaveID, 0, false); err != nil {
		log.Println(err)
	}
	if queries[5], err = WriteSingleRegister(slaveID, 0, 0); err != nil {
		log.Println(err)
	}
	if queries[6], err = WriteMultipleCoils(slaveID, 0, 2, values[0:1]); err != nil {
		log.Println(err)
	}
	if queries[7], err = WriteMultipleRegisters(slaveID, 0, 2, values); err != nil {
		log.Println(err)
	}
	if queries[8], err = MaskWriteRegister(slaveID, 0, 0, 0); err != nil {
		log.Println(err)
	}
}

var pattern = `\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2} socat\[\d+\] N PTY is (\/dev\/pts\/\d+)`
var socatCmd = "socat"
var socatArgs = []string{"-d", "-d", "pty,raw,echo=0", "pty,raw,echo=0"}

var diagslaveCmd = "diagslave"
var diagslaveASCIIArgs = []string{"-m", "ascii", "-a", "1"}
var diagslaveRTUArgs = []string{"-m", "rtu", "-a", "1"}
var diagslaveTCPArgs = []string{"-m", "tcp", "-a", "1", "-p", "5020"}

func TestMain(m *testing.M) {
	for i := range conSettings {
		cancel := setupModbusServer(&conSettings[i])
		defer cancel()
	}
	// Give time for the servers to start
	time.Sleep(150 * time.Millisecond)

	m.Run()
}

func TestInitialization(t *testing.T) {
	t.Run("NewClient", func(t *testing.T) {
		cl, err := NewClient(ConnectionSettings{})
		if nil != err {
			t.Error(err)
		}
		if nil == cl {
			t.Error("NewClient failed to return a valid Client")
		}
		if 1 != atomic.LoadUint32(&unmanagedClients) {
			t.Error("unmanagedClients not set to 1 after calling NewClient")
		}
	})
	t.Run("GetClientManager", func(t *testing.T) {
		cm, err := GetClientManager()
		if nil == err {
			t.Error("Failed to return an error after a call to NewClient")
		}
		if nil != cm {
			t.Error("Returned ClientManager after a call to NewClient")
		}
		atomic.StoreUint32(&unmanagedClients, 0)
		cm, err = GetClientManager()
		if nil != err {
			t.Fatal(err)
		}
		cM := clntMngr.Load().(*clientManager)
		if nil == cM.clients ||
			nil == cM.newClient ||
			nil == cM.deleteClient {
			t.Fatal("ClientManager was not properly initialized")
		}
		cm1, _ := GetClientManager()
		cm2, _ := GetClientManager()
		if cm1 != cm2 {
			t.Fatal("GetClientManager() returned two different " +
				"pointers")
		}
		cl, err := NewClient(ConnectionSettings{})
		if nil == err {
			t.Error("NewClient did not return an error after GetClientManager")
		}
		if nil != cl {
			t.Error("NewClient returned a Client after GetClientManager")
		}
	})
}

func TestClientManager(t *testing.T) {
	t.Run("SetupClient", func(t *testing.T) {
		for _, cs := range conSettings {
			t.Run(modeName[cs.Mode], func(t *testing.T) {
				t.Parallel()
				cm, err := GetClientManager()
				if nil != err {
					t.Fatal(err)
				}
				done := make(chan interface{}, 1)
				var ch *ClientHandle
				go func() {
					ch, err = cm.SetupClient(cs)
					done <- true
				}()
				select {
				case <-done:
				case <-time.After(500 * time.Millisecond):
					t.Fatal("SetupClient timed out")
				}
				if nil != err {
					t.Fatal(err)
				} else if nil == ch {
					t.Fatal("*ClientHandle is nil")
				}
				if ch.Close() != nil {
					t.Fatal(err)
				}
			})
		}
		t.Run("invalid", func(t *testing.T) {
			t.Parallel()
			cm, err := GetClientManager()
			if nil != err {
				t.Fatal(err)
			}
			done := make(chan interface{}, 1)
			var ch *ClientHandle
			go func() {
				ch, err = cm.SetupClient(ConnectionSettings{})
				done <- true
			}()
			select {
			case <-done:
			case <-time.After(500 * time.Millisecond):
				t.Fatal("SetupClient timed out")
			}
			if nil == err {
				t.Fatal("Did not return an error")
			}
			if nil != ch {
				t.Fatal("*ClientHandle is not nil")
			}
		})
	})
	// Give time for the clients to shutdown
	time.Sleep(50 * time.Millisecond)
	if len(clntMngr.Load().(*clientManager).clients) > 0 {
		t.Fatal("Clients did not shutdown on close")
	}
	t.Run("Query", func(t *testing.T) {
		testQueries(t)
	})
}

func testQuery(t *testing.T, ch ClientHandle, q Query) {
	done := make(chan interface{})
	var data []byte
	var err error
	go func() {
		data, err = ch.Send(q)
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Query timed out")
	}
	if nil != err {
		t.Fatal(err)
	}
	if nil == data {
		t.Fatal("Response data is nil")
	}
}

func testQueries(t *testing.T) {
	for i, cs := range conSettings {
		cs := cs
		cm, _ := GetClientManager()
		t.Run(modeName[i], func(t *testing.T) {
			t.Parallel()
			for _, q := range queries {
				q := q
				t.Run(FunctionNames[q.FunctionCode], func(t *testing.T) {
					t.Parallel()
					ch, err := cm.SetupClient(cs)
					if nil != err {
						t.Fatal(err)
					} else if nil == ch {
						t.Fatal("*ClientHandle is nil")
					}
					testQuery(t, *ch, q)
				})
			}
		})
	}
}

func setupModbusServer(cs *ConnectionSettings) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	var diagslaveArgs []string

	switch cs.Mode {
	case ModeASCII:
		diagslaveArgs = diagslaveASCIIArgs
	case ModeRTU:
		diagslaveArgs = diagslaveRTUArgs
	case ModeTCP:
		diagslaveArgs = diagslaveTCPArgs
	default:
		log.Fatal("Invalid Modbus Mode")
	}

	var waitFuncs []func() error
	// Set up pty devices using socat for serial modes
	if cs.Mode == ModeASCII || cs.Mode == ModeRTU {
		socat := exec.CommandContext(ctx, socatCmd, socatArgs...)
		rgx := regexp.MustCompile(pattern)
		stderr, err := socat.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		if err := socat.Start(); err != nil {
			log.Fatal(err)
		}
		waitFuncs = append(waitFuncs, socat.Wait)

		// Read the first three lines
		var out string
		rdr := bufio.NewReader(stderr)
		for i := 0; i < 3; i++ {
			o, err := rdr.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			out += o
		}

		// Parse out the pty device paths
		res := rgx.FindAllStringSubmatch(out, 2)
		if len(res) != 2 {
			log.Fatal(errors.New("Regex did not match"))
		}
		diagslaveArgs = append(diagslaveArgs, res[0][1])
		cs.Host = res[1][1]
	}

	diagslave := exec.CommandContext(ctx, diagslaveCmd, diagslaveArgs...)
	//if cs.Mode == ModeTCP {
	//diagslave.Stdout = os.Stdout
	//diagslave.Stderr = os.Stderr
	//}
	if err := diagslave.Start(); err != nil {
		log.Fatal(err)
	}
	waitFuncs = append(waitFuncs, diagslave.Wait)

	return func() {
		// Cancel context and wait for the processes to be killed
		cancel()
		for _, wait := range waitFuncs {
			wait()
		}
	}
}
