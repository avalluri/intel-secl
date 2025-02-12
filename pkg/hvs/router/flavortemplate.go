/*
 * Copyright (C) 2021 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package router

import (
	"fmt"

	"github.com/gorilla/mux"
	"github.com/intel-secl/intel-secl/v4/pkg/hvs/constants"
	"github.com/intel-secl/intel-secl/v4/pkg/hvs/controllers"
	"github.com/intel-secl/intel-secl/v4/pkg/hvs/postgres"
	"github.com/intel-secl/intel-secl/v4/pkg/lib/common/validation"
)

// SetFlavorTemplateRoutes registers routes for flavor template creation
func SetFlavorTemplateRoutes(router *mux.Router, store *postgres.DataStore, flavorGroupStore *postgres.FlavorGroupStore) *mux.Router {
	defaultLog.Trace("router/flavortemplate_creation:SetFlavorTemplateRoutes() Entering")
	defer defaultLog.Trace("router/flavortemplate_creation:SetFlavorTemplateRoutes() Leaving")

	flavorTemplateStore := postgres.NewFlavorTemplateStore(store)

	flavorTemplateController := controllers.NewFlavorTemplateController(flavorTemplateStore, flavorGroupStore, constants.CommonDefinitionsSchema, constants.FlavorTemplateSchema)

	flavorTemplateIdExpr := fmt.Sprintf("%s/{ftId:%s}", "/flavor-templates", validation.UUIDReg)
	flavorgroupExpr := fmt.Sprintf("%s/flavorgroups", flavorTemplateIdExpr)
	flavorgroupIdExpr := fmt.Sprintf("%s/{fgId:%s}", flavorgroupExpr, validation.UUIDReg)

	router.Handle("/flavor-templates",
		ErrorHandler(permissionsHandler(JsonResponseHandler(flavorTemplateController.Create),
			[]string{constants.FlavorTemplateCreate}))).Methods("POST")

	router.Handle(flavorTemplateIdExpr,
		ErrorHandler(permissionsHandler(JsonResponseHandler(flavorTemplateController.Retrieve),
			[]string{constants.FlavorTemplateRetrieve}))).Methods("GET")

	router.Handle("/flavor-templates",
		ErrorHandler(permissionsHandler(JsonResponseHandler(flavorTemplateController.Search),
			[]string{constants.FlavorTemplateSearch}))).Methods("GET")

	router.Handle(flavorTemplateIdExpr,
		ErrorHandler(permissionsHandler(JsonResponseHandler(flavorTemplateController.Delete),
			[]string{constants.FlavorTemplateDelete}))).Methods("DELETE")

	router.Handle(flavorgroupExpr, ErrorHandler(permissionsHandler(JsonResponseHandler(flavorTemplateController.AddFlavorgroup),
		[]string{constants.FlavorTemplateCreate}))).Methods("POST")
	router.Handle(flavorgroupIdExpr, ErrorHandler(permissionsHandler(JsonResponseHandler(flavorTemplateController.RetrieveFlavorgroup),
		[]string{constants.FlavorTemplateRetrieve}))).Methods("GET")
	router.Handle(flavorgroupIdExpr, ErrorHandler(permissionsHandler(ResponseHandler(flavorTemplateController.RemoveFlavorgroup),
		[]string{constants.FlavorTemplateDelete}))).Methods("DELETE")
	router.Handle(flavorgroupExpr, ErrorHandler(permissionsHandler(JsonResponseHandler(flavorTemplateController.SearchFlavorgroups),
		[]string{constants.FlavorTemplateSearch}))).Methods("GET")

	return router
}
