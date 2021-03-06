/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trustbloc/sidetree-core-go/pkg/docutil"
	"github.com/trustbloc/sidetree-core-go/pkg/mocks"
	"github.com/trustbloc/sidetree-core-go/pkg/restapi/common"
	"github.com/trustbloc/sidetree-core-go/pkg/restapi/diddochandler"
	"github.com/trustbloc/sidetree-core-go/pkg/restapi/dochandler"
	"github.com/trustbloc/sidetree-core-go/pkg/restapi/model"
)

const (
	url       = "localhost:8080"
	clientURL = "http://" + url

	didDocNamespace = "did:sidetree"
	basePath        = "/document"

	sha2_256        = 18
	sampleNamespace = "sample:sidetree"
	samplePath      = "/sample"
)

func TestServer_Start(t *testing.T) {
	didDocHandler := mocks.NewMockDocumentHandler().WithNamespace(didDocNamespace)
	sampleDocHandler := mocks.NewMockDocumentHandler().WithNamespace(sampleNamespace)

	s := New(url,
		diddochandler.NewUpdateHandler(basePath, didDocHandler),
		diddochandler.NewResolveHandler(basePath, didDocHandler),
		newSampleUpdateHandler(sampleDocHandler),
		newSampleResolveHandler(sampleDocHandler),
	)
	require.NoError(t, s.Start())
	require.Error(t, s.Start())

	// Wait for the service to start
	time.Sleep(time.Second)

	encodedPayload, err := getCreatePayload()
	require.NoError(t, err)

	didID, err := docutil.CalculateID(didDocNamespace, encodedPayload, sha2_256)
	require.NoError(t, err)

	sampleID, err := docutil.CalculateID(sampleNamespace, encodedPayload, sha2_256)
	require.NoError(t, err)

	t.Run("Create DID doc", func(t *testing.T) {
		createReq, err := getCreateRequest()
		require.NoError(t, err)

		request := &model.Request{}
		err = json.Unmarshal(createReq, request)
		require.NoError(t, err)

		resp, err := httpPut(t, clientURL+basePath, request)
		require.NoError(t, err)
		require.NotNil(t, resp)
		doc := make(map[string]interface{})
		require.NoError(t, json.Unmarshal(resp, &doc))
		require.Equal(t, didID, doc["id"])
	})
	t.Run("Resolve DID doc", func(t *testing.T) {
		resp, err := httpGet(t, clientURL+basePath+"/"+didID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		doc := make(map[string]interface{})
		require.NoError(t, json.Unmarshal(resp, &doc))
		require.Equal(t, didID, doc["id"])
	})
	t.Run("Create Sample doc", func(t *testing.T) {
		createReq, err := getCreateRequest()
		require.NoError(t, err)

		request := &model.Request{}
		err = json.Unmarshal(createReq, request)
		require.NoError(t, err)

		resp, err := httpPut(t, clientURL+samplePath, request)
		require.NoError(t, err)
		require.NotNil(t, resp)
		doc := make(map[string]interface{})
		require.NoError(t, json.Unmarshal(resp, &doc))
		require.Equal(t, sampleID, doc["id"])
	})
	t.Run("Resolve Sample doc", func(t *testing.T) {
		resp, err := httpGet(t, clientURL+samplePath+"/"+sampleID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		doc := make(map[string]interface{})
		require.NoError(t, json.Unmarshal(resp, &doc))
		require.Equal(t, sampleID, doc["id"])
	})
	t.Run("Stop", func(t *testing.T) {
		require.NoError(t, s.Stop(context.Background()))
		require.Error(t, s.Stop(context.Background()))
	})
}

// httpPut sends a regular POST request to the sidetree-node
// - If post request has operation "create" then return sidetree document else no response
func httpPut(t *testing.T, url string, req *model.Request) ([]byte, error) {
	client := &http.Client{}
	b, err := json.Marshal(req)
	require.NoError(t, err)

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(b))
	require.NoError(t, err)

	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := invokeWithRetry(
		func() (response *http.Response, e error) {
			return client.Do(httpReq)
		},
	)
	require.NoError(t, err)
	return handleHttpResp(t, resp)
}

// httpGet send a regular GET request to the sidetree-node and expects 'side tree document' argument as a response
func httpGet(t *testing.T, url string) ([]byte, error) {
	client := &http.Client{}
	resp, err := invokeWithRetry(
		func() (response *http.Response, e error) {
			return client.Get(url)
		},
	)
	require.NoError(t, err)
	return handleHttpResp(t, resp)
}

func handleHttpResp(t *testing.T, resp *http.Response) ([]byte, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body failed: %w", err)
	}

	if status := resp.StatusCode; status != http.StatusOK {
		return nil, fmt.Errorf(string(body))
	}
	return body, nil
}

func invokeWithRetry(invoke func() (*http.Response, error)) (*http.Response, error) {
	remainingAttempts := 20
	for {
		resp, err := invoke()
		if err == nil {
			return resp, err
		}
		remainingAttempts--
		if remainingAttempts == 0 {
			return nil, err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

type sampleUpdateHandler struct {
	*dochandler.UpdateHandler
}

func newSampleUpdateHandler(processor dochandler.Processor) *sampleUpdateHandler {
	return &sampleUpdateHandler{
		UpdateHandler: dochandler.NewUpdateHandler(processor),
	}
}

// Path returns the context path
func (h *sampleUpdateHandler) Path() string {
	return samplePath
}

// Method returns the HTTP method
func (h *sampleUpdateHandler) Method() string {
	return http.MethodPost
}

// Handler returns the handler
func (h *sampleUpdateHandler) Handler() common.HTTPRequestHandler {
	return h.Update
}

// Update creates/updates the document
func (o *sampleUpdateHandler) Update(rw http.ResponseWriter, req *http.Request) {
	o.UpdateHandler.Update(rw, req)
}

type sampleResolveHandler struct {
	*dochandler.ResolveHandler
}

func newSampleResolveHandler(resolver dochandler.Resolver) *sampleResolveHandler {
	return &sampleResolveHandler{
		ResolveHandler: dochandler.NewResolveHandler(resolver),
	}
}

// Path returns the context path
func (h *sampleResolveHandler) Path() string {
	return samplePath + "/{id}"
}

// Method returns the HTTP method
func (h *sampleResolveHandler) Method() string {
	return http.MethodGet
}

// Handler returns the handler
func (h *sampleResolveHandler) Handler() common.HTTPRequestHandler {
	return h.Resolve
}

func getCreatePayload() (string, error) {
	payload, err := json.Marshal(
		struct {
			Operation   model.OperationType `json:"type"`
			DIDDocument string              `json:"didDocument"`
		}{model.OperationTypeCreate, docutil.EncodeToString([]byte(validDoc))})

	if err != nil {
		return "", nil
	}

	return docutil.EncodeToString(payload), nil
}

func getCreateRequest() ([]byte, error) {
	encodedPayload, err := getCreatePayload()
	if err != nil {
		return nil, err
	}

	req := model.Request{
		Protected: &model.Header{
			Alg: "ES256K",
			Kid: "#key1",
		},
		Payload:   encodedPayload,
		Signature: "",
	}

	return json.Marshal(req)
}

const validDoc = `{
	"@context": ["https://w3id.org/did/v1"],
	"created": "2019-09-23T14:16:59.261024-04:00",
	"publicKey": [{
		"controller": "controller",
		"id": "#key-1",
		"publicKeyBase58": "GY4GunSXBPBfhLCzDL7iGmP5dR3sBDCJZkkaGK8VgYQf",
		"type": "Ed25519VerificationKey2018"
	}],
	"updated": "2019-09-23T14:16:59.261024-04:00"
}`
