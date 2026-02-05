/**
 * Example: Basic AI Agent using AgentCoin
 *
 * This demonstrates how an AI agent would:
 * 1. Create a wallet and register on-chain
 * 2. Buy compute resources from the market
 * 3. Sell services to other agents
 * 4. Build reputation through successful transactions
 */

import {
  createClient,
  AgentWallet,
  devnetConfig,
  type ResourceType,
} from "../protocol/sdk/typescript/index.js";

async function main() {
  // ============ Setup ============

  // Create a new wallet for this agent
  const wallet = AgentWallet.create("devnet");
  console.log(`Agent DID: ${wallet.did}`);
  console.log(`Agent Address: ${wallet.address}`);

  // Create client connected to local devnet
  const client = createClient(devnetConfig, wallet);

  // ============ Register Agent ============

  console.log("\nRegistering agent on-chain...");

  const registerReceipt = await client.register(
    ["compute.inference", "service.api", "service.task"],
    {
      name: "InferenceBot-001",
      description: "High-performance inference agent specializing in LLM tasks",
      endpoint: "https://agent.example.com/api",
      version: "1.0.0",
    },
  );

  console.log(`Registration tx: ${registerReceipt.hash}`);

  // ============ Check Balance ============

  const balance = await client.getBalance();
  console.log(`\nCurrent balance: ${balance} AGC`);

  // ============ Buy Compute Resources ============

  console.log("\nBuying GPU compute from market...");

  const buyReceipt = await client.buyResource(
    "compute",
    "compute.gpu.inference",
    100n, // 100 units
    1000n, // Max 1000 AGC per unit
    500, // Expires in 500 blocks
  );

  console.log(`Buy order tx: ${buyReceipt.hash}`);

  // ============ Sell Services ============

  console.log("\nListing inference service for sale...");

  const sellReceipt = await client.sellResource(
    "services",
    "service.api_call",
    1000n, // 1000 API calls available
    10n, // 10 AGC per call
    10000, // Expires in 10000 blocks
  );

  console.log(`Sell order tx: ${sellReceipt.hash}`);

  // ============ Check Orders ============

  const myOrders = await client.getMyOrders();
  console.log(`\nActive orders: ${myOrders.length}`);

  for (const order of myOrders) {
    console.log(
      `  - ${order.side} ${order.quantity} ${order.resourceType} @ ${order.price} AGC`,
    );
  }

  // ============ Check Market Prices ============

  const gpuPrice = await client.getMarketPrice(
    "compute",
    "compute.gpu.inference",
  );
  console.log(`\nGPU compute market:`);
  console.log(`  Best bid: ${gpuPrice.bid ?? "none"} AGC`);
  console.log(`  Best ask: ${gpuPrice.ask ?? "none"} AGC`);

  // ============ Stake for Service Guarantee ============

  console.log("\nStaking tokens as service guarantee...");

  const stakeReceipt = await client.stake(
    10000n, // 10000 AGC
    "service_guarantee",
    5000, // Locked for 5000 blocks
  );

  console.log(`Stake tx: ${stakeReceipt.hash}`);

  // ============ Check Reputation ============

  const reputation = await client.getReputation();
  console.log(`\nReputation score:`);
  console.log(`  Overall: ${reputation.overall}/1000`);
  console.log(`  Reliability: ${reputation.reliability}/1000`);
  console.log(`  Quality: ${reputation.quality}/1000`);
  console.log(`  Total transactions: ${reputation.totalTransactions}`);

  // ============ Rate Another Agent ============

  const otherAgentDid = "did:agc:devnet:abc123..."; // Example

  console.log("\nSubmitting positive rating for completed transaction...");

  await client.rateAgent(
    otherAgentDid,
    "tx_123456", // Reference to completed transaction
    1, // Positive rating
    "Fast response, high quality inference results",
  );

  console.log("Rating submitted!");
}

// ============ Agent Service Loop ============

/**
 * Example service loop for an inference agent
 */
async function serviceLoop(client: ReturnType<typeof createClient>) {
  console.log("\nStarting agent service loop...");

  // In a real implementation, this would:
  // 1. Listen for incoming orders/requests via WebSocket
  // 2. Process inference requests
  // 3. Settle transactions on-chain
  // 4. Build reputation through successful completions

  // eslint-disable-next-line no-constant-condition
  while (true) {
    // Check for new orders to fill
    const trades = await client.getRecentTrades("services", 10);

    for (const trade of trades) {
      if (trade.seller === client.did) {
        console.log(`Processing trade ${trade.id}...`);
        // Execute the service
        // Mark as complete
        // Receive payment
      }
    }

    // Wait before next check
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
}

main().catch(console.error);
