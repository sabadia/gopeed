package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type RespCode int

const (
	CodeOk RespCode = 0
	// CodeError is the common error code
	CodeError RespCode = 1000
	// CodeUnauthorized is the error code for unauthorized
	CodeUnauthorized RespCode = 1001
	// CodeInvalidParam is the error code for invalid parameter
	CodeInvalidParam RespCode = 1002
	// CodeTaskNotFound is the error code for task not found
	CodeTaskNotFound RespCode = 2001
)

type Result[T any] struct {
	Code RespCode `json:"code"`
	Msg  string   `json:"msg"`
	Data T        `json:"data"`
	Hash string   `json:"hash"`
}

func NewOkResult[T any](data T) *Result[T] {
	hash := generateHash(data)
	return &Result[T]{
		Code: CodeOk,
		Data: data,
		Hash: hash,
	}
}

func NewNilResult() *Result[any] {
	return &Result[any]{
		Code: CodeOk,
		Hash: generateHash(nil),
	}
}

func NewErrorResult(msg string, code ...RespCode) *Result[any] {
	// if code is not provided, the default code is CodeError
	c := CodeError
	if len(code) > 0 {
		c = code[0]
	}

	return &Result[any]{
		Code: c,
		Msg:  msg,
	}
}

func generateHash(data any) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}
