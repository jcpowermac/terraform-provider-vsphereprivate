package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/jcpowermac/terraform-provider-vsphereprivate/vsphereprivate"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vsphereprivate.Provider})
}
