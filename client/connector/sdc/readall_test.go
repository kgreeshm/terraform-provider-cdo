package sdc

import (
	"context"
	"reflect"
	"testing"

	"github.com/cisco-lockhart/go-client/internal/http"
	"github.com/jarcoal/httpmock"
)

func TestReadAll(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	validCdg := NewSdcResponseBuilder().
		AsDefaultCdg().
		WithUid(cdgUid).
		WithName(cdgName).
		WithTenantUid(tenantUid).
		Build()

	validSdc := NewSdcResponseBuilder().
		AsSdc().
		WithUid(sdcUid).
		WithName(sdcName).
		WithTenantUid(tenantUid).
		Build()

	testCases := []struct {
		testName   string
		setupFunc  func()
		assertFunc func(output *ReadAllOutput, err error, t *testing.T)
	}{
		{
			testName: "successfully fetches secure connectors",

			setupFunc: func() {
				httpmock.RegisterResponder(
					"GET",
					"/aegis/rest/v1/services/targets/proxies",
					httpmock.NewJsonResponderOrPanic(200, ReadAllOutput{validCdg, validSdc}),
				)
			},

			assertFunc: func(output *ReadAllOutput, err error, t *testing.T) {
				if err != nil {
					t.Errorf("unexpected error: %s", err.Error())
				}

				if output == nil {
					t.Fatal("output is nil!")
				}

				expectedResponse := ReadAllOutput{validCdg, validSdc}
				if !reflect.DeepEqual(expectedResponse, *output) {
					t.Errorf("expected: %+v\ngot: %+v", expectedResponse, output)
				}
			},
		},
		{
			testName: "returns empty slice no connectors have been onboarded",

			setupFunc: func() {
				httpmock.RegisterResponder(
					"GET",
					"/aegis/rest/v1/services/targets/proxies",
					httpmock.NewJsonResponderOrPanic(200, ReadAllOutput{}),
				)
			},

			assertFunc: func(output *ReadAllOutput, err error, t *testing.T) {
				if err != nil {
					t.Errorf("expected err to be nil, got: %s", err.Error())
				}

				if output == nil {
					t.Fatal("returned slice was nil!")
				}

				if len(*output) != 0 {
					t.Errorf("expected empty slice, got: %+v", *output)
				}
			},
		},
		{
			testName: "return error when fetching all secure connectors and remote service encounters issue",

			setupFunc: func() {
				httpmock.RegisterResponder(
					"GET",
					"/aegis/rest/v1/services/targets/proxies",
					httpmock.NewStringResponder(500, "service is experiencing issues"),
				)
			},

			assertFunc: func(output *ReadAllOutput, err error, t *testing.T) {
				if output != nil {
					t.Errorf("expected output to be nil, got (dereferenced): %+v", *output)
				}

				if err != nil {
					t.Errorf("expected err to be nil, got: %s", err.Error())
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			httpmock.Reset()

			testCase.setupFunc()

			output, err := ReadAll(context.Background(), *http.NewWithDefault("https://unittest.cdo.cisco.com", "a_valid_token"), *NewReadAllInput())

			testCase.assertFunc(output, err, t)
		})
	}
}
