package netbox

import (
	"errors"
	"fmt"
	"github.com/fbreckle/go-netbox/netbox/client"
	"github.com/fbreckle/go-netbox/netbox/client/dcim"
	"github.com/fbreckle/go-netbox/netbox/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/id"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
)

func dataSourceNetboxDeviceFrontPorts() *schema.Resource {
	return &schema.Resource{
		Read:        dataSourceNetboxDeviceFrontPortsRead,
		Description: `:meta:subcategory:Data Center Inventory Management (DCIM):`,
		Schema: map[string]*schema.Schema{
			"filter": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"name_regex": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsValidRegExp,
			},
			"front_ports": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"occupied": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"tag_ids": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeInt,
							},
						},
						"device_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceNetboxDeviceFrontPortsRead(d *schema.ResourceData, m interface{}) error {
	api := m.(*client.NetBoxAPI)

	params := dcim.NewDcimFrontPortsListParams()

	if filter, ok := d.GetOk("filter"); ok {
		var filterParams = filter.(*schema.Set)
		for _, f := range filterParams.List() {
			k := f.(map[string]interface{})["name"]
			v := f.(map[string]interface{})["value"]
			vString := v.(string)
			switch k {
			case "device_id":
				params.DeviceID = &vString
			case "device_name":
				params.Device = &vString
			case "name":
				params.Name = &vString
			default:
				return fmt.Errorf("'%s' is not a supported filter parameter", k)
			}
		}
	}

	res, err := api.Dcim.DcimFrontPortsList(params, nil)
	if err != nil {
		return err
	}

	if *res.GetPayload().Count == int64(0) {
		return errors.New("no result")
	}

	var filteredFrontPorts []*models.FrontPort
	if nameRegex, ok := d.GetOk("name_regex"); ok {
		r := regexp.MustCompile(nameRegex.(string))
		for _, frontPort := range res.GetPayload().Results {
			if r.MatchString(*frontPort.Name) {
				filteredFrontPorts = append(filteredFrontPorts, frontPort)
			}
		}
	} else {
		filteredFrontPorts = res.GetPayload().Results
	}

	var s []map[string]interface{}
	for _, frontPort := range filteredFrontPorts {
		var mapping = make(map[string]interface{})
		mapping["id"] = frontPort.ID
		if frontPort.Description != "" {
			mapping["description"] = frontPort.Description
		}
		mapping["occupied"] = frontPort.Occupied
		if frontPort.Name != nil {
			mapping["name"] = *frontPort.Name
		}
		if frontPort.Tags != nil {
			var tags []int64
			for _, t := range frontPort.Tags {
				tags = append(tags, t.ID)
			}
			mapping["tag_ids"] = tags
		}

		mapping["device_id"] = frontPort.Device.ID
		mapping["label"] = frontPort.Label

		s = append(s, mapping)
	}

	d.SetId(id.UniqueId())
	return d.Set("front_ports", s)
}
