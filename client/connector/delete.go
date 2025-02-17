package connector

import (
	"context"

	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/internal/http"
	"github.com/CiscoDevnet/terraform-provider-cdo/go-client/internal/url"
)

type DeleteInput struct {
	Uid string `json:"-"`
}

func NewDeleteInput(uid string) DeleteInput {
	return DeleteInput{
		Uid: uid,
	}
}

type DeleteOutput struct {
}

func Delete(ctx context.Context, client http.Client, inp DeleteInput) (*DeleteOutput, error) {

	url := url.DeleteConnector(client.BaseUrl(), inp.Uid)

	req := client.NewDelete(ctx, url)

	var deleteOutp DeleteOutput
	if err := req.Send(&deleteOutp); err != nil {
		return &DeleteOutput{}, err
	}

	return &deleteOutp, nil
}
