package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/plugin"
	"github.com/jcpowermac/terraform-provider-vsphereprivate/vsphereprivate"
	//"github.com/terraform-providers/terraform-provider-vsphere/vsphere"
)

func main() {

	//f := vsphere.Provider



	


	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: vsphereprivate.Provider})
}
