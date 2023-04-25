package kpmenulib

import (
	"testing"

	"github.com/tobischo/gokeepasslib/v3"
)

func Test_CreateOTP(t *testing.T) {
	a := gokeepasslib.NewEntry()
	a.Values = append(a.Values, gokeepasslib.ValueData{
		Key: OTP, Value: gokeepasslib.V{
			Content: "otpauth://totp/github:test?secret=NBSXEZLTMF2GK43UON2HE2LOM4FA====&period=30&digits=6&issuer=github",
		},
	})
	got, err := CreateOTP(a, 0)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if got != "717299" {
		t.Errorf("expected %q, got %q", "717299", got)
	}

	a = gokeepasslib.NewEntry()
	a.Values = append(a.Values, gokeepasslib.ValueData{
		Key: OTP, Value: gokeepasslib.V{
			Content: "otpauth://totp/buhtig:tset?secret=NBSXEZLTMF2GK43UON2HE2LOM4FA====&digits=6&issuer=Homeassistant",
		},
	})
	got, err = CreateOTP(a, 123456789)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if got != "045678" {
		t.Errorf("expected %q, got %q", "045678", got)
	}
}