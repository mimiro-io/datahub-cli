// Copyright 2021 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"github.com/mimiro-io/datahub-cli/internal/utils"
)

type TxnManager struct {
	server string
	token  string
}

func NewTxnManager(server string, token string) *TxnManager {
	return &TxnManager{
		server: server,
		token:  token,
	}
}

// ExecuteTransaction send txn to the server for execution
func (txnMgr *TxnManager) ExecuteTransaction(txnData []byte) error {
	_, err := utils.PostRequest(txnMgr.server, txnMgr.token, "/transactions", txnData)
	return err
}

