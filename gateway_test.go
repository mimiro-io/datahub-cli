package datahub_mim_cli

import (
	"fmt"
	"testing"

	"github.com/mimiro-io/datahub-cli/internal/gateway"
)

// This test is not meant to be run as part of automated testing
// It is to support running the gateway for manual testing
func TestGateway(t *testing.T) {
	c := gateway.Catalog{}
	fmt.Println(c)
	// 	gateway.StartGateway("local", "8242")
}
