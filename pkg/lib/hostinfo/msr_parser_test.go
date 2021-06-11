/*
 * Copyright (C) 2021 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */
package hostinfo

import (
	"reflect"
	"testing"

	model "github.com/intel-secl/intel-secl/v4/pkg/model/ta"
	"github.com/stretchr/testify/mock"
)

func testMsrInfoParser(t *testing.T, mockMsrReader msrReader, expectedResults *model.HostInfo, processorFlags string) {
	hostInfo := model.HostInfo{}
	hostInfo.ProcessorFlags = processorFlags // TXT looks for "SMX" in the process flags for "supported" flag

	msrInfoParser := msrInfoParser{
		msrReader: mockMsrReader,
	}

	msrInfoParser.parseTxt(&hostInfo)
	msrInfoParser.parseCbnt(&hostInfo)

	if !reflect.DeepEqual(hostInfo.HardwareFeatures, expectedResults.HardwareFeatures) {
		t.Errorf("The parsed MRS data does not match the expected results.\nExpected: %+v\nActual: %+v\n", expectedResults.HardwareFeatures, hostInfo.HardwareFeatures)
	}
}

func TestMsrPositive(t *testing.T) {

	// return values from a system with TXT enabled (i.e., from MSR offset 0x51) and
	// BTG Profile 5 (at 0x13A).  This data can be viewed in bash using 'xxd'...
	//
	// >> sudo hexdump -C -s 0x3A -n 8 /dev/cpu/0/msr
	// 0000003a  05 00 10 00 00 00 00 00                           |........|
	// 00000042
	//
	// >> sudo hexdump -C -s 0x13A -n 8 /dev/cpu/0/msr
	// 0000013a  7d 00 00 00 0f 00 00 00                           |}.......|
	// 00000142

	mockMsrReader := new(mockMsrReader)
	mockMsrReader.On("ReadAt", int64(txtMsrOffset)).Return(0x10ff07, nil)
	mockMsrReader.On("ReadAt", int64(cbntMsrOffset)).Return(0xf0000007d, nil)

	hostInfo := model.HostInfo{}
	hostInfo.HardwareFeatures.TXT.Supported = true
	hostInfo.HardwareFeatures.TXT.Enabled = true
	hostInfo.HardwareFeatures.CBNT.Supported = true
	hostInfo.HardwareFeatures.CBNT.Enabled = true
	hostInfo.HardwareFeatures.CBNT.Meta.Profile = cbntProfile5
	hostInfo.HardwareFeatures.CBNT.Meta.MSR = cbntMsrFlags

	testMsrInfoParser(t, mockMsrReader, &hostInfo, "SMX")
}

func TestMsrNegative(t *testing.T) {

	// return msr data where TXT and CBNT are disabled
	mockMsrReader := new(mockMsrReader)
	mockMsrReader.On("ReadAt", int64(txtMsrOffset)).Return(0x100005, nil)
	mockMsrReader.On("ReadAt", int64(cbntMsrOffset)).Return(0x400000000, nil)

	hostInfo := model.HostInfo{}
	hostInfo.HardwareFeatures.TXT.Supported = true
	hostInfo.HardwareFeatures.TXT.Enabled = false
	hostInfo.HardwareFeatures.CBNT.Supported = false
	hostInfo.HardwareFeatures.CBNT.Enabled = false
	hostInfo.HardwareFeatures.CBNT.Meta.Profile = ""
	hostInfo.HardwareFeatures.CBNT.Meta.MSR = ""

	testMsrInfoParser(t, mockMsrReader, &hostInfo, "")
}

//-------------------------------------------------------------------------------------------------
// Mock implementation of msrReader to support unit testing
//-------------------------------------------------------------------------------------------------
type mockMsrReader struct {
	mock.Mock
}

func (mockMsrReader mockMsrReader) ReadAt(offset int64) (uint64, error) {
	args := mockMsrReader.Called(offset)
	return uint64(args.Int(0)), args.Error(1)
}