package sdc

import (
	"crypto/rand"
	"crypto/rsa"

	internalRsa "github.com/cisco-lockhart/go-client/internal/crypto/rsa"
)

const (
	cdgUid    = "11111111-1111-1111-1111-111111111111"
	cdgName   = "Cloud Connector"
	sdcUid    = "22222222-2222-2222-2222-222222222222"
	sdcName   = "My On Prem SDC"
	tenantUid = "99999999-9999-9999-9999-999999999999"
)

type sdcResponseBuilder struct {
	readOutput ReadOutput
}

func NewSdcResponseBuilder() *sdcResponseBuilder {
	return &sdcResponseBuilder{}
}

func (builder *sdcResponseBuilder) Build() ReadOutput {
	return builder.readOutput
}

func (builder *sdcResponseBuilder) AsDefaultCloudConnector() *sdcResponseBuilder {
	builder.readOutput.DefaultLar = true
	builder.readOutput.Cdg = true

	return builder
}

func (builder *sdcResponseBuilder) AsOnPremConnector() *sdcResponseBuilder {
	builder.readOutput.Cdg = false
	builder.readOutput.PublicKey = PublicKey{
		EncodedKey: mustGenerateBase64PublicKey(),
		Version:    164,
		KeyId:      "01010101-0101-0101-0101-010101010101",
	}

	return builder
}

func (builder *sdcResponseBuilder) WithUid(uid string) *sdcResponseBuilder {
	builder.readOutput.Uid = uid

	return builder
}

func (builder *sdcResponseBuilder) WithName(name string) *sdcResponseBuilder {
	builder.readOutput.Name = name

	return builder
}

func (builder *sdcResponseBuilder) WithTenantUid(tenantUid string) *sdcResponseBuilder {
	builder.readOutput.TenantUid = tenantUid

	return builder
}

func mustGenerateBase64PublicKey() string {
	key, err := rsa.GenerateKey(rand.Reader, 512)
	if err != nil {
		panic(err)
	}

	return internalRsa.MustBase64PublicKeyFromRsaKey(key)
}
