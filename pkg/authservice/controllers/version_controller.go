/*
 * Copyright (C) 2019 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package controllers

import (
	"github.com/intel-secl/intel-secl/v4/pkg/authservice/version"
	"net/http"
)

type VersionController struct {
}

func (controller VersionController) GetVersion(w http.ResponseWriter, r *http.Request) (interface{}, int, error) {
	defaultLog.Trace("controllers/version:getVersion() Entering")
	defer defaultLog.Trace("controllers/version:getVersion() Leaving")

	w.Header().Add("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	return version.GetVersion(), http.StatusOK, nil
}
