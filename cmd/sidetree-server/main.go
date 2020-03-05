/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/trustbloc/sidetree-core-go/pkg/batch"
	"github.com/trustbloc/sidetree-core-go/pkg/dochandler"
	"github.com/trustbloc/sidetree-core-go/pkg/dochandler/didvalidator"
	"github.com/trustbloc/sidetree-core-go/pkg/processor"
	"github.com/trustbloc/sidetree-core-go/pkg/restapi/diddochandler"
	sidetreecontext "github.com/trustbloc/sidetree-mock/pkg/context"
	"github.com/trustbloc/sidetree-mock/pkg/httpserver"
	"github.com/trustbloc/sidetree-mock/pkg/observer"
)

var logger = logrus.New()
var config = viper.New()

const didDocNamespace = "did:sidetree"
const basePath = "/document"

func main() {
	config.SetEnvPrefix("SIDETREE_MOCK")
	config.AutomaticEnv()
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	logger.Info("starting sidetree node...")

	storage, err := newLeveldbStore()
	if err != nil {
		logger.Errorf("Failed to create new database: %s", err.Error())
		panic(err)
	}

	ctx, err := sidetreecontext.New(config, storage)
	if err != nil {
		logger.Errorf("Failed to create new context: %s", err.Error())
		panic(err)
	}

	// create new batch writer
	batchWriter, err := batch.New(didDocNamespace, ctx)
	if err != nil {
		logger.Errorf("Failed to create batch writer: %s", err.Error())
		panic(err)
	}

	// start routine for creating batches
	batchWriter.Start()

	// start observer
	observer.Start(ctx.Blockchain(), ctx.CAS(), ctx.OperationStore())

	// did document handler with did document validator for didDocNamespace
	didDocHandler := dochandler.New(
		didDocNamespace,
		ctx.Protocol(),
		didvalidator.New(ctx.OperationStore()),
		batchWriter,
		processor.New(didDocNamespace, ctx.OperationStore()),
	)

	restSvc := httpserver.New(
		getListenURL(),
		diddochandler.NewUpdateHandler(basePath, didDocHandler),
		diddochandler.NewResolveHandler(basePath, didDocHandler),
	)

	if restSvc.Start() != nil {
		panic(err)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interrupt
	<-interrupt

	// Shut down all services
	batchWriter.Stop()

	if err := storage.Close(); err != nil {
		logger.Errorf("Error stopping levelDB storage: %s", err)
	}

	if err = restSvc.Stop(context.Background()); err != nil {
		logger.Errorf("Error stopping REST service: %s", err)
	}
}

func getListenURL() string {
	host := config.GetString("host")
	if host == "" {
		host = "0.0.0.0"
	}
	port := config.GetInt("port")
	if port == 0 {
		panic("port is not set")
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func newLeveldbStore() (*leveldb.DB, error) {
	dbPath := config.GetString("dbpath")
	if dbPath == "" {
		panic("database path is not set")
	}
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}

	return db, nil
}
