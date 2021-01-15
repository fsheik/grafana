package api

import (
	"github.com/grafana/grafana/pkg/api/utils"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/setting"
)

func GetOrgQuotas(c *models.ReqContext) utils.Response {
	if !setting.Quota.Enabled {
		return utils.Error(404, "Quotas not enabled", nil)
	}
	query := models.GetOrgQuotasQuery{OrgId: c.ParamsInt64(":orgId")}

	if err := bus.Dispatch(&query); err != nil {
		return utils.Error(500, "Failed to get org quotas", err)
	}

	return utils.JSON(200, query.Result)
}

func UpdateOrgQuota(c *models.ReqContext, cmd models.UpdateOrgQuotaCmd) utils.Response {
	if !setting.Quota.Enabled {
		return utils.Error(404, "Quotas not enabled", nil)
	}
	cmd.OrgId = c.ParamsInt64(":orgId")
	cmd.Target = c.Params(":target")

	if _, ok := setting.Quota.Org.ToMap()[cmd.Target]; !ok {
		return utils.Error(404, "Invalid quota target", nil)
	}

	if err := bus.Dispatch(&cmd); err != nil {
		return utils.Error(500, "Failed to update org quotas", err)
	}
	return utils.Success("Organization quota updated")
}

func GetUserQuotas(c *models.ReqContext) utils.Response {
	if !setting.Quota.Enabled {
		return utils.Error(404, "Quotas not enabled", nil)
	}
	query := models.GetUserQuotasQuery{UserId: c.ParamsInt64(":id")}

	if err := bus.Dispatch(&query); err != nil {
		return utils.Error(500, "Failed to get org quotas", err)
	}

	return utils.JSON(200, query.Result)
}

func UpdateUserQuota(c *models.ReqContext, cmd models.UpdateUserQuotaCmd) utils.Response {
	if !setting.Quota.Enabled {
		return utils.Error(404, "Quotas not enabled", nil)
	}
	cmd.UserId = c.ParamsInt64(":id")
	cmd.Target = c.Params(":target")

	if _, ok := setting.Quota.User.ToMap()[cmd.Target]; !ok {
		return utils.Error(404, "Invalid quota target", nil)
	}

	if err := bus.Dispatch(&cmd); err != nil {
		return utils.Error(500, "Failed to update org quotas", err)
	}
	return utils.Success("Organization quota updated")
}
