/**
 * Sustainability RPC Handlers
 *
 * Provides visibility into the perpetual truth-finding loop health,
 * research fund status, efficiency metrics, and sustainability forecast.
 *
 * Endpoints:
 * - sustainability_getLoopHealth: Overall loop health score and component breakdown
 * - sustainability_getResearchFund: Research fund balance and flow metrics
 * - sustainability_getEfficiency: Loop efficiency and monetization rates
 * - sustainability_getForecast: Sustainability forecast and crossover projection
 * - sustainability_getBountyStatus: Bounty pool metrics
 */

import type { Handler } from "../handlers.js";
import type {
  SustainabilityGetLoopHealthRequest,
  SustainabilityGetLoopHealthResponse,
  SustainabilityGetResearchFundRequest,
  SustainabilityGetResearchFundResponse,
  SustainabilityGetEfficiencyRequest,
  SustainabilityGetEfficiencyResponse,
  SustainabilityGetForecastRequest,
  SustainabilityGetForecastResponse,
  SustainabilityGetBountyStatusRequest,
  SustainabilityGetBountyStatusResponse,
} from "../../types.js";

// ============ Handlers ============

export const sustainabilityGetLoopHealth: Handler<
  SustainabilityGetLoopHealthRequest,
  SustainabilityGetLoopHealthResponse
> = async (_params, ctx) => {
  const blockNumber = ctx.state.getBlockNumber();

  if (!ctx.loopController) {
    return {
      healthScore: 0,
      components: {},
      blockNumber: blockNumber.toString(),
    };
  }

  const state = ctx.loopController.getState();
  const components: Record<
    string,
    { status: string; score: number; message: string }
  > = {};

  for (const [key, component] of Object.entries(state.health.components)) {
    components[key] = {
      status: component.status,
      score: component.score,
      message: `${component.metric} (target: ${component.target})`,
    };
  }

  return {
    healthScore: state.health.healthScore,
    components,
    blockNumber: blockNumber.toString(),
  };
};

export const sustainabilityGetResearchFund: Handler<
  SustainabilityGetResearchFundRequest,
  SustainabilityGetResearchFundResponse
> = async (_params, ctx) => {
  if (!ctx.loopController) {
    return {
      balance: "0",
      totalInflow: "0",
      totalBountyFunding: "0",
      inflowPerBlock: "0",
    };
  }

  const state = ctx.loopController.getState();

  return {
    balance: state.researchFund.balance.toString(),
    totalInflow: state.researchFund.totalInflow.toString(),
    totalBountyFunding: state.researchFund.totalBountyFunding.toString(),
    inflowPerBlock: state.researchFund.inflowPerBlock.toString(),
  };
};

export const sustainabilityGetEfficiency: Handler<
  SustainabilityGetEfficiencyRequest,
  SustainabilityGetEfficiencyResponse
> = async (_params, ctx) => {
  if (!ctx.loopController) {
    return {
      bountyConversionRate: 0,
      truthDiscoveryCost: "0",
      apiValuePerFact: "0",
      knowledgeMonetizationRate: "0",
      loopVelocity: 0,
    };
  }

  const efficiency = ctx.loopController.getEfficiency();

  return {
    bountyConversionRate: efficiency.bountyConversionRate,
    truthDiscoveryCost: efficiency.truthDiscoveryCost.toString(),
    apiValuePerFact: efficiency.apiValuePerFact.toString(),
    knowledgeMonetizationRate: efficiency.knowledgeMonetizationRate.toString(),
    loopVelocity: efficiency.loopVelocity,
  };
};

export const sustainabilityGetForecast: Handler<
  SustainabilityGetForecastRequest,
  SustainabilityGetForecastResponse
> = async (params, ctx) => {
  if (!ctx.loopController) {
    return {
      currentBlockReward: "0",
      apiRevenuePerBlock: "0",
      researchInflowPerBlock: "0",
      researchFundBalance: "0",
      activeBountiesValue: "0",
      sustainabilityRatio: 0,
      blocksUntilCrossover: null,
      isSelfSustaining: false,
    };
  }

  const futureBlocks = params.futureBlocks
    ? BigInt(params.futureBlocks)
    : 240_000n; // Default: 1 epoch
  const forecast = ctx.loopController.getSustainabilityForecast(futureBlocks);

  return {
    currentBlockReward: forecast.currentBlockReward.toString(),
    apiRevenuePerBlock: forecast.apiRevenuePerBlock.toString(),
    researchInflowPerBlock: forecast.researchInflowPerBlock.toString(),
    researchFundBalance: forecast.researchFundBalance.toString(),
    activeBountiesValue: forecast.activeBountiesValue.toString(),
    sustainabilityRatio: forecast.sustainabilityRatio,
    blocksUntilCrossover: forecast.blocksUntilCrossover?.toString() ?? null,
    isSelfSustaining: forecast.isSelfSustaining,
  };
};

export const sustainabilityGetBountyStatus: Handler<
  SustainabilityGetBountyStatusRequest,
  SustainabilityGetBountyStatusResponse
> = async (_params, ctx) => {
  if (!ctx.loopController) {
    return {
      activeBountiesValue: "0",
      activeBountiesCount: 0,
      bountiesCreatedThisEpoch: 0,
      bountiesFulfilledThisEpoch: 0,
      averageBountyValue: "0",
    };
  }

  const state = ctx.loopController.getState();

  return {
    activeBountiesValue: state.bountyPool.activeBountiesValue.toString(),
    activeBountiesCount: state.bountyPool.activeBountiesCount,
    bountiesCreatedThisEpoch: state.bountyPool.bountiesCreatedThisEpoch,
    bountiesFulfilledThisEpoch: state.bountyPool.bountiesFulfilledThisEpoch,
    averageBountyValue: state.bountyPool.averageBountyValue.toString(),
  };
};

// ============ Handler Registry ============

export const sustainabilityHandlers = {
  sustainability_getLoopHealth: sustainabilityGetLoopHealth,
  sustainability_getResearchFund: sustainabilityGetResearchFund,
  sustainability_getEfficiency: sustainabilityGetEfficiency,
  sustainability_getForecast: sustainabilityGetForecast,
  sustainability_getBountyStatus: sustainabilityGetBountyStatus,
};
