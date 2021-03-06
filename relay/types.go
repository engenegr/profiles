package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec"
)

const (
	KindSetMetadata     uint8 = 0
	KindTextNote        uint8 = 1
	KindRecommendServer uint8 = 2
)

type Event struct {
	ID string `db:"id"` // it's the hash of the serialized event

	Pubkey string `db:"pubkey"`
	Time   uint32 `db:"time"`

	Kind uint8 `db:"kind"`

	Reference string `db:"reference"` // the id of another event, optional
	Content   string `db:"content"`
	Signature string `db:"signature"`
}

// Serialize outputs a byte array that can be hashed/signed to identify/authenticate
// this event. An error will be returned if anything is malformed.
func (evt *Event) Serialize() ([]byte, error) {
	b := bytes.Buffer{}

	// version: 0 (only because if more fields are added later the id will not match)
	b.Write([]byte{0})

	// pubkey
	pubkeyb, err := hex.DecodeString(evt.Pubkey)
	if err != nil {
		return nil, err
	}
	pubkey, err := btcec.ParsePubKey(pubkeyb, btcec.S256())
	if err != nil {
		return nil, fmt.Errorf("error parsing pubkey: %w", err)
	}
	if evt.Pubkey != hex.EncodeToString(pubkey.SerializeCompressed()) {
		return nil, fmt.Errorf("pubkey is not serialized in compressed format")
	}
	if _, err = b.Write(pubkeyb); err != nil {
		return nil, err
	}

	// time
	var timeb [4]byte
	binary.BigEndian.PutUint32(timeb[:], evt.Time)
	if _, err := b.Write(timeb[:]); err != nil {
		return nil, err
	}

	// kind
	var kindb [1]byte
	kindb[0] = evt.Kind
	if _, err := b.Write(kindb[:]); err != nil {
		return nil, err
	}

	// reference
	if len(evt.Reference) != 0 && len(evt.Reference) != 64 {
		return nil, errors.New("reference must be either blank or 32 bytes")
	}
	if evt.Reference != "" {
		reference, err := hex.DecodeString(evt.Reference)
		if err != nil {
			return nil, errors.New("reference is an invalid hex string")
		}
		if _, err = b.Write(reference); err != nil {
			return nil, err
		}
	}

	// content
	if _, err = b.Write([]byte(evt.Content)); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// CheckSignature checks if the signature is valid for the id
// (which is a hash of the serialized event content).
// returns an error if the signature itself is invalid.
func (evt Event) CheckSignature() (bool, error) {
	// validity of these is checked by Serialize()
	pubkeyb, _ := hex.DecodeString(evt.Pubkey)
	pubkey, _ := btcec.ParsePubKey(pubkeyb, btcec.S256())

	bsig, err := hex.DecodeString(evt.Signature)
	if err != nil {
		return false, fmt.Errorf("signature is invalid hex: %w", err)
	}
	signature, err := btcec.ParseDERSignature(bsig, btcec.S256())
	if err != nil {
		return false, fmt.Errorf("failed to parse DER signature: %w", err)
	}

	hash, _ := hex.DecodeString(evt.ID)
	return signature.Verify(hash, pubkey), nil
}
