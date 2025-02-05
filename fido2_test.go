package libfido2_test

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/keys-pub/go-libfido2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// TODO(codingllama): Revisit test suite.
//  It can't run in an automated manner and is rather difficult to run manually
//  (without editing files).

// TODO: It's important tests are run serially (a device can't handle concurrent requests).

func getPIN() string {
	return os.Getenv("FIDO2_PIN")
}

func TestDevices(t *testing.T) {
	locs, err := libfido2.DeviceLocations()
	require.NoError(t, err)
	t.Logf("Found %d devices", len(locs))

	for _, loc := range locs {
		device, err := libfido2.NewDevice(loc.Path)
		require.NoError(t, err)
		defer device.Close()

		isFIDO2, err := device.IsFIDO2()
		require.NoError(t, err)
		if !isFIDO2 {
			continue
		}

		typ, err := device.Type()
		require.NoError(t, err)
		require.Equal(t, libfido2.FIDO2, typ)

		// Testing info twice (hid_osx issues in the past caused a delayed 2nd request to fail).
		info, err := device.Info()
		require.NoError(t, err)
		time.Sleep(time.Millisecond * 100)

		info, err = device.Info()
		require.NoError(t, err)
		t.Logf("Info: %+v", info)
	}
}

func TestDeviceAssertionCancel(t *testing.T) {
	locs, err := libfido2.DeviceLocations()
	require.NoError(t, err)
	if len(locs) == 0 {
		t.Skip("No devices")
	}

	t.Logf("Using device: %+v\n", locs[0])
	path := locs[0].Path
	device, err := libfido2.NewDevice(path)
	if err != nil {
		log.Fatal(err)
	}
	defer device.Close()

	cdh := libfido2.RandBytes(32)
	userID := libfido2.RandBytes(32)
	salt := libfido2.RandBytes(32)
	pin := getPIN()

	t.Log("Touch your device")
	attest, err := device.MakeCredential(
		cdh,
		libfido2.RelyingParty{
			ID: "keys.pub",
		},
		libfido2.User{
			ID:   userID,
			Name: "gabriel",
		},
		libfido2.ES256, // Algorithm
		pin,
		&libfido2.MakeCredentialOpts{
			Extensions: []libfido2.Extension{libfido2.HMACSecretExtension},
			RK:         libfido2.True,
		},
	)
	require.NoError(t, err)

	go func() {
		time.Sleep(time.Second * 2)
		t.Log("Cancel")
		device.Cancel()
	}()

	t.Log("DON'T touch your device")
	_, err = device.Assertion(
		"keys.pub",
		cdh,
		[][]byte{attest.CredentialID},
		pin,
		&libfido2.AssertionOpts{
			Extensions: []libfido2.Extension{libfido2.HMACSecretExtension},
			UP:         libfido2.True,
			HMACSalt:   salt,
		},
	)
	require.EqualError(t, errors.Cause(err), "keep alive cancel")
}

func TestDevice_TouchRequest(t *testing.T) {
	locs, err := libfido2.DeviceLocations()
	if err != nil {
		t.Fatalf("DeviceLocations failed: %v", err)
	}
	if len(locs) == 0 {
		t.Fatalf("No devices found")
	}
	loc := locs[0]

	t.Logf("Using device: %+v\n", loc)
	dev, err := libfido2.NewDevice(loc.Path)
	if err != nil {
		t.Fatalf("NewDevice failed: %v", err)
	}
	defer dev.Close()

	t.Run("success", func(t *testing.T) {
		t.Logf("Touch your %v\n", locs[0].Product)
		touch, err := dev.TouchBegin()
		if err != nil {
			t.Fatalf("TouchBegin failed: %v", err)
		}

		maxWait := time.After(30 * time.Second)
		for {
			touched, err := touch.Status(200 * time.Second)
			if err != nil {
				t.Errorf("Status failed, aborting: %v", err)
				break
			}
			if touched {
				t.Log("Touch detected")
				break
			}

			select {
			case <-maxWait:
				// Exit select and break from loop below.
			default:
				continue
			}
			t.Error("Timed out waiting for touch")
			break
		}

		if err := touch.Stop(); err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		if err := touch.Stop(); err != nil {
			t.Errorf("Subsequent Stops should never error: %v", err)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		t.Log("Testing touch cancel")

		touch, err := dev.TouchBegin()
		if err != nil {
			t.Fatalf("TouchBegin failed: %v", err)
		}

		// Give it a moment to start.
		time.Sleep(2 * time.Second)

		// Terminate touch request.
		if err := touch.Stop(); err != nil {
			t.Errorf("Stop failed: %v", err)
		}
		if err := touch.Stop(); err != nil {
			t.Errorf("Subsequent Stops should never error: %v", err)
		}
	})
}
