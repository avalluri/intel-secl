/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package postgres

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/constants"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain/models"
	"github.com/intel-secl/intel-secl/v3/pkg/model/hvs"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"reflect"
	"strings"
	"time"
)

type HostStatusStore struct {
	Store *DataStore
}

func NewHostStatusStore(store *DataStore) *HostStatusStore {
	return &HostStatusStore{store}
}

// Create creates a HostStatus record in the DB
func (hss *HostStatusStore) Create(hs *hvs.HostStatus) (*hvs.HostStatus, error) {
	defaultLog.Trace("postgres/hoststatus_store:Create() Entering")
	defer defaultLog.Trace("postgres/hoststatus_store:Create() Leaving")

	dbHostStatus := hostStatus{
		ID:         uuid.New(),
		HostID:     hs.HostID,
		Status:     PGHostStatusInformation(hs.HostStatusInformation),
		HostReport: PGHostManifest(hs.HostManifest),
		CreatedAt:  time.Now(),
	}

	if err := hss.Store.Db.Create(&dbHostStatus).Error; err != nil {
		return nil, errors.Wrap(err, "postgres/hoststatus_store:Create() failed to create hostStatus")
	}

	return hs, nil
}

// Retrieve retrieves a single HostStatus record matching a provided hostStatusId
func (hss *HostStatusStore) Retrieve(hostStatusId uuid.UUID) (*hvs.HostStatus, error) {
	defaultLog.Trace("postgres/hoststatus_store:Retrieve() Entering")
	defer defaultLog.Trace("postgres/hoststatus_store:Retrieve() Leaving")

	dbHostStatus := hostStatus{
		ID: hostStatusId,
	}

	row := hss.Store.Db.Model(&dbHostStatus).Where(&dbHostStatus).Row()
	result := hvs.HostStatus{}
	if err := row.Scan(&result.ID, &result.HostID, (*PGHostStatusInformation)(&result.HostStatusInformation), &result.Created, (*PGHostManifest)(&result.HostManifest)); err != nil {
		return nil, errors.Wrap(err, "postgres/hoststatus_store:Retrieve() failed to retrieve hostStatus")
	}

	return &result, nil
}

// Search retrieves a HostStatusCollection pertaining to a user-provided HostStatusFilterCriteria
func (hss *HostStatusStore) Search(hsFilter *models.HostStatusFilterCriteria) ([]hvs.HostStatus, error) {
	defaultLog.Trace("postgres/hoststatus_store:Search() Entering")
	defer defaultLog.Trace("postgres/hoststatus_store:Search() Leaving")

	var tx *gorm.DB

	// setting to empty array
	hostStatuses := []hvs.HostStatus{}

	if hsFilter.FromDate.IsZero() && hsFilter.ToDate.IsZero() && hsFilter.LatestPerHost {
		tx = buildLatestHostStatusSearchQuery(hss.Store.Db, hsFilter)
		if tx == nil {
			return nil, errors.New("postgres/hoststatus_store:Search() Unexpected Error. Could not build" +
				" a gorm query object in HostStatus Search function.")
		}

		rows, err := tx.Rows()
		if err != nil {
			return nil, errors.Wrap(err, "postgres/hoststatus_store:Search() failed to retrieve records from db")
		}
		defer rows.Close()

		for rows.Next() {
			var result hvs.HostStatus

			if err := rows.Scan(&result.ID, &result.HostID, (*PGHostStatusInformation)(&result.HostStatusInformation), (*PGHostManifest)(&result.HostManifest), &result.Created); err != nil {
				return nil, errors.Wrap(err, "postgres/hoststatus_store:Search() failed to scan record")
			}
			hostStatuses = append(hostStatuses, result)
		}
	} else {
		tx = buildHostStatusSearchQuery(hss.Store.Db, hsFilter)
		if tx == nil {
			return nil, errors.New("postgres/hoststatus_store:Search() Unexpected Error. Could not build" +
				" a gorm query object in HostStatus Search function.")
		}

		rows, err := tx.Rows()
		if err != nil {
			return nil, errors.Wrap(err, "postgres/hoststatus_store:Search() failed to retrieve records from db")
		}
		defer rows.Close()

		for rows.Next() {
			result := models.AuditLogEntry{}
			if err := rows.Scan(&result.ID, &result.EntityID, &result.EntityType, &result.CreatedAt, &result.Action, (*PGAuditLogData)(&result.Data)); err != nil {
				return nil, errors.Wrap(err, "postgres/hoststatus_store:Search() failed to scan record")
			}
			if reflect.DeepEqual(models.AuditTableData{}, result.Data) || len(result.Data.Columns) == 0 {
				continue
			}
			hs, err := auditlogEntryToHostStatus(result)
			if err != nil {
				return nil, errors.Wrap(err, "postgres/hoststatus_store:Search() convert auditlog entry into HostStatus")
			}
			hostStatuses = append(hostStatuses, *hs)
		}
	}

	return hostStatuses, nil
}

func (hss *HostStatusStore) Update(hs *hvs.HostStatus) error {
	defaultLog.Trace("postgres/hoststatus_store:Update() Entering")
	defer defaultLog.Trace("postgres/hoststatus_store:Update() Leaving")

	if hs.ID == uuid.Nil {
		return errors.New("postgres/hoststatus_store:Update() - ID is invalid")
	}

	dbHostStatus := hostStatus{
		ID:         hs.ID,
		HostID:     hs.HostID,
		Status:     PGHostStatusInformation(hs.HostStatusInformation),
		HostReport: PGHostManifest(hs.HostManifest),
	}

	if db := hss.Store.Db.Model(&dbHostStatus).Updates(&dbHostStatus); db.Error != nil || db.RowsAffected != 1 {
		if db.Error != nil {
			return errors.Wrap(db.Error, "postgres/hoststatus_store:Update() failed to update HostStatus  "+hs.ID.String())
		} else {
			return errors.New("postgres/hoststatus_store:Update() - no rows affected - Record not found = id :  " + hs.ID.String())
		}

	}
	return nil
}

func (hss *HostStatusStore) Delete(hostStatusId uuid.UUID) error {
	defaultLog.Trace("postgres/hoststatus_store:Delete() Entering")
	defer defaultLog.Trace("postgres/hoststatus_store:Delete() Leaving")

	dbHostStatus := hostStatus{
		ID: hostStatusId,
	}
	if err := hss.Store.Db.Delete(&dbHostStatus).Error; err != nil {
		return errors.Wrap(err, "postgres/hoststatus_store:Delete() failed to delete HostStatus")
	}
	return nil
}

func (hss *HostStatusStore) FindHostIdsByKeyValue(key, value string) ([]uuid.UUID, error) {
	defaultLog.Trace("postgres/hoststatus_store:FindHostIdsByKeyValue() Entering")
	defer defaultLog.Trace("postgres/hoststatus_store:FindHostIdsByKeyValue() Leaving")

	rows, err := hss.Store.Db.Raw("SELECT host_id FROM host_status WHERE host_report::text != 'null' AND host_report -> 'host_info' ->> ? = ?", key, value).Rows()
	if err != nil {
		return nil, errors.Wrap(err, "postgres/hoststatus_store:FindHostIdsByKeyValue() failed to retrieve records from db")
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		id := uuid.UUID{}
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Wrap(err, "postgres/hoststatus_store:FindHostIdsByKeyValue() failed to scan record")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// buildHostStatusSearchQuery is a helper function to build the query object for a hostStatus search inlcuding results
// from audit table hostStatus records
func buildHostStatusSearchQuery(tx *gorm.DB, hsFilter *models.HostStatusFilterCriteria) *gorm.DB {
	defaultLog.Trace("postgres/hoststatus_store:buildHostStatusSearchQuery() Entering")
	defer defaultLog.Trace("postgres/hoststatus_store:buildHostStatusSearchQuery() Leaving")

	var tableJoinString, additionalOptionsQueryString string

	// define joins
	auditLogAbbrv := "au"
	if hsFilter.LatestPerHost {
		auditLogAbbrv = "auj"
	}

	formattedQuery := "SELECT au.* FROM audit_log_entry au"

	additionalOptionsQueryString = fmt.Sprintf("WHERE %s.entity_type = 'HostStatus' ", auditLogAbbrv)

	//Build table join string with host table if host identifier is set
	if hsFilter.HostName != "" {
		tableJoinString = fmt.Sprintf("INNER JOIN host h on CAST(h.id AS VARCHAR) = %s.data -> 'columns' -> 1 ->> 'value'", auditLogAbbrv)
	}

	//Build additional options query string is table join string set
	if tableJoinString != "" {
		//Build host ID partial query string and add it to the additional options query string
		if hsFilter.HostId != uuid.Nil {
			hostIdQueryString := fmt.Sprintf("h.id = '%s'", hsFilter.HostId.String())
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, hostIdQueryString)
		}
		//calling hostIdentifierQueryString method to improve the performance by referring to the host table when host identifier is set
		if hsFilter.HostName != "" {
			hostNameQueryString := fmt.Sprintf("h.name = '%s'", hsFilter.HostName)
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, hostNameQueryString)
		}
		if hsFilter.HostHardwareId != uuid.Nil {
			hostHWUUIDQueryString := fmt.Sprintf("h.hardware_uuid = '%s'", hsFilter.HostHardwareId.String())
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, hostHWUUIDQueryString)
		}
	} else {
		//Build host ID partial query string and add it to the additional options query string
		if hsFilter.HostId != uuid.Nil {
			hostIdQueryString := fmt.Sprintf("%s.data -> 'columns' -> 1 ->> 'value' = '%s'", auditLogAbbrv, hsFilter.HostId.String())
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, hostIdQueryString)
		}

		//Build host name partial query string and add it to the additional options query string
		if hsFilter.HostName != "" {
			hostNameQueryString := fmt.Sprintf("%s.data -> 'columns' -> 4 -> 'value' -> 'host_info' ->> 'host_name' = '%s'", auditLogAbbrv, hsFilter.HostName)
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, hostNameQueryString)
		}

		//Build hardware uuid partial query string and add it to the additional options query string
		if hsFilter.HostHardwareId != uuid.Nil {
			hostHWUUIDQueryString := fmt.Sprintf("LOWER(%s.data -> 'columns' -> 4 -> 'value' -> 'host_info' ->> 'hardware_uuid') = '%s' ", auditLogAbbrv, strings.ToLower(hsFilter.HostHardwareId.String()))
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, hostHWUUIDQueryString)
		}
	}

	//Build host state partial query string and add it to the additional options query string
	if hsFilter.HostStatus != "" {
		hostStateQueryString := fmt.Sprintf("%s.data -> 'columns' -> 2 -> 'value' ->> 'host_state' = '%s'", auditLogAbbrv, strings.ToUpper(hsFilter.HostStatus))
		additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, hostStateQueryString)
	}

	//Build host status ID partial query string and add it to the additional options query string
	if hsFilter.Id != uuid.Nil {
		hostStatusIDQueryString := fmt.Sprintf("%s.entity_id = '%s'", auditLogAbbrv, hsFilter.Id.String())
		additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, hostStatusIDQueryString)
	}

	// Number of days and Date Filters are supposed to be mutually exclusive
	if hsFilter.NumberOfDays != 0 {
		// first parse numDays
		curTime := time.Now()
		hsFilter.FromDate = curTime.AddDate(0, 0, -hsFilter.NumberOfDays)
	}

	if !hsFilter.FromDate.IsZero() || !hsFilter.ToDate.IsZero() {
		// determine what dates params are set - try all combinations till one matches up
		if !hsFilter.FromDate.IsZero() && hsFilter.ToDate.IsZero() {
			fromDateQueryString := fmt.Sprintf("CAST(%s.created AS TIMESTAMP) >= CAST('%s' AS TIMESTAMP)", auditLogAbbrv, hsFilter.FromDate.Format(constants.ParamDateFormatUTC))
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, fromDateQueryString)
		} else if hsFilter.FromDate.IsZero() && !hsFilter.ToDate.IsZero() {
			toDateQueryString := fmt.Sprintf("CAST(%s.created AS TIMESTAMP) <= CAST('%s' AS TIMESTAMP)", auditLogAbbrv, hsFilter.ToDate.Format(constants.ParamDateFormatUTC))
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, toDateQueryString)
		} else if !hsFilter.FromDate.IsZero() && !hsFilter.ToDate.IsZero() {
			fromToDateQueryString := fmt.Sprintf("CAST(%s.created AS TIMESTAMP) >= CAST('%s' AS TIMESTAMP) AND CAST(%s.created AS TIMESTAMP) <= CAST('%s' AS TIMESTAMP) ", auditLogAbbrv, hsFilter.FromDate.Format(constants.ParamDateFormatUTC), auditLogAbbrv, hsFilter.ToDate.Format(constants.ParamDateFormatUTC))
			additionalOptionsQueryString = fmt.Sprintf("%s AND %s", additionalOptionsQueryString, fromToDateQueryString)
		}
	}

	if tableJoinString != "" {
		additionalOptionsQueryString = strings.Join([]string{tableJoinString, additionalOptionsQueryString}, " ")
	}

	if hsFilter.LatestPerHost {
		maxDateQueryString := fmt.Sprintf("INNER JOIN (SELECT entity_id, max(auj.created) AS max_date"+
			"FROM audit_log_entry auj %s GROUP BY entity_id) a "+
			"ON a.entity_id = au.entity_id "+
			"AND a.max_date = au.created", additionalOptionsQueryString)
		formattedQuery = fmt.Sprintf("%s %s ORDER BY au.Created DESC", formattedQuery, maxDateQueryString)
	} else {
		formattedQuery = fmt.Sprintf("%s %s", formattedQuery, additionalOptionsQueryString)
	}

	// finalize query
	tx = tx.Raw(formattedQuery).Limit(hsFilter.Limit)

	return tx
}

// buildLatestHostStatusSearchQuery is a helper function to build the query object for a hostStatus search
// with the LatestPerHost filter set to true
func buildLatestHostStatusSearchQuery(tx *gorm.DB, hsFilter *models.HostStatusFilterCriteria) *gorm.DB {
	defaultLog.Trace("postgres/hoststatus_store:buildLatestHostStatusSearchQuery() Entering")
	defer defaultLog.Trace("postgres/hoststatus_store:buildLatestHostStatusSearchQuery() Leaving")

	tx = tx.Model(&hostStatus{})
	if tx == nil {
		return nil
	}

	// no criteria are specified
	if hsFilter == nil {
		defaultLog.Info("postgres/hoststatus_store:buildLatestHostStatusSearchQuery() No criteria specified in search query" +
			". Returning all rows.")
		return tx
	}

	// Host Status ID
	if hsFilter.Id != uuid.Nil {
		tx = tx.Where("id = ?", hsFilter.Id)
	}

	// Host UUID
	if hsFilter.HostId != uuid.Nil {
		tx = tx.Where("host_id = ?", hsFilter.HostId)
	}

	// HWUUID
	if hsFilter.HostHardwareId != uuid.Nil {
		tx = tx.Where(`host_report @> '{"host_info": {"hardware_uuid": "` + hsFilter.HostHardwareId.String() + `"}}'`)
	}

	// HostName
	if hsFilter.HostName != "" {
		tx = tx.Where(`host_report @> '{"host_info": {"host_name": "` + hsFilter.HostName + `"}}'`)
	}

	// Host Connection Status
	if hsFilter.HostStatus != "" {
		tx = tx.Where(`status @> '{"host_state": "` + strings.ToUpper(hsFilter.HostStatus) + `"}'`)
	}

	// apply result limits
	tx = tx.Limit(hsFilter.Limit)

	return tx
}

func auditlogEntryToHostStatus(auRecord models.AuditLogEntry) (*hvs.HostStatus, error) {
	defaultLog.Trace("postgres/report_store:auditlogEntryToHostStatus() Entering")
	defer defaultLog.Trace("postgres/report_store:auditlogEntryToHostStatus() Leaving")

	var hostStatus hvs.HostStatus
	var err error

	if auRecord.EntityID != uuid.Nil {
		hostStatus.ID = auRecord.EntityID
	}

	// TODO remove duplicate data: first column and the entityID are both same
	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[1]) && auRecord.Data.Columns[1].Value != nil {
		hostStatus.HostID = uuid.MustParse(fmt.Sprintf("%v", auRecord.Data.Columns[1].Value))
	}

	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[2]) && auRecord.Data.Columns[2].Value != nil {
		createdString := fmt.Sprintf("%v", auRecord.Data.Columns[2].Value)
		hostStatus.Created, err = time.Parse(constants.ParamDateFormatUTC, createdString)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/reports_store:auditlogEntryToHostStatus() - error parsing Created timestamp")
		}
	}

	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[3]) && auRecord.Data.Columns[3].Value != nil {
		c, err := json.Marshal(auRecord.Data.Columns[3].Value)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/hoststatus_store:auditlogEntryToHostStatus() - marshalling HostStatusInformation failed")
		}
		err = json.Unmarshal(c, &hostStatus.HostStatusInformation)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/reports_store:auditlogEntryToHostStatus() - unmarshalling HostStatusInformation failed")
		}
	}

	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[4]) && auRecord.Data.Columns[4].Value != nil {
		c, err := json.Marshal(auRecord.Data.Columns[4].Value)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/hoststatus_store:auditlogEntryToHostStatus() - marshalling HostManifest failed")
		}
		err = json.Unmarshal(c, &hostStatus.HostManifest)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/reports_store:auditlogEntryToHostStatus() - unmarshalling HostManifest failed")
		}
	}

	return &hostStatus, nil
}
