package resource

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/venikkin/neo4j-aura-terraform-provider/internal/client"
	"github.com/venikkin/neo4j-aura-terraform-provider/internal/util"
	"strings"
	"time"
)

// Ensure resource defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &InstanceResource{}
	_ resource.ResourceWithConfigure = &InstanceResource{}
)

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

type InstanceResource struct {
	auraApi *client.AuraApi
}

type InstanceResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Region        types.String `tfsdk:"region"`
	Memory        types.String `tfsdk:"memory"`
	Type          types.String `tfsdk:"type"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	TenantId      types.String `tfsdk:"tenant_id"`
	ConnectionUrl types.String `tfsdk:"connection_url"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	Version       types.String `tfsdk:"version"`
	Paused        types.Bool   `tfsdk:"paused"`
}

func (r *InstanceResource) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	auraClient, ok := request.ProviderData.(*client.AuraClient)

	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.AuraClient, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)
		return
	}
	r.auraApi = client.NewAuraApi(auraClient)
}

func (r *InstanceResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	// todo markdown descriptions and rest of metadata
	response.Schema = schema.Schema{
		MarkdownDescription: "Aura instance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Id of the instance",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name if the instance",
				Required:            true,
			},
			"region": schema.StringAttribute{
				Required: true,
			},
			"memory": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("1GB"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("free-db"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cloud_provider": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("gcp"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tenant_id": schema.StringAttribute{
				Required: true,
			},
			"connection_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("5"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"paused": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *InstanceResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	postInstanceRequest := client.PostInstanceRequest{
		Version:       data.Version.ValueString(),
		Region:        data.Region.ValueString(),
		Memory:        data.Memory.ValueString(),
		Name:          data.Name.ValueString(),
		Type:          data.Type.ValueString(),
		TenantId:      data.TenantId.ValueString(),
		CloudProvider: data.CloudProvider.ValueString(),
	}

	postInstanceResp, err := r.auraApi.PostInstance(postInstanceRequest)
	if err != nil {
		response.Diagnostics.AddError("Error while creating an instance", err.Error())
		return
	}

	data.Id = types.StringValue(postInstanceResp.Data.Id)
	data.ConnectionUrl = types.StringValue(postInstanceResp.Data.ConnectionUrl)
	data.Username = types.StringValue(postInstanceResp.Data.Username)
	data.Password = types.StringValue(postInstanceResp.Data.Password)

	tflog.Debug(ctx, "Created an instance with id "+postInstanceResp.Data.Id)

	_, err = r.WaitUntilInstanceIsInState(ctx, postInstanceResp.Data.Id, func(r client.GetInstanceResponse) bool {
		return strings.ToLower(r.Data.Status) == "running"
	})
	if err != nil {
		response.Diagnostics.AddError("Instance is not running in time", err.Error())
	}

	tflog.Debug(ctx, fmt.Sprintf("Instance %s is running", postInstanceResp.Data.Id))

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *InstanceResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var plan InstanceResourceModel

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)

	if response.Diagnostics.HasError() {
		return
	}

	instance, err := r.auraApi.GetInstanceById(plan.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error while getting instance details", err.Error())
	}
	// Regular inplace update
	if plan.Name.ValueString() != instance.Data.Name || plan.Memory.ValueString() != instance.Data.Memory {
		_, err := r.auraApi.PatchInstanceById(instance.Data.Id, client.PatchInstanceRequest{
			Name:   plan.Name.ValueStringPointer(),
			Memory: plan.Memory.ValueStringPointer(),
		})

		if err != nil {
			response.Diagnostics.AddError("Error while updating the instance details", err.Error())
			return
		}

		_, err = r.WaitUntilInstanceIsInState(ctx, plan.Id.ValueString(), func(resp client.GetInstanceResponse) bool {
			return resp.Data.Memory == plan.Memory.ValueString() &&
				resp.Data.Name == plan.Name.ValueString() &&
				(strings.ToLower(resp.Data.Status) == "running" || strings.ToLower(instance.Data.Status) == "paused")
		})
	}

	// Pause
	if plan.Paused.ValueBool() && strings.ToLower(instance.Data.Status) != "paused" {
		_, err := r.auraApi.PauseInstanceById(instance.Data.Id)
		if err != nil {
			response.Diagnostics.AddError("Error while pausing the instance", err.Error())
			return
		}
		_, err = r.WaitUntilInstanceIsInState(ctx, plan.Id.ValueString(), func(resp client.GetInstanceResponse) bool {
			return strings.ToLower(instance.Data.Status) == "paused"
		})

		// Resume
	} else if !plan.Paused.ValueBool() && strings.ToLower(instance.Data.Status) == "paused" {
		_, err := r.auraApi.ResumeInstanceById(instance.Data.Id)
		if err != nil {
			response.Diagnostics.AddError("Error while resume the instance", err.Error())
			return
		}
		_, err = r.WaitUntilInstanceIsInState(ctx, plan.Id.ValueString(), func(resp client.GetInstanceResponse) bool {
			return strings.ToLower(instance.Data.Status) == "running"
		})
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *InstanceResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data InstanceResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	_, err := r.auraApi.DeleteInstanceById(data.Id.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Error while deleting an instance", err.Error())
	}
}

func (r *InstanceResource) WaitUntilInstanceIsInState(
	ctx context.Context,
	id string,
	condition func(client.GetInstanceResponse) bool) (client.GetInstanceResponse, error) {
	return util.WaitUntil(
		func() (client.GetInstanceResponse, error) {
			r, e := r.auraApi.GetInstanceById(id)
			tflog.Debug(ctx, fmt.Sprintf("Received response %+v and error %+v", r, e))
			return r, e
		},
		func(resp client.GetInstanceResponse, e error) bool {
			return e == nil && condition(resp)
		},
		time.Second,
		time.Minute*time.Duration(7),
	)
}
