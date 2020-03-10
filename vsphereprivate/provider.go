package vsphereprivate

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/terraform-providers/terraform-provider-vsphere/vsphere"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	vsphereProvider := vsphere.Provider()

	vsphereProvider.(*schema.Provider).ResourcesMap = map[string]*schema.Resource{
		"vsphere_import_ova": resourceVSphereImportOva(),
	}

	return vsphereProvider
}
