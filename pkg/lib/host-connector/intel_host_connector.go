/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package host_connector

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"github.com/intel-secl/intel-secl/v4/pkg/model/hvs"

	client "github.com/intel-secl/intel-secl/v4/pkg/clients/ta"
	"github.com/intel-secl/intel-secl/v4/pkg/lib/host-connector/util"
	taModel "github.com/intel-secl/intel-secl/v4/pkg/model/ta"
	"github.com/pkg/errors"
	"github.com/vmware/govmomi/vim25/mo"
)

type IntelConnector struct {
	client client.TAClient
}

func (ic *IntelConnector) GetHostDetails() (taModel.HostInfo, error) {

	log.Trace("intel_host_connector:GetHostDetails() Entering")
	defer log.Trace("intel_host_connector:GetHostDetails() Leaving")
	hostInfo, err := ic.client.GetHostInfo()
	return hostInfo, err
}

func (ic *IntelConnector) GetHostManifest(pcrList []int) (hvs.HostManifest, error) {
	log.Trace("intel_host_connector:GetHostManifest() Entering")
	defer log.Trace("intel_host_connector:GetHostManifest() Leaving")

	nonce, err := util.GenerateNonce(20)
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifest() Error generating "+
			"nonce for TPM quote request")
	}

	hostManifest, err := ic.GetHostManifestAcceptNonce(nonce, pcrList)
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifest() Error creating "+
			"host manifest")
	}
	return hostManifest, nil
}

//Separate function has been created that accepts nonce to support unit test.
//Else it would be difficult to mock random nonce.
func (ic *IntelConnector) GetHostManifestAcceptNonce(nonce string, pcrList []int) (hvs.HostManifest, error) {
	log.Trace("intel_host_connector:GetHostManifestAcceptNonce() Entering")
	defer log.Trace("intel_host_connector:GetHostManifestAcceptNonce() Leaving")

	var verificationNonce string
	var hostManifest hvs.HostManifest

	//Hardcoded pcr list here since there is no use case for customized pcr list
	if pcrList == nil || len(pcrList) == 0 {
		log.Infof("intel_host_connector:GetHostManifestAcceptNonce() pcrList is empty")
		pcrList = []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}
	}

	//check if AIK Certificate is present on host before getting host manifest
	aikInDER, err := ic.client.GetAIK()
	if err != nil || len(aikInDER) == 0 {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Invalid AIK"+
			"certificate returned by TA")
	}
	secLog.Debug("intel_host_connector:GetHostManifestAcceptNonce() Successfully received AIK certificate in DER format")

	hostManifest.HostInfo, err = ic.client.GetHostInfo()
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Error getting "+
			"host details from TA")
	}

	tpmQuoteResponse, err := ic.client.GetTPMQuote(nonce, pcrList, []string{})
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Error getting TPM "+
			"quote response")
	}

	nonceInBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Base64 decode of TPM "+
			"nonce failed")
	}

	verificationNonce, err = util.GetVerificationNonce(nonceInBytes, tpmQuoteResponse)
	if err != nil {
		return hvs.HostManifest{}, err
	}
	secLog.Debug("intel_host_connector:GetHostManifestAcceptNonce() Updated Verification nonce is : ", verificationNonce)

	aikCertInBytes, err := base64.StdEncoding.DecodeString(tpmQuoteResponse.Aik)
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Error decoding"+
			"AIK certificate to bytes")
	}

	//Convert base64 encoded AIK to Pem format
	aikPem, _ := pem.Decode(aikCertInBytes)
	aikCertificate, err := x509.ParseCertificate(aikPem.Bytes)

	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Error parsing "+
			"AIK certicate")
	}

	tpmQuoteInBytes, err := base64.StdEncoding.DecodeString(tpmQuoteResponse.Quote)
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Error converting "+
			"tpm quote to bytes")
	}

	verificationNonceInBytes, err := base64.StdEncoding.DecodeString(verificationNonce)
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Error converting "+
			"nonce to bytes")
	}
	log.Info("intel_host_connector:GetHostManifestAcceptNonce() Verifying quote and retrieving PCR manifest from TPM quote " +
		"response ...")
	pcrManifest, pcrsDigest, err := util.VerifyQuoteAndGetPCRManifest(tpmQuoteResponse.EventLog, verificationNonceInBytes,
		tpmQuoteInBytes, aikCertificate)
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Error verifying "+
			"TPM Quote")
	}
	log.Info("intel_host_connector:GetHostManifestAcceptNonce() Successfully retrieved PCR manifest from quote")

	isWlaInstalled := false
	for _, component := range hostManifest.HostInfo.InstalledComponents {
		if component == taModel.HostComponentWlagent.String() {
			isWlaInstalled = true
			break
		}
	}

	bindingKeyCertificateBase64 := ""
	if hostManifest.HostInfo.IsDockerEnvironment {
		bindingKeyBytes, _ := ic.client.GetBindingKeyCertificate()
		if bindingKeyBytes != nil && len(bindingKeyBytes) != 0 {
			bindingKeyCertificate, _ := pem.Decode(bindingKeyBytes)
			if bindingKeyCertificate == nil {
				log.Warn("intel_host_connector:GetHostManifestAcceptNonce() - " +
					"Could not decode Binding key certificate. Unexpected response from client")
			} else {
				bindingKeyCertificateBase64 = base64.StdEncoding.EncodeToString(bindingKeyCertificate.Bytes)
			}
		} else {
			log.Warn("intel_host_connector:GetHostManifestAcceptNonce() " +
				"Empty Binding Key received")
		}
	} else if isWlaInstalled {
		bindingKeyBytes, err := ic.client.GetBindingKeyCertificate()
		if err != nil {
			return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() "+
				"Error getting binding key certificate from TA")
		}

		if bindingKeyBytes == nil || len(bindingKeyBytes) == 0 {
			return hvs.HostManifest{}, errors.New("intel_host_connector:GetHostManifestAcceptNonce() " +
				"Empty Binding Key received")
		}

		bindingKeyCertificate, _ := pem.Decode(bindingKeyBytes)
		if bindingKeyCertificate == nil {
			return hvs.HostManifest{}, errors.New("intel_host_connector:GetHostManifestAcceptNonce() - " +
				"Could not decode Binding key certificate. Unexpected response from client")
		}
		bindingKeyCertificateBase64 = base64.StdEncoding.EncodeToString(bindingKeyCertificate.Bytes)
	}
	aikCertificateBase64 := base64.StdEncoding.EncodeToString(aikPem.Bytes)

	hostManifest.PcrManifest = pcrManifest
	hostManifest.AIKCertificate = aikCertificateBase64
	hostManifest.AssetTagDigest = tpmQuoteResponse.AssetTag
	hostManifest.BindingKeyCertificate = bindingKeyCertificateBase64
	hostManifest.MeasurementXmls = tpmQuoteResponse.TcbMeasurements.TcbMeasurements
	hostManifest.QuoteDigest = hex.EncodeToString(pcrsDigest) + hostManifest.AssetTagDigest

	hostManifestJson, err := json.Marshal(hostManifest)
	if err != nil {
		return hvs.HostManifest{}, errors.Wrap(err, "intel_host_connector:GetHostManifestAcceptNonce() Error "+
			"marshalling host manifest to JSON")
	}
	log.Debugf("intel_host_connector:GetHostManifestAcceptNonce() Host Manifest : %s", string(hostManifestJson))
	log.Info("intel_host_connector:GetHostManifestAcceptNonce() Host manifest created successfully")
	return hostManifest, err
}

func (ic *IntelConnector) DeployAssetTag(hardwareUUID, tag string) error {

	log.Trace("intel_host_connector:DeployAssetTag() Entering")
	defer log.Trace("intel_host_connector:DeployAssetTag() Leaving")
	err := ic.client.DeployAssetTag(hardwareUUID, tag)
	return err
}

func (ic *IntelConnector) DeploySoftwareManifest(manifest taModel.Manifest) error {

	log.Trace("intel_host_connector:DeploySoftwareManifest() Entering")
	defer log.Trace("intel_host_connector:DeploySoftwareManifest() Leaving")
	err := ic.client.DeploySoftwareManifest(manifest)
	return err
}

func (ic *IntelConnector) GetMeasurementFromManifest(manifest taModel.Manifest) (taModel.Measurement, error) {

	log.Trace("intel_host_connector:GetMeasurementFromManifest() Entering")
	defer log.Trace("intel_host_connector:GetMeasurementFromManifest() Leaving")
	measurement, err := ic.client.GetMeasurementFromManifest(manifest)
	return measurement, err
}

func (ic *IntelConnector) GetClusterReference(clusterName string) ([]mo.HostSystem, error) {
	return nil, errors.New("intel_host_connector :GetClusterReference() Operation not supported")
}
