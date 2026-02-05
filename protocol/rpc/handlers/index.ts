/**
 * RPC Handler Exports
 *
 * B3 FIX: Exports all handler modules for registration.
 */

// Individual handler exports
export {
  lgmStake,
  lgmUnstake,
  lgmDelegate,
  lgmUndelegate,
  stakingHandlers,
} from "./staking.js";

export { lgmSendTransaction, transactionHandlers } from "./transactions.js";

export {
  lgmOpenDispute,
  lgmSubmitEvidence,
  lgmResolveDispute,
  disputeHandlers,
} from "./disputes.js";

export {
  lgmOpenChannel,
  lgmCloseChannel,
  channelHandlers,
} from "./channels.js";

export { sustainabilityHandlers } from "./sustainability.js";

export { lgmRotateKey, lgmGetKeyInfo, identityHandlers } from "./identity.js";

export { treeHandlers } from "./tree.js";

export { governanceHandlers } from "./governance.js";

// Combined handlers for easy registration
import { stakingHandlers } from "./staking.js";
import { transactionHandlers } from "./transactions.js";
import { disputeHandlers } from "./disputes.js";
import { channelHandlers } from "./channels.js";
import { sustainabilityHandlers } from "./sustainability.js";
import { identityHandlers } from "./identity.js";
import { treeHandlers } from "./tree.js";
import { governanceHandlers } from "./governance.js";

/**
 * All B3 handlers combined
 */
export const b3Handlers = {
  ...stakingHandlers,
  ...transactionHandlers,
  ...disputeHandlers,
  ...channelHandlers,
  ...sustainabilityHandlers,
  ...identityHandlers,
  ...treeHandlers,
  ...governanceHandlers,
};
