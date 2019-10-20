// Copyright 2019 Canonical Ltd.
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package tpm2

import (
	"bytes"
	"errors"
	"fmt"
)

// ErrResourceDoesNotExist is returned from TPMContext.WrapHandle if it is called with a handle that does not correspond to a
// resource that is loaded in to the TPM.
var ErrResourceDoesNotExist = errors.New("the resource does not exist on the TPM")

// InvalidResponseHeaderError is returned from TPMContext.RunCommandBytes and TPMContext.RunCommand (and any other methods that wrap
// around this function) if the TPM responds with a header that is invalid. This could be because there are insufficient bytes,
// or because the responseSize field has an invalid value
type InvalidResponseHeaderError struct {
	Command CommandCode
	msg     string
}

func (e InvalidResponseHeaderError) Error() string {
	return fmt.Sprintf("TPM returned an invalid header for command %s: %v", e.Command, e.msg)
}

// InvalidResponseHeaderError is returned from TPMContext.RunCommandBytes and TPMContext.RunCommand (and any other methods that wrap
// around this function) if the TPM responds with a payload that is invalid. This could be because there are fewer bytes than
// indicated in the header, or unmarshalling of the response payload failed because of an invalid union selector value.
type InvalidResponsePayloadError struct {
	Command CommandCode
	Bytes   []byte
	msg     string
}

func (e InvalidResponsePayloadError) Error() string {
	return fmt.Sprintf("TPM returned an invalid payload for command %s: %v", e.Command, e.msg)
}

// InvalidResponseAuthError is returned from TPMContext.RunCommand (and any other methods that wrap around this function) if a
// response HMAC check failed.
type InvalidResponseAuthError struct {
	Command CommandCode
	Index   int
	msg     string
}

func (e InvalidResponseAuthError) Error() string {
	return fmt.Sprintf("TPM returned an invalid authorization for command %s at index %d: %s", e.Command, e.Index, e.msg)
}

// TPMReadError is returned from TPMContext.RunCommandBytes and TPMContext.RunCommand (and any other methods that wrap around this
// function) if the transmission interface returns an error during reading.
type TPMReadError struct {
	Command CommandCode // Command code associated with this error
	Err     error       // Error returned from the transmission interface
}

func (e TPMReadError) Error() string {
	return fmt.Sprintf("cannot read response to command %s from TPM: %v", e.Command, e.Err)
}

// TPMWriteError is returned from TPMContext.RunCommandBytes and TPMContext.RunCommand (and any other methods that wrap around this
// function) if the transmission interface returns an error during writing.
type TPMWriteError struct {
	Command CommandCode // Command code associated with this error
	Err     error       // Error returned from the transmission interface
}

func (e TPMWriteError) Error() string {
	return fmt.Sprintf("cannot write command %s to TPM: %v", e.Command, e.Err)
}

// Ideally we would just support go's error wrapping using the %w verb support in fmt.Errorf, but that requires go >= 1.13
type wrapError struct {
	msg string
	err error
}

func (e *wrapError) Error() string {
	return e.msg
}

func (e *wrapError) Unwrap() error {
	return e.err
}

func errorUnwrapOriginal(err error) error {
	for {
		e, ok := err.(*wrapError)
		if !ok {
			return err
		}
		err = e.Unwrap()
	}
}

const (
	formatMask ResponseCode = 1 << 7

	fmt0ErrorCodeMask ResponseCode = 0x7f
	fmt0VersionMask   ResponseCode = 1 << 8
	fmt0VendorMask    ResponseCode = 1 << 10
	fmt0SeverityMask  ResponseCode = 1 << 11

	fmt1ErrorCodeMask            ResponseCode = 0x3f
	fmt1ParameterIndexMask       ResponseCode = 0xf00
	fmt1HandleOrSessionIndexMask ResponseCode = 0x700
	fmt1ParameterMask            ResponseCode = 1 << 6
	fmt1SessionMask              ResponseCode = 1 << 11

	fmt1IndexShift uint = 8
)

// TPM1Error is returned from DecodeResponseCode and TPMContext.RunCommand (and any other methods that wrap around this function) if
// the TPM response code indicates an error from a TPM 1.2 device.
type TPM1Error struct {
	Command CommandCode  // Command code associated with this error
	Code    ResponseCode // Response code
}

func (e TPM1Error) Error() string {
	return fmt.Sprintf("TPM returned a 1.2 error whilst executing command %s: 0x%08x", e.Command, e.Code)
}

// TPMVendorError is returned from DecodeResponseCode and TPMContext.RunCommand (and any other methods that wrap around this function)
// if the TPM response code indicates a vendor-specific error.
type TPMVendorError struct {
	Command CommandCode  // Command code associated with this error
	Code    ResponseCode // Response code
}

func (e TPMVendorError) Error() string {
	return fmt.Sprintf("TPM returned a vendor defined error whilst executing command %s: 0x%08x", e.Command,
		e.Code)
}

// WarningCode represents a response from the TPM that is not necessarily an error.
type WarningCode ResponseCode

// TPMWarning is returned from DecodeResponseCode and TPMContext.RunCommand (and any other methods that wrap around this function) if
// the TPM response code indicates a condition that is not necessarily an error.
type TPMWarning struct {
	Command CommandCode // Command code associated with this error
	Code    WarningCode // Warning code
}

func (e TPMWarning) Error() string {
	var builder bytes.Buffer
	fmt.Fprintf(&builder, "TPM returned a warning whilst executing command %s: %s", e.Command, e.Code)
	if desc, hasDesc := warningCodeDescriptions[e.Code]; hasDesc {
		fmt.Fprintf(&builder, " (%s)", desc)
	}
	return builder.String()
}

// ErrorCode0 represents a format-zero error code from the TPM. These error codes are not associated with a handle, parameter or
// session.
type ErrorCode0 ResponseCode

// TPMError is returned from DecodeResponseCode and TPMContext.RunCommand (and any other methods that wrap around this function) if
// the TPM response code indicates an error that is not associated with a handle, parameter or session.
type TPMError struct {
	Command CommandCode // Command code associated with this error
	Code    ErrorCode0  // Error code
}

func (e TPMError) Error() string {
	var builder bytes.Buffer
	fmt.Fprintf(&builder, "TPM returned an error whilst executing command %s: %s", e.Command, e.Code)
	if desc, hasDesc := errorCode0Descriptions[e.Code]; hasDesc {
		fmt.Fprintf(&builder, " (%s)", desc)
	}
	return builder.String()
}

// ErrorCode1 represents a format-one error code from the TPM. These error codes are associated with a handle, parameter or session.
type ErrorCode1 ResponseCode

// TPMParameterError is returned from DecodeResponseCode and TPMContext.RunCommand (and any other methods that wrap around this
// function) if the TPM response code indicates an error that is associated with a command parameter.
type TPMParameterError struct {
	Command CommandCode // Command code associated with this error
	Code    ErrorCode1  // Error code
	Index   int         // Index of the command parameter associated with this error, starting from 1
}

func (e TPMParameterError) Error() string {
	var builder bytes.Buffer
	fmt.Fprintf(&builder, "TPM returned an error for parameter %d whilst executing command %s: %s", e.Index, e.Command, e.Code)
	if desc, hasDesc := errorCode1Descriptions[e.Code]; hasDesc {
		fmt.Fprintf(&builder, " (%s)", desc)
	}
	return builder.String()
}

// TPMSessionError is returned from DecodeResponseCode and TPMContext.RunCommand (and any other methods that wrap around this
// function) if the TPM response code indicates an error that is associated with a session.
type TPMSessionError struct {
	Command CommandCode // Command code associated with this error
	Code    ErrorCode1  // Error code
	Index   int         // Index of the session associated with this error in the authorization area, starting from 1
}

func (e TPMSessionError) Error() string {
	var builder bytes.Buffer
	fmt.Fprintf(&builder, "TPM returned an error for session %d whilst executing command %s: %s", e.Index, e.Command, e.Code)
	if desc, hasDesc := errorCode1Descriptions[e.Code]; hasDesc {
		fmt.Fprintf(&builder, " (%s)", desc)
	}
	return builder.String()
}

// TPMHandleError is returned from DecodeResponseCode and TPMContext.RunCommand (and any other methods that wrap around this function)
// if the TPM response code indicates an error that is associated with a command handle.
type TPMHandleError struct {
	Command CommandCode // Command code associated with this error
	Code    ErrorCode1  // Error code
	Index   int         // Index of the command handle associated with this error, starting from 1
}

func (e TPMHandleError) Error() string {
	var builder bytes.Buffer
	fmt.Fprintf(&builder, "TPM returned an error for handle %d whilst executing command %s: %s", e.Index, e.Command, e.Code)
	if desc, hasDesc := errorCode1Descriptions[e.Code]; hasDesc {
		fmt.Fprintf(&builder, " (%s)", desc)
	}
	return builder.String()
}

// DecodeResponseCode decodes the ResponseCode provided via resp. If the specified response code is Success, it returns no error,
// else it returns an error that is appropriate for the response code. The command code is used for adding context to the returned
// error.
func DecodeResponseCode(command CommandCode, resp ResponseCode) error {
	if resp == ResponseCode(Success) {
		return nil
	}

	if resp&formatMask == 0 {
		if resp&fmt0VersionMask == 0 {
			return TPM1Error{command, resp}
		}

		if resp&fmt0VendorMask > 0 {
			return TPMVendorError{command, resp}
		}

		if resp&fmt0SeverityMask > 0 {
			return TPMWarning{command, WarningCode(resp & fmt0ErrorCodeMask)}
		}

		return TPMError{command, ErrorCode0(resp & fmt0ErrorCodeMask)}
	}

	if resp&fmt1ParameterMask > 0 {
		return TPMParameterError{command, ErrorCode1(resp & fmt1ErrorCodeMask), int((resp & fmt1ParameterIndexMask) >> fmt1IndexShift)}
	}

	if resp&fmt1SessionMask > 0 {
		return TPMSessionError{command, ErrorCode1(resp & fmt1ErrorCodeMask), int((resp & fmt1HandleOrSessionIndexMask) >> fmt1IndexShift)}
	}

	return TPMHandleError{command, ErrorCode1(resp & fmt1ErrorCodeMask), int((resp & fmt1HandleOrSessionIndexMask) >> fmt1IndexShift)}
}
