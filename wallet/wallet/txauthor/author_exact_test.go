// Copyright (c) 2025-2026 The Pearl Research Labs
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package txauthor

import (
	"testing"

	"github.com/pearl-research-labs/pearl/node/btcutil"
	"github.com/pearl-research-labs/pearl/wallet/wallet/txsizes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExactNoChangeSpend asserts that a send whose value plus the fee of a
// no-change transaction is covered by the available inputs authors
// successfully, even when the same inputs cannot additionally cover a fee
// estimated with a change output that would never be created.  See issue
// #456: the fee was always estimated with a reserved change output, so
// affordable exact sends returned an insufficient funds error.
func TestExactNoChangeSpend(t *testing.T) {
	t.Parallel()

	changeSource := &ChangeSource{
		NewScript:  func() ([]byte, error) { return make([]byte, txsizes.P2TRPkScriptSize), nil },
		ScriptSize: txsizes.P2TRPkScriptSize,
	}

	tests := []struct {
		name  string
		utxo  btcutil.Amount
		pay   btcutil.Amount
		feeRb btcutil.Amount
	}{
		{
			// Retry path: the first fetch covers target+initial fee, but
			// the retry for the change-inflated maxRequiredFee does not.
			name:  "exact spend within retry reserve gap",
			utxo:  1e8,
			pay:   99_999_888,
			feeRb: 1e3,
		},
		{
			// Initial-selection path: the very first target
			// (amount + fee estimated with change) exceeds the input
			// amount, while amount + no-change fee does not.
			name:  "exact spend within initial reserve gap",
			utxo:  1e8,
			pay:   99_999_860,
			feeRb: 1e3,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			authored, err := NewUnsignedTransaction(
				p2trOutputs(tt.pay), tt.feeRb,
				makeInputSource(p2trOutputs(tt.utxo)), changeSource,
			)
			require.NoError(t, err,
				"exact-amount send must author without change")

			assert.Len(t, authored.Tx.TxOut, 1,
				"no change output should be created")
			assert.Equal(t, -1, authored.ChangeIndex)
			assert.Equal(t, 1, len(authored.Tx.TxIn))

			feePaid := authored.TotalInput - btcutil.Amount(authored.Tx.TxOut[0].Value)
			assert.Greater(t, int64(feePaid), int64(0),
				"transaction must pay a positive fee")
			t.Logf("input=%d output=%d fee=%d", authored.TotalInput,
				authored.Tx.TxOut[0].Value, feePaid)
		})
	}
}
