// Copyright 2019 Canonical Ltd.
// Licensed under the LGPLv3 with static-linking exception.
// See LICENCE file for details.

package tpm2

// Section 29 - Clocks and Timers

// ReadClock executes the TPM2_ReadClock command. On succesful completion, it will return a TimeInfo struct that contains the current
// value of time, clock, reset and restart counts.
func (t *TPMContext) ReadClock(sessions ...SessionContext) (*TimeInfo, error) {
	var currentTime TimeInfo
	if err := t.RunCommand(CommandReadClock, sessions,
		Delimiter,
		Delimiter,
		Delimiter,
		&currentTime); err != nil {
		return nil, err
	}
	return &currentTime, nil
}

// func (t *TPMContext) ClockSet(auth Handle, newTime uint64, authAuth interface{}) error {
// }

// func (t *TPMContext) ClockRateAdjust(auth Handle, rateAdjust ClockAdjust, authAuth interface{}) error {
// }
