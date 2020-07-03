/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package hosttrust

import (
	"fmt"
	faultsConst "github.com/intel-secl/intel-secl/v3/pkg/hvs/constants/verifier-rules-and-faults"
	"github.com/intel-secl/intel-secl/v3/pkg/lib/flavor/common"
	"github.com/intel-secl/intel-secl/v3/pkg/lib/flavor/constants"
	"github.com/intel-secl/intel-secl/v3/pkg/lib/saml"
	"github.com/intel-secl/intel-secl/v3/pkg/model/hvs"
	"github.com/intel-secl/intel-secl/v3/pkg/model/ta"
	log "github.com/sirupsen/logrus"
	"reflect"
	"strconv"
	"strings"
)

type SamlReportGenerator struct {
	tagIssuer *saml.IssuerConfiguration
}

func NewSamlReportGenerator(tagIssuer *saml.IssuerConfiguration) *SamlReportGenerator {
	return &SamlReportGenerator{tagIssuer}
}

func (srg *SamlReportGenerator) GenerateSamlReport(report *hvs.TrustReport) saml.SamlAssertion {
	defaultLog.Trace("hosttrust/saml_report:generateSamlReport() Entering")
	defer defaultLog.Trace("hosttrust/saml_report:generateSamlReport() Leaving")

	libSaml, err := saml.NewLegacySAML(*srg.tagIssuer)
	if err != nil {
		log.WithError(err).Errorf("hosttrust/saml_report:generateSamlReport() Failed to instantiate SAML library")
	}
	mapFormatter := saml.NewLegacyMapFormatter(getSamlReportMap(report))
	assertion, err := libSaml.GenerateSamlAssertion(mapFormatter)
	if err != nil {
		log.WithError(err).Errorf("hosttrust/saml_report:generateSamlReport() Failed to generate SAML assertions")
	}
	return assertion
}

// load attributes map for saml report
func getSamlReportMap(t *hvs.TrustReport) map[string]string {
	defaultLog.Trace("hosttrust/saml_report:getSamlReportMap() Entering")
	defer defaultLog.Trace("hosttrust/saml_report:getSamlReportMap() Leaving")

	samlReportMap := make(map[string]string)

	for field, value := range getHostInfoMap(t.HostManifest.HostInfo) {
		samlReportMap[field] = value
	}
	for field, value := range getHardwareFeaturesMap(t.HostManifest.HostInfo.HardwareFeatures) {
		samlReportMap[field] = value
	}
	for field, value := range getMarkersMap(t) {
		samlReportMap[field] = value
	}
	if t.HostManifest.BindingKeyCertificate != "" {
		samlReportMap["Binding_Key_Certificate"] = t.HostManifest.BindingKeyCertificate
	}
	if t.HostManifest.AIKCertificate != "" {
		samlReportMap["AIK_Certificate"] = t.HostManifest.AIKCertificate
	}

	for field, value := range getTags(t) {
		samlReportMap[field] = value
	}

	return samlReportMap
}

// load markers map for saml report
func getMarkersMap(t *hvs.TrustReport) map[string]string {
	defaultLog.Trace("hosttrust/saml_report:getMarkersMap() Entering")
	defer defaultLog.Trace("hosttrust/saml_report:getMarkersMap() Leaving")

	trustedPrefix := "TRUST_"
	markersMap := make(map[string]string)
	for _, flavorType := range common.GetFlavorTypes() {
		marker := flavorType.String()
		if len(t.GetResultsForMarker(marker)) > 0 {
			markersMap[trustedPrefix+strings.ToUpper(marker)] = strconv.FormatBool(t.IsTrustedForMarker(marker))
		} else {
			markersMap[trustedPrefix+strings.ToUpper(marker)] = "NA"
		}
	}
	markersMap[trustedPrefix+"OVERALL"] = strconv.FormatBool(t.IsTrusted())
	return markersMap
}

// load host info map for saml report
func getHostInfoMap(hostInfo model.HostInfo) map[string]string {
	defaultLog.Trace("hosttrust/saml_report:getHostInfoMap() Entering")
	defer defaultLog.Trace("hosttrust/saml_report:getHostInfoMap() Leaving")

	hostValues := reflect.ValueOf(hostInfo)
	hostTypes := hostValues.Type()

	hostInfoMap := make(map[string]string)
	for i := 0; i < hostValues.NumField(); i++ {
		if hostTypes.Field(i).Name != "HardwareFeatures" {
			hostInfoMap[hostTypes.Field(i).Name] = fmt.Sprintf("%v", hostValues.Field(i).Interface())
		}
		defaultLog.Debugf("hosttrust/saml_report:getHostInfoMap() Field: %s\tValue: %v\n", hostTypes.Field(i).Name, hostValues.Field(i).Interface())
	}
	return hostInfoMap
}

// load hardware features map for saml report
func getHardwareFeaturesMap(features model.HardwareFeatures) map[string]string {
	defaultLog.Trace("hosttrust/saml_report:getHardwareFeaturesMap() Entering")
	defer defaultLog.Trace("hosttrust/saml_report:getHardwareFeaturesMap() Leaving")

	hwFeaturesMap := make(map[string]string)
	featurePrefix := "FEATURE_"
	if features.CBNT != nil && features.CBNT.Enabled {
		hwFeaturesMap[featurePrefix + constants.Cbnt] = strconv.FormatBool(features.CBNT.Enabled)
		hwFeaturesMap["FEATURE_cbntProfile"] = features.CBNT.Meta.Profile
	}
	if features.SUEFI != nil && features.SUEFI.Enabled {
		hwFeaturesMap[featurePrefix + constants.Suefi] = strconv.FormatBool(features.SUEFI.Enabled)
	}
	if features.TPM.Enabled {
		hwFeaturesMap[featurePrefix+ constants.Tpm] = strconv.FormatBool(features.TPM.Enabled)
	}
	if features.TXT != nil && features.TXT.Enabled {
		hwFeaturesMap[featurePrefix + constants.Txt] = strconv.FormatBool(features.TXT.Enabled)
	}
	return hwFeaturesMap
}

// load tags map for saml report
func getTags(trustReport *hvs.TrustReport) map[string]string {
	defaultLog.Trace("hosttrust/saml_report:getTags() Entering")
	defer defaultLog.Trace("hosttrust/saml_report:getTags() Leaving")

	tagsMap := make(map[string]string)
	for _, result := range trustReport.GetResultsForMarker(common.FlavorPartAssetTag.String()) {
		if result.Rule.Name == faultsConst.RuleAssetTagMatches {
			//TODO: Add tags in Asset Tag Rule
		}
	}
	return tagsMap
}
