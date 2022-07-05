package server

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	separator = "_"
)

// AccessToken = {tunnelID}_{random string}
type AccessToken string

func (token AccessToken) TunnelID() string {
	parts := strings.Split(string(token), separator)
	return parts[0]
}

// ConnectionID = {tunnelID}_{random string}
type ConnectionID string

func (connectionID ConnectionID) TunnelID() string {
	parts := strings.Split(string(connectionID), separator)
	return parts[0]
}

type IDTokenGen struct {
	TunnelIDDigit     uint
	TokenDigit        uint
	ConnectionIDDigit uint
}

func (idTokenGen *IDTokenGen) GenerateID(digit uint) (string, error) {

	const availableChars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const len = int64(len(availableChars))

	id := make([]byte, digit)

	for i := uint(0); i < digit; i++ {
		randomNumber, err := rand.Int(rand.Reader, big.NewInt(len))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		id[i] = availableChars[randomNumber.Int64()]
	}

	return string(id), nil
}

func (idTokenGen *IDTokenGen) NewTunnelID() (string, error) {

	tunnelID, err := idTokenGen.GenerateID(idTokenGen.TunnelIDDigit)
	return tunnelID, err
}

func (idTokenGen *IDTokenGen) NewToken(tunnelID string) (AccessToken, error) {

	id, err := idTokenGen.GenerateID(idTokenGen.TokenDigit)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("%s%s%s", tunnelID, separator, id)

	return AccessToken(result), nil
}

type IDTokens struct {
	TunnelID         string
	SourceToken      AccessToken
	DestinationToken AccessToken
}

func (idTokenGen *IDTokenGen) NewIDTokens() (IDTokens, error) {

	tunnelID, err := idTokenGen.NewTunnelID()
	if err != nil {
		return IDTokens{}, err
	}

	sourceToken, err := idTokenGen.NewToken(tunnelID)
	if err != nil {
		return IDTokens{}, err
	}

	destinationToken, err := idTokenGen.NewToken(tunnelID)
	if err != nil {
		return IDTokens{}, err
	}

	ids := IDTokens{
		TunnelID:         tunnelID,
		SourceToken:      sourceToken,
		DestinationToken: destinationToken,
	}

	return ids, nil
}

func (idTokenGen *IDTokenGen) NewConnectionID(tunnelID string) (ConnectionID, error) {

	id, err := idTokenGen.GenerateID(idTokenGen.ConnectionIDDigit)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("%s%s%s", tunnelID, separator, id)

	return ConnectionID(result), nil
}
