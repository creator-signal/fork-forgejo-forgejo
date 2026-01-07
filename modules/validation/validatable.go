// Copyright 2023, 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"forgejo.org/modules/timeutil"

	ap "github.com/go-ap/activitypub"
)

// ErrNotValid represents an validation error
type ErrNotValid struct {
	Message string
}

func (err ErrNotValid) Error() string {
	return fmt.Sprintf("Validation Error: %v", err.Message)
}

// IsErrNotValid checks if an error is a ErrNotValid.
func IsErrNotValid(err error) bool {
	_, ok := err.(ErrNotValid)
	return ok
}

type Validateable interface {
	Validate() []string
}

func IsValid(v Validateable) (bool, error) {
	if valdationErrors := v.Validate(); len(valdationErrors) > 0 {
		typeof := reflect.TypeOf(v)
		errString := strings.Join(valdationErrors, "\n")
		return false, ErrNotValid{fmt.Sprint(typeof, ": ", errString)}
	}

	return true, nil
}

func ValidateIDExists(value ap.Item, name string) []string {
	if value == nil {
		return []string{fmt.Sprintf("Field %v must not be nil", name)}
	}
	return ValidateNotEmpty(value.GetID().String(), name)
}

func ValidateNotEmpty(value any, name string) []string {
	isValid := true
	switch v := value.(type) {
	case string:
		if v == "" {
			isValid = false
		}
	case timeutil.TimeStamp:
		if v.IsZero() {
			isValid = false
		}
	case uint16:
		if v == 0 {
			isValid = false
		}
	case int64:
		if v == 0 {
			isValid = false
		}
	case []byte:
		if len(v) == 0 {
			isValid = false
		}
	default:
		isValid = false
	}

	if isValid {
		return []string{}
	}
	return []string{fmt.Sprintf("Value %v should not be empty", name)}
}

func ValidateMaxLen(value string, maxLen int, name string) []string {
	if utf8.RuneCountInString(value) > maxLen {
		return []string{fmt.Sprintf("Value %v is longer than expected length %v", name, maxLen)}
	}
	return []string{}
}

func ValidateOneOf(value any, allowed []any, name string) []string {
	for _, allowedElem := range allowed {
		if value == allowedElem {
			return []string{}
		}
	}
	return []string{fmt.Sprintf("Field %s contains the value %v, which is not in allowed subset %v", name, value, allowed)}
}

// ValidateIRI validates an ActivityPub IRI.
func ValidateIRI(iri ap.IRI, name string) []string {
	var res []string

	res = append(res, ValidateNotEmpty(iri.String(), name)...)
	if _, err := iri.URL(); err != nil {
		res = append(res, fmt.Sprintf("invalid IRI for field %s: %v", name, err))
	}

	return res
}

// ValidatePublicKey validates a PEM-encoded public key.
func ValidatePublicKey(key []byte, name string) []string {
	var res []string

	res = append(res, ValidateNotEmpty(key, name)...)

	if block, _ := pem.Decode(key); block == nil {
		return append(res, "invalid public key PEM encoding")
	} else if key, err := x509.ParsePKIXPublicKey(block.Bytes); err != nil {
		res = append(res, fmt.Sprintf("invalid public key encoding: %v", err))
	} else {
		switch key.(type) {
		case *rsa.PublicKey, *ecdsa.PublicKey, ed25519.PublicKey:
		default:
			res = append(res, "invalid public key type")
		}
	}

	return res
}
