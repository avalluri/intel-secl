/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package rules

import (
	commLog "github.com/intel-secl/intel-secl/v4/pkg/lib/common/log"
	"github.com/intel-secl/intel-secl/v4/pkg/model/hvs"
)

// This is the interface that a rule must implement to perform
// verification against the data in a host manifest.
type Rule interface {
	Apply(hostManifest *hvs.HostManifest) (*hvs.RuleResult, error)
}

var log = commLog.GetDefaultLogger()
var secLog = commLog.GetSecurityLogger()
