// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ztls

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

var tests = []interface{}{
	&clientHelloMsg{},
	&serverHelloMsg{},
	&finishedMsg{},

	&certificateMsg{},
	&certificateRequestMsg{},
	&certificateVerifyMsg{},
	&certificateStatusMsg{},
	&clientKeyExchangeMsg{},
	&nextProtoMsg{},
	&newSessionTicketMsg{},
	&sessionState{},
	&serverHelloMsg13{},
	&helloRequestMsg{},
	&encryptedExtensionsMsg{},
	&certificateMsg13{},
}

type testMessage interface {
	marshal() []byte
	unmarshal([]byte) bool
	equal(interface{}) bool
}

func TestMarshalUnmarshal(t *testing.T) {
	rand := rand.New(rand.NewSource(0))

	for i, iface := range tests {
		ty := reflect.ValueOf(iface).Type()

		n := 100
		if testing.Short() {
			n = 5
		}
		for j := 0; j < n; j++ {
			v, ok := quick.Value(ty, rand)
			if !ok {
				t.Errorf("#%d: failed to create value", i)
				break
			}

			m1 := v.Interface().(testMessage)
			marshaled := m1.marshal()
			m2 := iface.(testMessage)
			if !m2.unmarshal(marshaled) {
				t.Errorf("#%d.%d failed to unmarshal %#v %x", i, j, m1, marshaled)
				break
			}
			m2.marshal() // to fill any marshal cache in the message

			if !m1.equal(m2) {
				t.Errorf("#%d.%d got:%#v want:%#v %x", i, j, m2, m1, marshaled)
				break
			}

			if i >= 3 {
				// The first three message types (ClientHello,
				// ServerHello and Finished) are allowed to
				// have parsable prefixes because the extension
				// data is optional and the length of the
				// Finished varies across versions.
				for j := 0; j < len(marshaled); j++ {
					if m2.unmarshal(marshaled[0:j]) {
						t.Errorf("#%d unmarshaled a prefix of length %d of %#v", i, j, m1)
						break
					}
				}
			}
		}
	}
}

func TestFuzz(t *testing.T) {
	rand := rand.New(rand.NewSource(0))
	for _, iface := range tests {
		m := iface.(testMessage)

		for j := 0; j < 1000; j++ {
			len := rand.Intn(100)
			bytes := randomBytes(len, rand)
			// This just looks for crashes due to bounds errors etc.
			m.unmarshal(bytes)
		}
	}
}

func randomBytes(n int, rand *rand.Rand) []byte {
	r := make([]byte, n)
	for i := 0; i < n; i++ {
		r[i] = byte(rand.Int31())
	}
	return r
}

func randomString(n int, rand *rand.Rand) string {
	b := randomBytes(n, rand)
	return string(b)
}

func (*clientHelloMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &clientHelloMsg{}
	m.vers = uint16(rand.Intn(65536))
	m.random = randomBytes(32, rand)
	m.sessionId = randomBytes(rand.Intn(32), rand)
	m.cipherSuites = make([]uint16, rand.Intn(63)+1)
	for i := 0; i < len(m.cipherSuites); i++ {
		m.cipherSuites[i] = uint16(rand.Int31())
	}
	m.compressionMethods = randomBytes(rand.Intn(63)+1, rand)
	if rand.Intn(10) > 5 {
		m.nextProtoNeg = true
	}
	if rand.Intn(10) > 5 {
		m.serverName = randomString(rand.Intn(255), rand)
	}
	m.ocspStapling = rand.Intn(10) > 5
	m.supportedPoints = randomBytes(rand.Intn(5)+1, rand)
	m.supportedCurves = make([]CurveID, rand.Intn(5)+1)
	for i := range m.supportedCurves {
		m.supportedCurves[i] = CurveID(rand.Intn(30000))
	}
	if rand.Intn(10) > 5 {
		m.ticketSupported = true
		if rand.Intn(10) > 5 {
			m.sessionTicket = randomBytes(rand.Intn(300), rand)
		}
	}
	if rand.Intn(10) > 5 {
		m.signatureAndHashes = supportedSKXSignatureAlgorithms
	}
	m.alpnProtocols = make([]string, rand.Intn(5))
	for i := range m.alpnProtocols {
		m.alpnProtocols[i] = randomString(rand.Intn(20)+1, rand)
	}
	m.keyShares = make([]keyShare, rand.Intn(4))
	for i := range m.keyShares {
		m.keyShares[i].group = CurveID(rand.Intn(30000))
		m.keyShares[i].data = randomBytes(rand.Intn(300), rand)
	}
	m.supportedVersions = make([]uint16, rand.Intn(5))
	for i := range m.supportedVersions {
		m.supportedVersions[i] = uint16(rand.Intn(30000))
	}

	return reflect.ValueOf(m)
}

func (*serverHelloMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &serverHelloMsg{}
	m.vers = uint16(rand.Intn(65536))
	m.random = randomBytes(32, rand)
	m.sessionId = randomBytes(rand.Intn(32), rand)
	m.cipherSuite = uint16(rand.Int31())
	m.compressionMethod = uint8(rand.Intn(256))

	if rand.Intn(10) > 5 {
		m.nextProtoNeg = true

		n := rand.Intn(10)
		m.nextProtos = make([]string, n)
		for i := 0; i < n; i++ {
			m.nextProtos[i] = randomString(20, rand)
		}
	}
	m.alpnProtocol = randomString(rand.Intn(32)+1, rand)

	if rand.Intn(10) > 5 {
		m.ocspStapling = true
	}
	if rand.Intn(10) > 5 {
		m.ticketSupported = true
	}

	return reflect.ValueOf(m)
}

func (*serverHelloMsg13) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &serverHelloMsg13{}
	m.vers = uint16(rand.Intn(65536))
	m.random = randomBytes(32, rand)
	m.cipherSuite = uint16(rand.Int31())
	m.keyShare.group = CurveID(rand.Intn(30000))
	m.keyShare.data = randomBytes(rand.Intn(300), rand)
	m.signatureAlgorithms = true

	return reflect.ValueOf(m)
}

/*
func (*helloRequestMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &helloRequestMsg{}
	m.vers = uint16(rand.Intn(65536))
	m.cipherSuite = uint16(rand.Int31())
	m.keyShare.group = CurveID(rand.Intn(30000))
	m.keyShare.data = randomBytes(rand.Intn(300), rand)
	m.signatureAlgorithms = true

	return reflect.ValueOf(m)
}*/

func (*encryptedExtensionsMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &encryptedExtensionsMsg{}
	m.alpnProtocol = randomString(rand.Intn(32)+1, rand)

	return reflect.ValueOf(m)
}

func (*certificateMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &certificateMsg{}
	numCerts := rand.Intn(20)
	m.certificates = make([][]byte, numCerts)
	for i := 0; i < numCerts; i++ {
		m.certificates[i] = randomBytes(rand.Intn(10)+1, rand)
	}
	return reflect.ValueOf(m)
}

func (*certificateMsg13) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &certificateMsg13{}
	numCerts := rand.Intn(20)
	m.certificates = make([][]byte, numCerts)
	for i := 0; i < numCerts; i++ {
		m.certificates[i] = randomBytes(rand.Intn(10)+1, rand)
	}
	m.requestContext = randomBytes(rand.Intn(5), rand)
	return reflect.ValueOf(m)
}

func (*certificateRequestMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &certificateRequestMsg{}
	m.certificateTypes = randomBytes(rand.Intn(5)+1, rand)
	numCAs := rand.Intn(100)
	m.certificateAuthorities = make([][]byte, numCAs)
	for i := 0; i < numCAs; i++ {
		m.certificateAuthorities[i] = randomBytes(rand.Intn(15)+1, rand)
	}
	return reflect.ValueOf(m)
}

func (*certificateVerifyMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &certificateVerifyMsg{}
	m.signature = randomBytes(rand.Intn(15)+1, rand)
	return reflect.ValueOf(m)
}

func (*certificateStatusMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &certificateStatusMsg{}
	if rand.Intn(10) > 5 {
		m.statusType = statusTypeOCSP
		m.response = randomBytes(rand.Intn(10)+1, rand)
	} else {
		m.statusType = 42
	}
	return reflect.ValueOf(m)
}

func (*clientKeyExchangeMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &clientKeyExchangeMsg{}
	m.ciphertext = randomBytes(rand.Intn(1000)+1, rand)
	return reflect.ValueOf(m)
}

func (*finishedMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &finishedMsg{}
	m.verifyData = randomBytes(12, rand)
	return reflect.ValueOf(m)
}

func (*nextProtoMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &nextProtoMsg{}
	m.proto = randomString(rand.Intn(255), rand)
	return reflect.ValueOf(m)
}

func (*newSessionTicketMsg) Generate(rand *rand.Rand, size int) reflect.Value {
	m := &newSessionTicketMsg{}
	m.ticket = randomBytes(rand.Intn(4), rand)
	return reflect.ValueOf(m)
}

func (*sessionState) Generate(rand *rand.Rand, size int) reflect.Value {
	s := &sessionState{}
	s.vers = uint16(rand.Intn(10000))
	s.cipherSuite = uint16(rand.Intn(10000))
	s.masterSecret = randomBytes(rand.Intn(100), rand)
	numCerts := rand.Intn(20)
	s.certificates = make([][]byte, numCerts)
	for i := 0; i < numCerts; i++ {
		s.certificates[i] = randomBytes(rand.Intn(10)+1, rand)
	}
	return reflect.ValueOf(s)
}