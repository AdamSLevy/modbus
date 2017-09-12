package modbus

import (
	"bufio"
	"context"
	"errors"
	"log"
	//"os"
	"os/exec"
	"regexp"
	"testing"
	"time"
)

var conSettings = [...]*ConnectionSettings{
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
	for i := range queries {
		queries[i].SlaveID = 1
	}
	values := make([]byte, 4)
	if v, err := queries[0].ReadCoils(0, 1); !v {
		log.Println(err)
	}
	if v, err := queries[1].ReadDiscreteInputs(0, 1); !v {
		log.Println(err)
	}
	if v, err := queries[2].ReadInputRegisters(0, 1); !v {
		log.Println(err)
	}
	if v, err := queries[3].ReadHoldingRegisters(0, 1); !v {
		log.Println(err)
	}
	if v, err := queries[4].WriteSingleCoil(0, false); !v {
		log.Println(err)
	}
	if v, err := queries[5].WriteSingleRegister(0, 0); !v {
		log.Println(err)
	}
	if v, err := queries[6].WriteMultipleCoils(0, 2, values[0:1]); !v {
		log.Println(err)
	}
	if v, err := queries[7].WriteMultipleRegisters(0, 2, values); !v {
		log.Println(err)
	}
	if v, err := queries[8].MaskWriteRegister(0, 0, 0); !v {
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
	for _, cs := range conSettings {
		cancel := setupModbusServer(cs)
		defer cancel()
	}
	// Give time for the servers to start
	time.Sleep(50 * time.Millisecond)

	m.Run()
}

func TestGetClientManager(t *testing.T) {
	t.Run("Initialization", func(t *testing.T) {
		t.Parallel()
		GetClientManager()
		if nil == clntMngr.clients ||
			nil == clntMngr.newClient ||
			nil == clntMngr.deleteClient {
			t.Fatal("ClientManager was not properly initialized")
		}
	})
	t.Run("Singleton", func(t *testing.T) {
		t.Parallel()
		GetClientManager()
		cm := clntMngr
		GetClientManager()
		if cm != clntMngr {
			t.Fatal("GetClientManager() returned two different " +
				"pointers")
		}
	})
}

func TestClientManager(t *testing.T) {
	t.Run("SetupClient", func(t *testing.T) {
		for i, cs := range conSettings {
			t.Run(modeName[i], func(t *testing.T) {
				t.Parallel()
				cm := GetClientManager()
				ch := make(chan interface{}, 1)
				var qq chan Query
				var err error
				go func() {
					qq, err = cm.SetupClient(*cs)
					ch <- true
				}()
				select {
				case <-ch:
				case <-time.After(500 * time.Millisecond):
					t.Fatal("SetupClient timed out")
				}
				if nil != err {
					t.Fatal(err)
				} else if nil == qq {
					t.Fatal("Query channel is nil")
				}
				close(qq)
			})
		}
		t.Run("invalid", func(t *testing.T) {
			t.Parallel()
			cm := GetClientManager()
			ch := make(chan interface{}, 1)
			var qq chan Query
			var err error
			go func() {
				qq, err = cm.SetupClient(ConnectionSettings{})
				ch <- true
			}()
			select {
			case <-ch:
			case <-time.After(500 * time.Millisecond):
				t.Fatal("SetupClient timed out")
			}
			if nil == err {
				t.Fatal("Did not return an error")
			}
			if nil != qq {
				t.Fatal("Query channel is not nil")
			}
		})
	})
	// Give time for the clients to shutdown
	time.Sleep(10 * time.Millisecond)
	if len(clntMngr.clients) > 0 {
		t.Fatal("Clients did not shutdown on close")
	}
}

func TestQueries(t *testing.T) {
	for i, cs := range conSettings {
		cs := cs
		cm := GetClientManager()
		t.Run(modeName[i], func(t *testing.T) {
			t.Parallel()
			for _, q := range queries {
				q := q
				t.Run(FunctionNames[q.FunctionCode], func(t *testing.T) {
					t.Parallel()
					qq, err := cm.SetupClient(*cs)
					if nil != err {
						t.Fatal(err)
					} else if nil == qq {
						t.Fatal("Query channel is nil")
					}
					q.Response = make(chan QueryResponse)
					ch := make(chan interface{})
					go func() {
						qq <- q
						ch <- true
					}()

					select {
					case <-ch:
					case <-time.After(500 * time.Millisecond):
						t.Fatal("Query timed out")
					}

					select {
					case res := <-q.Response:
						if nil != res.Err {
							t.Fatal(res.Err)
						}
						if nil == res.Data {
							t.Fatal("Data is nil")
						}
					case <-time.After(500 * time.Millisecond):
						t.Fatal("Response timeout")
					}
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
