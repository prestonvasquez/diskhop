// Copyright 2024 Preston Vasquez
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dcrypto

import (
	"context"
	"crypto/cipher"
	"fmt"
)

const DefaultAEADNonceSize = 12

type AEAD struct {
	Cipher    cipher.AEAD
	Mgr       IVManagerGetter
	NonceSize int
}

var _ SealOpener = (*AEAD)(nil)

func NewAEAD(mgr IVManagerGetter, cipher cipher.AEAD) *AEAD {
	return &AEAD{Mgr: mgr, Cipher: cipher}
}

func NewAEADWithNonceSize(mgr IVManagerGetter, cipher cipher.AEAD, nonceSize int) *AEAD {
	return &AEAD{Mgr: mgr, Cipher: cipher, NonceSize: nonceSize}
}

func (a *AEAD) Seal(ctx context.Context, plaintext []byte) ([]byte, error) {
	nonceSize := a.NonceSize
	if nonceSize == 0 {
		nonceSize = DefaultAEADNonceSize
	}

	nonce, err := generateInitializationVector(ctx, a.Mgr, nonceSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	return a.Cipher.Seal(nonce, nonce, plaintext, nil), nil
}

func (a *AEAD) Open(ctx context.Context, ciphertext []byte) ([]byte, error) {
	nonceSize := a.NonceSize
	if nonceSize == 0 {
		nonceSize = DefaultAEADNonceSize
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	return a.Cipher.Open(nil, nonce, ciphertext, nil)
}
