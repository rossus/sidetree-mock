/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mocks

import (
	"bytes"
	"encoding/base64"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/trustbloc/sidetree-core-go/pkg/docutil"
)

const sha2_256 = 18

// MockLevelDBClient mocks levelDB CAS for running server in test mode.
type MockLevelDBClient struct {
	store *leveldb.DB
}

// NewMockLevelDBClient creates mock levelDB cas client;
func NewMockLevelDBClient(store *leveldb.DB) *MockLevelDBClient {
	return &MockLevelDBClient{store: store}
}

// Write writes the given content to levelDB CAS.
// returns the SHA256 hash in base64url encoding which represents the address of the content.
func (m *MockLevelDBClient) Write(content []byte) (string, error) {
	hash, err := docutil.ComputeMultihash(sha2_256, content)
	if err != nil {
		return "", fmt.Errorf("failed to compute multihash")
	}

	key := base64.URLEncoding.EncodeToString(hash)

	err = m.store.Put([]byte(key), content, nil)
	if err != nil {
		return "", fmt.Errorf("cannot put content into the database")
	}

	log.Debugf("added content with address[%s]", key)

	return key, nil
}

// Read reads the content of the given address in CAS.
// returns the content of the given address.
func (m *MockLevelDBClient) Read(address string) ([]byte, error) {
	content, err := m.store.Get([]byte(address), nil)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to get content from the database")
	}

	// decode address to verify hashes
	decoded, err := base64.URLEncoding.DecodeString(address)
	if err != nil {
		return nil, fmt.Errorf("cannot decode address string")
	}

	valueHash, err := docutil.ComputeMultihash(sha2_256, content)
	if err != nil {
		return nil, fmt.Errorf("failed to compute multihash")
	}

	if !bytes.Equal(valueHash, decoded) {
		return nil, fmt.Errorf("hashes don't match")
	}

	return content, nil
}
