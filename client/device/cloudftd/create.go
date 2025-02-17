package cloudftd

/**
* The cloud FTD corresponds to the CDO Device Type FTDC, which is an FTD managed by a cdFMC.
 */
import (
	"context"
	"fmt"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/device"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/device/cloudfmc"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/device/cloudfmc/fmcplatform"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/internal/cdo"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/internal/http"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/internal/retry"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/internal/url"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/model/device/tags"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/model/devicetype"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/model/ftd/license"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/model/ftd/tier"
)

type CreateInput struct {
	Name             string
	AccessPolicyName string
	PerformanceTier  *tier.Type // ignored if it is physical device
	Virtual          bool
	Licenses         *[]license.Type
	Tags             tags.Type
}

type CreateOutput struct {
	Uid      string    `json:"uid"`
	Name     string    `json:"name"`
	Metadata *Metadata `json:"metadata"`
	Tags     tags.Type `json:"tags"`
}

func NewCreateInput(
	name string,
	accessPolicyName string,
	performanceTier *tier.Type,
	virtual bool,
	licenses *[]license.Type,
	tags tags.Type,
) CreateInput {
	return CreateInput{
		Name:             name,
		AccessPolicyName: accessPolicyName,
		PerformanceTier:  performanceTier,
		Virtual:          virtual,
		Licenses:         licenses,
		Tags:             tags,
	}
}

type createRequestBody struct {
	FmcId      string          `json:"associatedDeviceUid"`
	DeviceType devicetype.Type `json:"deviceType"`
	Metadata   *Metadata       `json:"metadata"`
	Name       string          `json:"name"`
	State      string          `json:"state"` // TODO: use queueTriggerState?
	Type       string          `json:"type"`
	Model      bool            `json:"model"`
	Tags       tags.Type       `json:"tags"`
}

func Create(ctx context.Context, client http.Client, createInp CreateInput) (*CreateOutput, error) {

	client.Logger.Println("creating cloud ftd")

	// 1. read Cloud FMC
	fmcRes, err := cloudfmc.Read(ctx, client, cloudfmc.NewReadInput())
	if err != nil {
		return nil, err
	}

	// 2. get FMC domain uid by reading Cloud FMC domain info
	readFmcDomainRes, err := fmcplatform.ReadFmcDomainInfo(ctx, client, fmcplatform.NewReadDomainInfoInput(fmcRes.Host))
	if err != nil {
		return nil, err
	}
	if len(readFmcDomainRes.Items) == 0 {
		return nil, fmt.Errorf("fmc domain info not found")
	}

	// 3. get access policies using Cloud FMC domain uid
	accessPoliciesRes, err := cloudfmc.ReadAccessPolicies(
		ctx,
		client,
		cloudfmc.NewReadAccessPoliciesInput(fmcRes.Host, readFmcDomainRes.Items[0].Uuid, 1000), // 1000 is what CDO UI uses
	)
	if err != nil {
		return nil, err
	}
	selectedPolicy, ok := accessPoliciesRes.Find(createInp.AccessPolicyName)
	if !ok {
		return nil, fmt.Errorf(
			`access policy: "%s" not found, available policies: %s. In rare cases where you have more than 1000 access policies, please raise an issue at: %s`,
			createInp.AccessPolicyName,
			accessPoliciesRes.Items,
			cdo.TerraformProviderCDOIssuesUrl,
		)
	}

	// handle performance tier
	var performanceTier *tier.Type = nil // physical is nil
	if createInp.Virtual {
		performanceTier = createInp.PerformanceTier
	}

	client.Logger.Println("posting FTD device")

	// 4. create the cloud ftd device
	createUrl := url.CreateDevice(client.BaseUrl())
	createBody := createRequestBody{
		Name:       createInp.Name,
		FmcId:      fmcRes.Uid,
		DeviceType: devicetype.CloudFtd,
		Metadata: &Metadata{
			AccessPolicyName: selectedPolicy.Name,
			AccessPolicyUid:  selectedPolicy.Id,
			LicenseCaps:      license.SerializeAllAsCdo(*createInp.Licenses),
			PerformanceTier:  performanceTier,
		},
		State: "NEW",
		Type:  "devices",
		Model: false,
		Tags:  createInp.Tags,
	}
	createReq := client.NewPost(ctx, createUrl, createBody)
	var createOup CreateOutput
	if err := createReq.Send(&createOup); err != nil {
		return nil, err
	}

	client.Logger.Println("reading FTD specific device")

	// 5. read created cloud ftd's specific device's uid
	readSpecRes, err := device.ReadSpecific(ctx, client, *device.NewReadSpecificInput(createOup.Uid))
	if err != nil {
		return nil, err
	}

	// 6. initiate cloud ftd onboarding by triggering an endpoint at the specific device
	_, err = UpdateSpecific(ctx, client,
		NewUpdateSpecificFtdInput(
			readSpecRes.SpecificUid,
			"INITIATE_FTDC_ONBOARDING",
		),
	)
	if err != nil {
		return nil, err
	}

	// 8. wait for generate command available
	var metadata Metadata
	err = retry.Do(
		ctx,
		UntilGeneratedCommandAvailable(ctx, client, createOup.Uid, &metadata),
		retry.NewOptionsBuilder().
			Message("Waiting for FTD record to be created in CDO...").
			Retries(3).
			Timeout(retry.DefaultTimeout).
			Delay(retry.DefaultDelay).
			Logger(client.Logger).
			EarlyExitOnError(true).
			Build(),
	)
	if err != nil {
		return nil, err
	}

	// done!
	return &CreateOutput{
		Uid:      createOup.Uid,
		Name:     createOup.Name,
		Metadata: &metadata,
		Tags:     createOup.Tags,
	}, nil
}
