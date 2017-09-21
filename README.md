# Threadsafe Modbus Client Library
[![GoDoc](https://godoc.org/github.com/AdamSLevy/modbus?status.svg)](https://godoc.org/github.com/AdamSLevy/modbus)

This Go library implements a Modbus Client (i.e. a master) that can be used
concurrently in multiple goroutines.

## Supported Protocols
- RTU
- ASCII
- TCP

## Supported Queries
- Read Coils
- Read Discrete Inputs
- Read Holding Registers
- Read Input Registers
- Write Single Coil
- Write Single Register
- Write Multiple Coils
- Write Multiple Registers
- Mask Write Register

## Example
Initialize a ConnectionSettings struct. Set the Mode, Host, Timeout, and Baud
if the Mode is ModeRTU or ModeASCII. When using ModeTCP the Host is the fully
qualified domain name or ip address and port number. The port number in the
Host string is required. When using ModeRTU or ModeASCII, the Baud rate is
required and the Host string is the full path to the serial device or the name
of the COM port if on Windows. For all modes, the Timeout cannot be zero.
```go
csTCP := ConnectionSettings{
        Mode: ModeTCP,
        Host: "192.168.1.121:502",
        Timeout: 500 * time.Millisecond,
}
csASCII := ConnectionSettings{
        Mode: ModeASCII,
        Host: "/dev/ttyS1",
        Baud: 9600,
        Timeout: 500 * time.Millisecond,
}
csRTU := ConnectionSettings{
        Mode: ModeRTU,
        Host: "/dev/ttyUSB0",
        Baud: 115200,
        Timeout: 500 * time.Millisecond,
}
```
GetClientHandle returns a ClientHandle object which can be used to concurrently
send Query objects to the underlying client. This starts the client with the
given ConnectionSettings if it's not already running. 
```go
ch, err := modbus.GetClientHandle(csTCP)
if nil != err {
        fmt.Println(err)
        return
}
```
Multiple ClientHandles can be acquired or the same ClientHandle can be copied
and reused in multiple goroutines. The ConnectionSettings must match exactly if
a client is already running with the same Host string.
```go
cs := csTCP
ch1, err := modbus.GetClientHandle(cs) // Returns another ClientHandle for the same client
if nil != err {
        fmt.Println(err)
        return
}
cs.Timeout = 1000
_, err := modbus.GetClientHandle(cs) // Returns error since the Timeout was changed
if nil != err {
        fmt.Println(err)
        return
}
```
Create a Query using one of the function code initializers. 
```go
readCoils, err := ReadCoils(0, 0, 5) // SlaveID, Address, Quantity
if nil != err {
        fmt.Println(err)
        return
}
data, err := ch.Send(readCoils)
```
You can edit and reuse the Query, say to change the SlaveID.
```go
readCoils.SlaveID = 1
data, err := ch.Send(readCoils)
```
Alternatively you can manually initialize a Query struct and call IsValid() on
the Query to make sure that it is well formed.
```go
readDiscreteInputs := Query{
        FunctionCode: FunctionReadDiscreteInputs,
        SlaveID: 1,
        Address: 3,
        Quantity: 4,
}
writeMultitpleRegisters := Query{
        FunctionCode: FunctionWriteMultipleRegisters,
        Address:      1,
        Quantity:     2,
        Values:       []uint16{0x8081, 500},
}

if valid, err := readDiscreteInputs.IsValid(); !valid {
        fmt.Println(err)
        return
}
if valid, err := writeMultipleRegisters.IsValid(); !valid {
        fmt.Println(err)
        return
}

```
When you are finished using the ClientHandle, call its Close() method. The
underlying client is closed and all associated goroutines will return once all
open ClientHandles are closed. Keep in mind that if you are sharing a
ClientHandle between multiple goroutines, and one call Close, that ClientHandle
will fail to send any further Queries.
