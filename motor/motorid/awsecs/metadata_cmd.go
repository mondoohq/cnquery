package awsecsid

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"errors"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/motorid/containerid"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/os"
)

const (
	identityUrl = "${ECS_CONTAINER_METADATA_URI_V4}"
)

func NewContainerMetadata(provider os.OperatingSystemProvider, pf *platform.Platform) *ContainerMetadata {
	return &ContainerMetadata{
		provider: provider,
		platform: pf,
	}
}

type ContainerMetadata struct {
	provider os.OperatingSystemProvider
	platform *platform.Platform
}

func (m *ContainerMetadata) Identify() (Identity, error) {
	log.Debug().Msg("getting ecs container identity")

	containerDocument, err := m.containerIdentityDocument()
	if err != nil {
		return Identity{}, err
	}
	// parse into struct
	doc := EcrContainerIdentityDoc{}
	if err := json.NewDecoder(strings.NewReader(containerDocument)).Decode(&doc); err != nil {
		return Identity{}, errors.Join(err, errors.New("failed to decode ECS container identity document"))
	}
	var accountID string
	if arn.IsARN(doc.ContainerArn) {
		if p, err := arn.Parse(doc.ContainerArn); err == nil {
			accountID = p.AccountID
		}
	}
	return Identity{
		Name:              doc.Name,
		ContainerArn:      doc.ContainerArn,
		RuntimeID:         doc.DockerId,
		AccountPlatformID: "//platformid.api.mondoo.app/runtime/aws/accounts/" + accountID,
		PlatformIds:       []string{MondooECSContainerID(doc.ContainerArn), containerid.MondooContainerID(doc.DockerId)},
	}, nil
}

func (m *ContainerMetadata) curlDocument(url string) (string, error) {
	cmd, err := m.provider.RunCommand("curl " + url)
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func (m *ContainerMetadata) containerIdentityDocument() (string, error) {
	return m.curlDocument(identityUrl)
}

type EcrContainerIdentityDoc struct {
	DockerId     string `json:"DockerId"`
	Name         string `json:"Name"`
	ContainerArn string `json:"ContainerARN"`
}
