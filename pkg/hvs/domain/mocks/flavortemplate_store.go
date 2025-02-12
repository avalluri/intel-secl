/*
 * Copyright (C) 2021 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package mocks

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/intel-secl/intel-secl/v4/pkg/hvs/domain/models"
	commErr "github.com/intel-secl/intel-secl/v4/pkg/lib/common/err"
	"github.com/intel-secl/intel-secl/v4/pkg/model/hvs"
)

// MockFlavorTemplateStore provides a mocked implementation of interface hvs.FlavorTemplate
type MockFlavorTemplateStore struct {
	FlavorTemplates  []hvs.FlavorTemplate
	DeletedTemplates []hvs.FlavorTemplate
}

var flavorTemplate = `{
	"id": "426912bd-39b0-4daa-ad21-0c6933230b50",
	"label": "default-uefi",
	"condition": [
		"//host_info/vendor='Linux'",
		"//host_info/tpm_version='2.0'",
		"//host_info/uefi_enabled='true'",
		"//host_info/suefi_enabled='true'"
	],
	"flavor_parts": {
		"PLATFORM": {
			"meta": {
				"tpm_version": "2.0",
				"uefi_enabled": true,
				"vendor": "Linux"
			},
			"pcr_rules": [
				{
					"pcr": {
						"index": 0,
						"bank": "SHA256"
					},
					"pcr_matches": true,
					"eventlog_equals": {}
				}
			]
		},
		"OS": {
			"meta": {
				"tpm_version": "2.0",
				"uefi_enabled": true,
				"vendor": "Linux"
			},
			"pcr_rules": [
				{
					"pcr": {
						"index": 7,
						"bank": "SHA256"
					},
					"pcr_matches": true,
					"eventlog_includes": [
						"shim",
						"db",
						"kek",
						"vmlinuz"
					]
				}
			]
		}
	}
}`

// Create and inserts a Flavortemplate
func (store *MockFlavorTemplateStore) Create(ft *hvs.FlavorTemplate) (*hvs.FlavorTemplate, error) {

	if ft.ID == uuid.Nil {
		ft.ID = uuid.New()
	}

	store.FlavorTemplates = append(store.FlavorTemplates, *ft)

	return ft, nil
}

// Retrieve a Flavortemplate
func (store *MockFlavorTemplateStore) Retrieve(templateID uuid.UUID, includeDeleted bool) (*hvs.FlavorTemplate, error) {

	for _, template := range store.FlavorTemplates {
		if template.ID == templateID {
			return &template, nil
		}
	}
	return nil, &commErr.StatusNotFoundError{Message: "FlavorTemplate with given ID is not found"}
}

// Search a Flavortemplate(s)
func (store *MockFlavorTemplateStore) Search(criteria *models.FlavorTemplateFilterCriteria) ([]hvs.FlavorTemplate, error) {
	rec := store.FlavorTemplates
	if criteria.IncludeDeleted {
		rec = append(rec, store.DeletedTemplates...)
	}
	for _, template := range rec {
		//ID
		if criteria.Ids != nil {
			for _, id := range criteria.Ids {
				if template.ID == id {
					rec = append(rec, template)
					break
				}
			}
		}

		//Label
		if criteria.Label != "" {
			if template.Label == criteria.Label {
				rec = append(rec, template)
			}
		}

		//Condition
		if criteria.ConditionContains != "" {
			for _, condition := range template.Condition {
				if condition == criteria.ConditionContains {
					rec = append(rec, template)
				}
			}
		}

		//FlavorPart
		if criteria.FlavorPartContains != "" {
			if template.FlavorParts.Platform != nil && criteria.FlavorPartContains == "PLATFORM" {
				rec = append(rec, template)
			}
			if template.FlavorParts.OS != nil && criteria.FlavorPartContains == "OS" {
				rec = append(rec, template)
			}
			if template.FlavorParts.HostUnique != nil && criteria.FlavorPartContains == "HOST_UNIQUE" {
				rec = append(rec, template)
			}
		}
	}
	return rec, nil
}

// Detele a Flavortemplate
func (store *MockFlavorTemplateStore) Delete(templateID uuid.UUID) error {
	flavorTemplates := store.FlavorTemplates
	for i, template := range flavorTemplates {
		if template.ID == templateID {
			store.DeletedTemplates = append(store.DeletedTemplates, template)
			store.FlavorTemplates[i] = store.FlavorTemplates[len(store.FlavorTemplates)-1]
			store.FlavorTemplates = store.FlavorTemplates[:len(store.FlavorTemplates)-1]
			return nil
		}
	}
	return &commErr.StatusNotFoundError{Message: "FlavorTemplate with given ID is not found"}
}

// Recover a Flavortemplate
func (store *MockFlavorTemplateStore) Recover(labels []string) error {
	return nil
}

// NewFakeFlavorTemplateStore provides two dummy data for FlavorTemplates
func NewFakeFlavorTemplateStore() *MockFlavorTemplateStore {
	store := &MockFlavorTemplateStore{}

	var sf hvs.FlavorTemplate
	err := json.Unmarshal([]byte(flavorTemplate), &sf)
	fmt.Println("error: ", err)

	// add to store
	store.Create(&sf)

	return store
}

func (store *MockFlavorTemplateStore) AddFlavorgroups(uuid.UUID, []uuid.UUID) error {
	return nil
}
func (store *MockFlavorTemplateStore) RetrieveFlavorgroup(uuid.UUID, uuid.UUID) (*hvs.FlavorTemplateFlavorgroup, error) {
	return nil, nil
}
func (store *MockFlavorTemplateStore) RemoveFlavorgroups(uuid.UUID, []uuid.UUID) error {
	return nil
}

func (store *MockFlavorTemplateStore) SearchFlavorgroups(uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
