package vsphereprivate

import (
	_ "context"
	_ "errors"
	_ "fmt"
	_ "path"
	_ "strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"

	_ "github.com/vmware/govmomi/object"
	_ "github.com/vmware/govmomi/vim25/types"
)

func resourceVSphereImportOva() *schema.Resource {
	return &schema.Resource{
		Create: resourceVSphereImportOvaCreate,
		Read:   resourceVSphereImportOvaRead,
		Update: resourceVSphereImportOvaUpdate,
		Delete: resourceVSphereImportOvaDelete,
		Importer: &schema.ResourceImporter{
			State: resourceVSphereImportOvaImport,
		},

		SchemaVersion: 1,
		//MigrateState:  resourceVSphereFolderMigrateState,
		/*
		   type ImportOvaParams struct {
		   	ResourcePool *object.ResourcePool
		   	Datacenter   *object.Datacenter
		   	Datastore    *object.Datastore
		   	Network      *object.Network
		   	Host         *object.HostSystem
		   	Folder       *object.Folder
		   }
		*/

		Schema: map[string]*schema.Schema{
			"path": {
				Type:         schema.TypeString,
				Description:  "",
				Required:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"datacenter_id": {
				Type:        schema.TypeString,
				Description: "The ID of the datacenter. Can be ignored if creating a datacenter folder, otherwise required.",
				ForceNew:    true,
				Optional:    false,
			},
			"resource_pool_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of a resource pool to put the virtual machine in.",
			},
			"network_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of a network that the virtual machine will use.",
			},
			"datastore_id": {
				Type:        schema.TypeString,
				Required:    true,
				Computed:    true,
				Description: "The ID of the virtual machine's datastore. The virtual machine configuration is placed here, along with any virtual disks that are created without datastores.",
			},
			"folder": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the folder to locate the virtual machine in.",
			},

			// Tagging
			//vSphereTagAttributeKey: tagsSchema(),
			// Custom Attributes
			//customattribute.ConfigKey: customattribute.ConfigSchema(),
		},
	}
}

func resourceVSphereImportOvaCreate(d *schema.ResourceData, meta interface{}) error {
	foo := FileArchive{}
	spew.Dump(foo)

	return nil
}

func resourceVSphereImportOvaRead(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceVSphereImportOvaUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVSphereImportOvaDelete(d *schema.ResourceData, meta interface{}) error {

	return nil
}

func resourceVSphereImportOvaImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	return nil, nil
}
