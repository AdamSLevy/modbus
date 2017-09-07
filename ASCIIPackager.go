package modbus

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"
)

// ASCIIPackager generates packet frames from Queries, sends the packet, and
// reads and validates the response via the Transporter
type ASCIIPackager struct {
	PackagerSettings
}

// NewASCIIPackager initializes a new ASCIIPackager with the given Transporter
// t.
func NewASCIIPackager(t Transporter) *ASCIIPackager {
	return &ASCIIPackager{
		PackagerSettings{
			Transporter: t,
			Validate:    true,
			Debug:       false,
		},
	}
}

// GeneratePacket generates the bytes of the packet for the given Query after
// validating that the Query is well formed. This must be called prior to
// calling Send.
func (pkgr *ASCIIPackager) GeneratePacket(qry *Query) error {
	if pkgr.Validate {
		valid, err := qry.IsValid()
		if !valid {
			return err
		}
		if qry.SlaveID == 0 {
			return errors.New("SlaveID cannot be 0 for Modbus ASCII")
		}
	}
	pkgr.Query = qry
	packetLen := 7
	if len(qry.Data) > 0 {
		packetLen += len(qry.Data) + 1
		if packetLen > MaxASCIISize {
			return errors.New("Query Data is too long")
		}
	}

	rawPkt := make([]byte, packetLen)
	rawPkt[0] = qry.SlaveID
	rawPkt[1] = qry.FunctionCode
	rawPkt[2] = byte(qry.Address >> 8)    // (High Byte)
	rawPkt[3] = byte(qry.Address & 0xff)  // (Low Byte)
	rawPkt[4] = byte(qry.Quantity >> 8)   // (High Byte)
	rawPkt[5] = byte(qry.Quantity & 0xff) // (Low Byte)
	bytesUsed := 6

	for i := 0; i < len(qry.Data); i++ {
		rawPkt[(bytesUsed + i)] = qry.Data[i]
	}
	bytesUsed += len(qry.Data)

	// add the lrc to the end
	pktLrc := lrc(rawPkt[:bytesUsed])
	rawPkt[bytesUsed] = byte(pktLrc)
	bytesUsed++

	// Convert raw bytes to ASCII packet
	asciiPkt := make([]byte, bytesUsed*2+3)
	hex.Encode(asciiPkt[1:], rawPkt)

	asciiBytesUsed := bytesUsed*2 + 1

	// Frame the packet
	asciiPkt[0] = ':'                 // 0x3A
	asciiPkt[asciiBytesUsed] = '\r'   // CR 0x0D
	asciiPkt[asciiBytesUsed+1] = '\n' // LF 0x0A
	asciiBytesUsed += 2

	pkgr.pkt = bytes.ToUpper(asciiPkt[:asciiBytesUsed])
	pkgr.pktLen = asciiBytesUsed
	return nil
}

// Send sends the packet generated by GeneratePacket and then reads, validates
// and returns the response bytes.
func (pkgr *ASCIIPackager) Send() ([]byte, error) {
	if pkgr.Debug {
		log.Println(fmt.Sprintf("Tx: %x", pkgr.pkt))
		log.Println(fmt.Sprintf("Tx: %s", pkgr.pkt))
	}

	// transmit the ADU to the slave device via the
	// serial port represented by the fd pointer
	_, err := pkgr.Write(pkgr.pkt)
	if err != nil {
		if pkgr.Debug {
			log.Println(fmt.Sprintf("ASCII Write Err: %s", err))
		}
		return nil, err
	}

	// allow the slave device adequate time to respond
	time.Sleep(time.Duration(pkgr.TimeoutInMilliseconds) * time.Millisecond)

	// then attempt to read the reply
	asciiResponse := make([]byte, MaxASCIISize)
	asciiN, rerr := pkgr.Read(asciiResponse)
	if rerr != nil {
		if pkgr.Debug {
			log.Println(fmt.Sprintf("ASCII Read Err: %s", rerr))
		}
		return nil, rerr
	}

	// check the framing of the response
	if asciiResponse[0] != ':' ||
		asciiResponse[asciiN-2] != '\r' ||
		asciiResponse[asciiN-1] != '\n' {
		if pkgr.Debug {
			log.Println("ASCII Response Framing Invalid")
			log.Println(fmt.Sprintf("%s", asciiResponse))
		}
		return nil, exceptions[exceptionUnspecified]
	}

	// convert to raw bytes
	rawN := (asciiN - 3) / 2
	response := make([]byte, rawN)
	hex.Decode(response, asciiResponse[1:asciiN-2])

	// check the validity of the response
	ok, err := pkgr.isValidResponse(response)
	if !ok {
		return nil, err
	}

	// confirm the checksum (lrc)
	responseLrc := lrc(response[:rawN-1])
	if response[rawN-1] != responseLrc {
		// lrc failed (odd that there's no specific code for it)
		if pkgr.Debug {
			log.Println("ASCII Response Invalid: Bad Checksum")
		}
		// return the response bytes anyway, and let the caller decide
		return response, exceptions[exceptionBadChecksum]
	}

	// return only the number of bytes read
	return response[2 : rawN-1], nil
}

// Modbus ASCII uses Longitudinal Redundancy Check. lrc computes and returns
// the 2's compliment (-) of the sum of the given byte array modulo 256
func lrc(data []byte) uint8 {
	var sum uint8
	var lrc8 uint8
	for _, b := range data {
		sum += b
	}
	lrc8 = uint8(-int8(sum))
	return lrc8
}
