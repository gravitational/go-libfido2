# Why fork?

This repository is a fork of
[keys-pub/go-libfido2](https://github.com/keys-pub/go-libfido2/), focused on the
needs of [Teleport](https://github.com/gravitational/teleport).

The fork exists because:

1. It allows us to experiment and land required changes to go-libfido2 faster;
   and
2. It allows us to tweak CGO build directives for the Teleport binaries.

We do intend to contribute widely valuable changes to the original repo, meaning
that over time (1) should become a non-issue. If the CGO build changes turn out
to be widely applicable (and accepted by the upstream repo), then the fork
becomes obsolete.

The `master` branch reflects the upstream master.

The `teleport` branch is the default and reflects the library version used by
Teleport. There is no concept of semantic versioning for the `teleport` branch,
it rolls forward as Teleport needs it to.

All builds are dynamic by default (Linux and macOS). You may use the
`libfido2static` build tag to force static builds instead. The following
libraries are required for static builds:

* /usr/local/lib/libfido2.a
* /usr/local/lib/libcbor.a
* /usr/lib/x86_64-linux-gnu/libcrypto.a (Linux)
* /usr/local/opt/openssl@1.1/lib/libcrypto.a (macOS)
* /usr/local/lib/libudev.a (Linux,
  [libeudev](https://github.com/eudev-project/eudev))

Other system libraries are linked dynamically (such as `libudev` and `libdl` on
Linux).

You are looking at the `teleport` branch now.

# go-libfido2

Go wrapper for libfido2.

```go
import (
    "github.com/keys-pub/go-libfido2"
)

func ExampleDevice_Assertion() {
    locs, err := libfido2.DeviceLocations()
    if err != nil {
        log.Fatal(err)
    }
    if len(locs) == 0 {
        log.Println("No devices")
        return
    }

    log.Printf("Using device: %+v\n", locs[0])
    path := locs[0].Path
    device, err := libfido2.NewDevice(path)
    if err != nil {
        log.Fatal(err)
    }

    cdh := libfido2.RandBytes(32)
    userID := libfido2.RandBytes(32)
    salt := libfido2.RandBytes(32)
    pin := "12345"

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
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Attestation:\n")
    log.Printf("AuthData: %s\n", hex.EncodeToString(attest.AuthData))
    log.Printf("ClientDataHash: %s\n", hex.EncodeToString(attest.ClientDataHash))
    log.Printf("ID: %s\n", hex.EncodeToString(attest.CredentialID))
    log.Printf("Type: %s\n", attest.CredentialType)
    log.Printf("Sig: %s\n", hex.EncodeToString(attest.Sig))

    assertion, err := device.Assertion(
        "keys.pub",
        cdh,
        [][]byte{attest.CredentialID},
        pin,
        &libfido2.AssertionOpts{
            Extensions: []libfido2.Extension{libfido2.HMACSecretExtension},
            HMACSalt:   salt,
        },
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Assertion:\n")
    log.Printf("%s\n", hex.EncodeToString(assertion.AuthData))
    log.Printf("%s\n", hex.EncodeToString(assertion.HMACSecret))
    log.Printf("%s\n", hex.EncodeToString(assertion.Sig))

    // Output:
    //
}
```

## Examples

The examples require a device.

To run an example, set FIDO2_EXAMPLES=1.

```shell
FIDO2_EXAMPLES=1 go test -v -run ExampleDeviceLocations
FIDO2_EXAMPLES=1 go test -v -run ExampleDevice_Assertion
FIDO2_EXAMPLES=1 go test -v -run ExampleDevice_Credentials
```

## Dependencies

### Linux

```shell
sudo apt install software-properties-common
sudo apt-add-repository ppa:yubico/stable
sudo apt update
sudo apt install libfido2-dev
```

### macOS

```shell
brew install keys-pub/tap/libfido2
```

### Windows

```shell
scoop bucket add keys.pub https://github.com/keys-pub/scoop-bucket
scoop install libfido2
```


### Building libfido2

#### macOS

```shell
export CFLAGS="-I/usr/local/include -I/usr/local/opt/openssl@1.1/include"
export LDFLAGS="-L/usr/local/lib -L/usr/local/opt/openssl@1.1/lib/"
(rm -rf build && mkdir build && cd build && cmake ..) && make -C build
```
