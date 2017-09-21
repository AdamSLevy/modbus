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

func TestMain(m *testing.M) {
	for i := range testConSettings {
		if testConSettings[i].isValid {
			cancel := setupModbusServer(&testConSettings[i].ConnectionSettings)
			defer cancel()
		}
	}
	//testConSettings[3].Host = testConSettings[1].Host
	// Give time for the servers to start
	time.Sleep(250 * time.Millisecond)

	m.Run()
}

type testConnectionSettings struct {
	isValid bool
	delay   time.Duration
	ConnectionSettings
}

var testConSettings = [...]testConnectionSettings{
	{isValid: true, ConnectionSettings: ConnectionSettings{
		Mode: ModeASCII, Baud: 19200, Timeout: 500 * time.Millisecond}},
	{isValid: true, ConnectionSettings: ConnectionSettings{
		Mode: ModeRTU, Baud: 19200, Timeout: 500 * time.Millisecond}},
	{isValid: true, ConnectionSettings: ConnectionSettings{
		Mode: ModeTCP, Host: "localhost:5020", Timeout: 500 * time.Millisecond}},
	//{isValid: false, delay: 50 * time.Millisecond,
	//	ConnectionSettings: ConnectionSettings{
	//		Mode: ModeRTU, Baud: 9600, Timeout: 500 * time.Millisecond}},
	{isValid: false, ConnectionSettings: ConnectionSettings{
		Mode: ModeASCII, Timeout: 500 * time.Millisecond}},
	{isValid: false, ConnectionSettings: ConnectionSettings{
		Mode: ModeRTU, Timeout: 500 * time.Millisecond}},
	{isValid: false, ConnectionSettings: ConnectionSettings{
		Mode: ModeTCP, Timeout: 500 * time.Millisecond}},
}

var ttyPathPattern = `\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2} socat\[\d+\] N PTY is (\/dev\/pts\/\d+)`
var socatCmd = "socat"
var socatArgs = []string{"-d", "-d", "pty,raw,echo=0", "pty,raw,echo=0"}

var diagslaveCmd = "diagslave"
var diagslaveASCIIArgs = []string{"-m", "ascii", "-a", "1"}
var diagslaveRTUArgs = []string{"-m", "rtu", "-a", "1"}
var diagslaveTCPArgs = []string{"-m", "tcp", "-a", "1", "-p", "5020"}

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
		rgx := regexp.MustCompile(ttyPathPattern)
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
	//	diagslave.Stdout = os.Stdout
	//	diagslave.Stderr = os.Stderr
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
