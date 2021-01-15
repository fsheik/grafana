package ngalert

import (
	"github.com/go-macaron/binding"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/api/utils"
	"github.com/grafana/grafana/pkg/middleware"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/ngalert/eval"
	"github.com/grafana/grafana/pkg/tsdb"
	"github.com/grafana/grafana/pkg/util"
)

func (ng *AlertNG) registerAPIEndpoints() {
	ng.RouteRegister.Group("/api/alert-definitions", func(alertDefinitions routing.RouteRegister) {
		alertDefinitions.Get("", middleware.ReqSignedIn, utils.Wrap(ng.listAlertDefinitions))
		alertDefinitions.Get("/eval/:alertDefinitionUID", ng.validateOrgAlertDefinition, utils.Wrap(ng.alertDefinitionEvalEndpoint))
		alertDefinitions.Post("/eval", middleware.ReqSignedIn, binding.Bind(evalAlertConditionCommand{}), utils.Wrap(ng.conditionEvalEndpoint))
		alertDefinitions.Get("/:alertDefinitionUID", ng.validateOrgAlertDefinition, utils.Wrap(ng.getAlertDefinitionEndpoint))
		alertDefinitions.Delete("/:alertDefinitionUID", ng.validateOrgAlertDefinition, utils.Wrap(ng.deleteAlertDefinitionEndpoint))
		alertDefinitions.Post("/", middleware.ReqSignedIn, binding.Bind(saveAlertDefinitionCommand{}), utils.Wrap(ng.createAlertDefinitionEndpoint))
		alertDefinitions.Put("/:alertDefinitionUID", ng.validateOrgAlertDefinition, binding.Bind(updateAlertDefinitionCommand{}), utils.Wrap(ng.updateAlertDefinitionEndpoint))
	})

	ng.RouteRegister.Group("/api/ngalert/", func(schedulerRouter routing.RouteRegister) {
		schedulerRouter.Post("/pause", utils.Wrap(ng.pauseScheduler))
		schedulerRouter.Post("/unpause", utils.Wrap(ng.unpauseScheduler))
	}, middleware.ReqOrgAdmin)
}

// conditionEvalEndpoint handles POST /api/alert-definitions/eval.
func (ng *AlertNG) conditionEvalEndpoint(c *models.ReqContext, dto evalAlertConditionCommand) utils.Response {
	if err := ng.validateCondition(dto.Condition, c.SignedInUser); err != nil {
		return utils.Error(400, "invalid condition", err)
	}

	evalResults, err := eval.ConditionEval(&dto.Condition, timeNow())
	if err != nil {
		return utils.Error(400, "Failed to evaluate conditions", err)
	}

	frame := evalResults.AsDataFrame()
	df := tsdb.NewDecodedDataFrames([]*data.Frame{&frame})
	instances, err := df.Encoded()
	if err != nil {
		return utils.Error(400, "Failed to encode result dataframes", err)
	}

	return utils.JSON(200, util.DynMap{
		"instances": instances,
	})
}

// alertDefinitionEvalEndpoint handles GET /api/alert-definitions/eval/:alertDefinitionUID.
func (ng *AlertNG) alertDefinitionEvalEndpoint(c *models.ReqContext) utils.Response {
	alertDefinitionUID := c.ParamsEscape(":alertDefinitionUID")

	condition, err := ng.LoadAlertCondition(alertDefinitionUID, c.SignedInUser.OrgId)
	if err != nil {
		return utils.Error(400, "Failed to load alert definition conditions", err)
	}

	if err := ng.validateCondition(*condition, c.SignedInUser); err != nil {
		return utils.Error(400, "invalid condition", err)
	}

	evalResults, err := eval.ConditionEval(condition, timeNow())
	if err != nil {
		return utils.Error(400, "Failed to evaluate alert", err)
	}
	frame := evalResults.AsDataFrame()

	df := tsdb.NewDecodedDataFrames([]*data.Frame{&frame})
	if err != nil {
		return utils.Error(400, "Failed to instantiate Dataframes from the decoded frames", err)
	}

	instances, err := df.Encoded()
	if err != nil {
		return utils.Error(400, "Failed to encode result dataframes", err)
	}
	return utils.JSON(200, util.DynMap{
		"instances": instances,
	})
}

// getAlertDefinitionEndpoint handles GET /api/alert-definitions/:alertDefinitionUID.
func (ng *AlertNG) getAlertDefinitionEndpoint(c *models.ReqContext) utils.Response {
	alertDefinitionUID := c.ParamsEscape(":alertDefinitionUID")

	query := getAlertDefinitionByUIDQuery{
		UID:   alertDefinitionUID,
		OrgID: c.SignedInUser.OrgId,
	}

	if err := ng.getAlertDefinitionByUID(&query); err != nil {
		return utils.Error(500, "Failed to get alert definition", err)
	}

	return utils.JSON(200, &query.Result)
}

// deleteAlertDefinitionEndpoint handles DELETE /api/alert-definitions/:alertDefinitionUID.
func (ng *AlertNG) deleteAlertDefinitionEndpoint(c *models.ReqContext) utils.Response {
	alertDefinitionUID := c.ParamsEscape(":alertDefinitionUID")

	cmd := deleteAlertDefinitionByUIDCommand{
		UID:   alertDefinitionUID,
		OrgID: c.SignedInUser.OrgId,
	}

	if err := ng.deleteAlertDefinitionByUID(&cmd); err != nil {
		return utils.Error(500, "Failed to delete alert definition", err)
	}

	return utils.Success("Alert definition deleted")
}

// updateAlertDefinitionEndpoint handles PUT /api/alert-definitions/:alertDefinitionUID.
func (ng *AlertNG) updateAlertDefinitionEndpoint(c *models.ReqContext, cmd updateAlertDefinitionCommand) utils.Response {
	cmd.UID = c.ParamsEscape(":alertDefinitionUID")
	cmd.OrgID = c.SignedInUser.OrgId

	if err := ng.validateCondition(cmd.Condition, c.SignedInUser); err != nil {
		return utils.Error(400, "invalid condition", err)
	}

	if err := ng.updateAlertDefinition(&cmd); err != nil {
		return utils.Error(500, "Failed to update alert definition", err)
	}

	return utils.JSON(200, cmd.Result)
}

// createAlertDefinitionEndpoint handles POST /api/alert-definitions.
func (ng *AlertNG) createAlertDefinitionEndpoint(c *models.ReqContext, cmd saveAlertDefinitionCommand) utils.Response {
	cmd.OrgID = c.SignedInUser.OrgId

	if err := ng.validateCondition(cmd.Condition, c.SignedInUser); err != nil {
		return utils.Error(400, "invalid condition", err)
	}

	if err := ng.saveAlertDefinition(&cmd); err != nil {
		return utils.Error(500, "Failed to create alert definition", err)
	}

	return utils.JSON(200, cmd.Result)
}

// listAlertDefinitions handles GET /api/alert-definitions.
func (ng *AlertNG) listAlertDefinitions(c *models.ReqContext) utils.Response {
	query := listAlertDefinitionsQuery{OrgID: c.SignedInUser.OrgId}

	if err := ng.getOrgAlertDefinitions(&query); err != nil {
		return utils.Error(500, "Failed to list alert definitions", err)
	}

	return utils.JSON(200, util.DynMap{"results": query.Result})
}

func (ng *AlertNG) pauseScheduler() utils.Response {
	err := ng.schedule.pause()
	if err != nil {
		return utils.Error(500, "Failed to pause scheduler", err)
	}
	return utils.JSON(200, util.DynMap{"message": "alert definition scheduler paused"})
}

func (ng *AlertNG) unpauseScheduler() utils.Response {
	err := ng.schedule.unpause()
	if err != nil {
		return utils.Error(500, "Failed to unpause scheduler", err)
	}
	return utils.JSON(200, util.DynMap{"message": "alert definition scheduler unpaused"})
}
