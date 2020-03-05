/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package context

import (
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/trustbloc/sidetree-core-go/pkg/api/protocol"
	"github.com/trustbloc/sidetree-core-go/pkg/batch"
	"github.com/trustbloc/sidetree-core-go/pkg/batch/cutter"
	"github.com/trustbloc/sidetree-core-go/pkg/batch/opqueue"
	"github.com/trustbloc/sidetree-core-go/pkg/processor"

	"github.com/trustbloc/sidetree-core-go/pkg/mocks"

	servermocks "github.com/trustbloc/sidetree-mock/pkg/mocks"
)

func New(cfg *viper.Viper, storage *leveldb.DB) (*ServerContext, error) { // nolint

	opsStore := mocks.NewMockOperationStore(nil)
	store := servermocks.NewMockLevelDBClient(storage)

	ctx := &ServerContext{
		ProtocolClient:       servermocks.NewMockProtocolClient(),
		StoreClient:          store,
		BlockchainClient:     mocks.NewMockBlockchainClient(nil),
		OperationStoreClient: opsStore,
		OpQueue:              &opqueue.MemQueue{},
	}

	return ctx, nil

}

// ServerContext implements batch context
type ServerContext struct {
	ProtocolClient       *servermocks.MockProtocolClient
	StoreClient          *servermocks.MockLevelDBClient
	BlockchainClient     *mocks.MockBlockchainClient
	OperationStoreClient *mocks.MockOperationStore
	OpQueue              *opqueue.MemQueue
}

// Protocol returns the ProtocolClient
func (m *ServerContext) Protocol() protocol.Client {
	return m.ProtocolClient
}

// Blockchain returns the block chain client
func (m *ServerContext) Blockchain() batch.BlockchainClient {
	return m.BlockchainClient
}

// CAS returns the CAS client
func (m *ServerContext) CAS() batch.CASClient {
	return m.StoreClient
}

// OperationStore returns the OperationStore client
func (m *ServerContext) OperationStore() processor.OperationStoreClient {
	return m.OperationStoreClient
}

// OperationQueue returns the queue containing the pending operations
func (m *ServerContext) OperationQueue() cutter.OperationQueue {
	return m.OpQueue
}
