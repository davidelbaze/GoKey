// https://github.com/f-secure-foundry/GoKey
//
// Copyright (c) F-Secure Corporation
// https://foundry.f-secure.com
//
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// +build tamago,arm

package main

import (
	"fmt"
	"log"

	"github.com/f-secure-foundry/GoKey/internal"
	"github.com/f-secure-foundry/GoKey/internal/ccid"
	"github.com/f-secure-foundry/GoKey/internal/icc"
	"github.com/f-secure-foundry/GoKey/internal/usb"

	"github.com/f-secure-foundry/tamago/imx6"
	imxusb "github.com/f-secure-foundry/tamago/imx6/usb"
	_ "github.com/f-secure-foundry/tamago/usbarmory/mark-two"
)

const IP = "10.0.0.10"
const hostMAC = "1a:55:89:a2:69:42"
const deviceMAC = "1a:55:89:a2:69:41"

func init() {
	if !imx6.Native {
		return
	}

	if err := imx6.SetARMFreq(900000000); err != nil {
		panic(fmt.Sprintf("WARNING: error setting ARM frequency: %v\n", err))
	}
}

func main() {
	log.Println(gokey.Banner)

	device := &imxusb.Device{}
	usb.ConfigureDevice(device)

	// Initialize an OpenPGP card with the bundled key information (defined
	// in `keys.go` and generated at compilation time).
	card := &icc.Interface{
		SNVS:       SNVS,
		ArmoredKey: pgpSecretKey,
		Name:       NAME,
		Language:   LANGUAGE,
		Sex:        SEX,
		URL:        URL,
		Debug:      false,
	}

	if initAtBoot {
		err := card.Init()

		if err != nil {
			log.Printf("card initialization error: %v", err)
		}
	}

	// initialize CCID interface
	reader := &ccid.Interface{
		ICC: card,
	}

	if imx6.Native {
		// set card serial number to 2nd half of NXP Unique ID
		uid := imx6.UniqueID()
		copy(card.Serial[0:4], uid[4:8])

		// configure Smart Card over USB endpoints (CCID protocol)
		usb.ConfigureCCID(device, reader)
	}

	// start basic networking
	stack, link := gokey.StartNetworking(deviceMAC, IP)

	if len(sshPublicKey) != 0 {
		started := make(chan bool)

		// start SSH server for management console
		err := gokey.StartSSHServer(stack, IP, sshPublicKey, sshPrivateKey, card, started)

		if err != nil {
			log.Printf("SSH server initialization error: %v", err)
		}

		// wait for ssh server to start before responding to USB requests
		<-started
	}

	if !imx6.Native {
		return
	}

	// Configure Ethernet over USB endpoints
	// (ECM protocol, only supported on Linux hosts).
	usb.ConfigureECM(device, hostMAC, deviceMAC, link)

	imxusb.USB1.Init()
	imxusb.USB1.DeviceMode()
	imxusb.USB1.Reset()

	// never returns
	imxusb.USB1.Start(device)
}
