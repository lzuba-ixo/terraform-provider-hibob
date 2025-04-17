package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &employeesDataSource{}
	_ datasource.DataSourceWithConfigure = &employeesDataSource{}
)

func NewEmployeesDataSource() datasource.DataSource {
	return &employeesDataSource{}
}

// employeesDataSource is the data source implementation.
type employeesDataSource struct {
	authorizationHeader string
}

// Configure adds the provider configured client to the data source.
func (d *employeesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Add a nil check when handling ProviderData because Terraform
	// sets that data after it calls the ConfigureProvider RPC.
	if req.ProviderData == nil {
		return
	}

	authorizationHeader := req.ProviderData
	if authorizationHeader == "" {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected authorization header, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.authorizationHeader = authorizationHeader.(string)
}

// Metadata returns the data source type name.
func (d *employeesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_employees"
}

// Schema defines the schema for the data source.
func (d *employeesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the list of employees.",
		Attributes: map[string]schema.Attribute{
			"employees": schema.ListNestedAttribute{
				Description: "List of employees.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Employees ID within hibob",
							Computed:    true,
						},
						"company_id": schema.StringAttribute{
							Description: "Id of the company assigned to the employee",
							Computed:    true,
						},
						"creation_datetime": schema.StringAttribute{
							Description: "Timestamp of inital creation of employee",
							Computed:    true,
						},
						"display_name": schema.StringAttribute{
							Description: "Name displayed for employee used for searches, etc",
							Computed:    true,
						},
						"email": schema.StringAttribute{
							Description: "Company email address",
							Computed:    true,
						},
						"first_name": schema.StringAttribute{
							Description: "First name of employee",
							Computed:    true,
						},
						"full_name": schema.StringAttribute{
							Description: "Full name of employee",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Status of employee",
							Computed:    true,
						},
						"lifecycle_status": schema.StringAttribute{
							Description: "Lifecycle stage of employee",
							Computed:    true,
						},
						"second_name": schema.StringAttribute{
							Description: "User chosen name (nickname)",
							Computed:    true,
						},
						"surname": schema.StringAttribute{
							Description: "Surname of employee",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// employeesDataSourceModel maps the data source schema data.
type employeesDataSourceModel struct {
	Employees []employeesModel `tfsdk:"employees"`
}

// employeesModel maps employee schema data.
type employeesModel struct {
	ID               types.String `tfsdk:"id"`
	CompanyId        types.String `tfsdk:"company_id"`
	CreationDateTime types.String `tfsdk:"creation_datetime" `
	DisplayName      types.String `tfsdk:"display_name"`
	Email            types.String `tfsdk:"email"`
	FirstName        types.String `tfsdk:"first_name"`
	FullName         types.String `tfsdk:"full_name"`
	SecondName       types.String `tfsdk:"second_name"`
	Surname          types.String `tfsdk:"surname"`
	Status           types.String `tfsdk:"status"`
	LifecycleStatus  types.String `tfsdk:"lifecycle_status"`
}

// Read refreshes the Terraform state with the latest data.
func (d *employeesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state employeesDataSourceModel

	request, _ := http.NewRequest("GET", "https://api.hibob.com/v1/people", nil)

	request.Header.Add("accept", "application/json")
	request.Header.Add("Authorization", d.authorizationHeader)

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to read hibob employee api",
			err.Error(),
		)
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	if string(body) == "Too many attempts" {
		resp.Diagnostics.AddError(
			"Encountered rate limit from hibob api, try again later",
			"",
		)
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		resp.Diagnostics.AddError(
			"Unable to read hibob employee api reponse",
			err.Error(),
		)
		return
	}

	// Map response body to model
	for _, item := range result["employees"].([]interface{}) {
		employee := item.(map[string]interface{})
		internal := employee["internal"].(map[string]interface{})
		newEmployee := employeesModel{
			ID:               types.StringValue(employee["id"].(string)),
			CompanyId:        types.StringValue(strconv.FormatFloat(employee["companyId"].(float64), 'f', -1, 64)),
			CreationDateTime: types.StringValue(If(employee["creationDateTime"])),
			DisplayName:      types.StringValue(If(employee["displayName"])),
			Email:            types.StringValue(If(employee["email"])),
			FirstName:        types.StringValue(If(employee["firstName"])),
			FullName:         types.StringValue(If(employee["fullName"])),
			SecondName:       types.StringValue(If(employee["secondName"])),
			Surname:          types.StringValue(If(employee["surname"])),
			Status:           types.StringValue(If(internal["status"])),
			LifecycleStatus:  types.StringValue(If(internal["lifecycleStatus"])),
		}
		state.Employees = append(state.Employees, newEmployee)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func If(item interface{}) string {
	if item == nil {
		return ""
	} else {
		return item.(string)
	}
}
