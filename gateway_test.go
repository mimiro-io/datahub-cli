package datahub_mim_cli

import (
	"testing"

	"github.com/mimiro-io/datahub-cli/internal/gateway"
)

func TestGateway(t *testing.T) {
	gateway.StartGateway("local-admin", "8242")
}
