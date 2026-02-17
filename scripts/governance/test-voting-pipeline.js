#!/usr/bin/env node
// test-voting-pipeline.js — Local test harness for the governance voting pipeline.
// Tests core logic without requiring GitHub Actions runtime.
//
// Usage: node scripts/governance/test-voting-pipeline.js

const fs = require('fs');
const path = require('path');

// ============================================================
// Extract functions from tally-votes.js (avoid @actions/* imports)
// ============================================================

const QUORUM_MIN_VOTES = 3;
const SUPPORT_THRESHOLD_BPS = 7500;
const MIN_JUSTIFICATION_LENGTH = 10;
const MIN_ACCOUNT_AGE_DAYS = 7;
const TALLY_SENTINEL = '<!-- VOTE_TALLY -->';

function loadKnownAgents() {
  const registryPath = path.join(__dirname, 'known-agents.json');
  return JSON.parse(fs.readFileSync(registryPath, 'utf8'));
}

function lookupParticipant(registry, username) {
  const lower = username.toLowerCase();
  for (const [name, entry] of Object.entries(registry.agents || {})) {
    if (name.toLowerCase() === lower) {
      return { type: 'agent', stake: entry.stake || '0', did: entry.did, tier: entry.tier };
    }
  }
  for (const [name, entry] of Object.entries(registry.humans || {})) {
    if (name.toLowerCase() === lower) {
      return { type: 'human', stake: entry.stake || '0', role: entry.role };
    }
  }
  return null;
}

function quadraticWeight(stakeStr) {
  if (!stakeStr || stakeStr === '0') return 1;
  const stake = BigInt(stakeStr);
  if (stake <= 0n) return 1;
  let x = stake;
  let y = (x + 1n) / 2n;
  while (y < x) {
    x = y;
    y = (x + stake / x) / 2n;
  }
  return Number(x) || 1;
}

function parseVote(commentBody) {
  const lines = commentBody.split('\n').map(l => l.trim());
  let direction = null;
  let justification = null;
  let agentDid = null;

  for (const line of lines) {
    const voteMatch = line.match(/^\/vote\s+(support|oppose|abstain)\s*$/i);
    if (voteMatch) direction = voteMatch[1].toLowerCase();

    const justMatch = line.match(/^Justification:\s*(.+)$/i);
    if (justMatch) justification = justMatch[1].trim();

    const didMatch = line.match(/^Agent-DID:\s*(did:lgm:\S+)\s*$/i);
    if (didMatch) agentDid = didMatch[1];
  }

  if (!direction) return null;
  return { direction, justification, agentDid };
}

function isAccountOldEnough(userCreatedAt) {
  const created = new Date(userCreatedAt);
  const now = new Date();
  const diffDays = (now - created) / (1000 * 60 * 60 * 24);
  return diffDays >= MIN_ACCOUNT_AGE_DAYS;
}

// Builds tally with an injectable registry (for testing)
function buildTallyComment(votes, issueAuthor, registry) {
  const validVotes = [];
  const rejectedVotes = [];

  for (const [user, vote] of Object.entries(votes)) {
    const participant = lookupParticipant(registry, user);
    if (!participant) {
      rejectedVotes.push({ user, reason: 'Not a registered ecosystem participant' });
      continue;
    }
    if (user.toLowerCase() === issueAuthor.toLowerCase()) {
      rejectedVotes.push({ user, reason: 'Self-voting not allowed' });
      continue;
    }
    if (!vote.justification || vote.justification.length < MIN_JUSTIFICATION_LENGTH) {
      rejectedVotes.push({ user, reason: `Justification too short (need ${MIN_JUSTIFICATION_LENGTH}+ chars)` });
      continue;
    }
    if (!vote.accountOldEnough) {
      rejectedVotes.push({ user, reason: `Account younger than ${MIN_ACCOUNT_AGE_DAYS} days` });
      continue;
    }

    const stakeWeight = quadraticWeight(participant.stake);
    validVotes.push({
      user, direction: vote.direction, justification: vote.justification,
      agentDid: vote.agentDid || participant.did || null,
      weight: stakeWeight, participantType: participant.type,
    });
  }

  let supportWeight = 0, opposeWeight = 0, abstainWeight = 0;
  for (const v of validVotes) {
    if (v.direction === 'support') supportWeight += v.weight;
    else if (v.direction === 'oppose') opposeWeight += v.weight;
    else if (v.direction === 'abstain') abstainWeight += v.weight;
  }

  const totalVotes = validVotes.length;
  const quorumMet = totalVotes >= QUORUM_MIN_VOTES;
  const supportPct = (supportWeight + opposeWeight) > 0
    ? Math.round((supportWeight / (supportWeight + opposeWeight)) * 10000)
    : 0;

  let status;
  if (!quorumMet) status = 'Voting Open — Quorum Not Met';
  else if (supportPct >= SUPPORT_THRESHOLD_BPS) status = 'Support Majority';
  else if (supportPct >= 4000) status = 'Contested';
  else status = 'Opposed';

  const labels = [];
  if (quorumMet) labels.push('vote:quorum-met');
  if (quorumMet && supportPct >= SUPPORT_THRESHOLD_BPS) labels.push('vote:support-majority');
  else if (quorumMet && supportPct >= 4000) labels.push('vote:contested');
  else if (quorumMet) labels.push('vote:opposed');

  return { validVotes, rejectedVotes, labels, status, quorumMet, supportPct, supportWeight, opposeWeight, abstainWeight };
}

// ============================================================
// Test runner
// ============================================================

let passed = 0;
let failed = 0;

function assert(condition, testName) {
  if (condition) {
    passed++;
    console.log(`  PASS  ${testName}`);
  } else {
    failed++;
    console.log(`  FAIL  ${testName}`);
  }
}

function assertEq(actual, expected, testName) {
  if (actual === expected) {
    passed++;
    console.log(`  PASS  ${testName}`);
  } else {
    failed++;
    console.log(`  FAIL  ${testName} — expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
}

// ============================================================
// Test registry for simulated voting
// ============================================================

const TEST_REGISTRY = {
  agents: {
    'agent-alpha': { did: 'did:lgm:aaa111', stake: '1000000', tier: 2 },
    'agent-beta':  { did: 'did:lgm:bbb222', stake: '250000',  tier: 1 },
    'agent-gamma': { did: 'did:lgm:ccc333', stake: '4000000', tier: 3 },
    'agent-zero':  { did: 'did:lgm:000000', stake: '0',       tier: 1 },
  },
  humans: {
    'yournameisai': { role: 'manager', stake: '500000' },
    'contributor-1': { role: 'contributor' },
  }
};

// ============================================================
// 1. Parse Vote Tests
// ============================================================

console.log('\n=== 1. PARSE VOTE ===\n');

assertEq(
  parseVote('/vote support\nJustification: This improves the protocol significantly\nAgent-DID: did:lgm:aaa111').direction,
  'support', 'Parse: support direction'
);

assertEq(
  parseVote('/vote oppose\nJustification: This is a bad idea for many reasons').direction,
  'oppose', 'Parse: oppose direction'
);

assertEq(
  parseVote('/vote abstain\nJustification: Need more information to decide').direction,
  'abstain', 'Parse: abstain direction'
);

assertEq(
  parseVote('/vote SUPPORT\nJustification: Case insensitive test works').direction,
  'support', 'Parse: case insensitive direction'
);

assertEq(
  parseVote('Just a regular comment, no vote here'),
  null, 'Parse: no vote command returns null'
);

assertEq(
  parseVote('/vote invalid\nJustification: bad direction'),
  null, 'Parse: invalid direction returns null'
);

assertEq(
  parseVote('/vote support\nJustification: Solid improvement to knowledge verification\nAgent-DID: did:lgm:abc123def456').agentDid,
  'did:lgm:abc123def456', 'Parse: extracts Agent-DID'
);

assertEq(
  parseVote('/vote support\nJustification: No DID provided here').agentDid,
  null, 'Parse: missing DID is null'
);

assertEq(
  parseVote('/vote support').justification,
  null, 'Parse: missing justification is null'
);

assertEq(
  parseVote('/vote support extra text'),
  null, 'Parse: extra text after direction is rejected'
);

// ============================================================
// 2. Quadratic Weight Tests
// ============================================================

console.log('\n=== 2. QUADRATIC WEIGHT (sqrt(stake)) ===\n');

assertEq(quadraticWeight('0'), 1, 'Weight: zero stake = 1');
assertEq(quadraticWeight(null), 1, 'Weight: null stake = 1');
assertEq(quadraticWeight(''), 1, 'Weight: empty stake = 1');
assertEq(quadraticWeight('1'), 1, 'Weight: sqrt(1) = 1');
assertEq(quadraticWeight('4'), 2, 'Weight: sqrt(4) = 2');
assertEq(quadraticWeight('100'), 10, 'Weight: sqrt(100) = 10');
assertEq(quadraticWeight('1000000'), 1000, 'Weight: sqrt(1M) = 1000');
assertEq(quadraticWeight('250000'), 500, 'Weight: sqrt(250K) = 500');
assertEq(quadraticWeight('4000000'), 2000, 'Weight: sqrt(4M) = 2000');

// Verify non-perfect squares floor correctly
assertEq(quadraticWeight('2'), 1, 'Weight: sqrt(2) = 1 (floor)');
assertEq(quadraticWeight('10'), 3, 'Weight: sqrt(10) = 3 (floor)');
assertEq(quadraticWeight('999999'), 999, 'Weight: sqrt(999999) = 999 (floor)');

// Large stake
assertEq(quadraticWeight('1000000000000'), 1000000, 'Weight: sqrt(1T) = 1M');

// ============================================================
// 3. Participant Lookup Tests
// ============================================================

console.log('\n=== 3. PARTICIPANT LOOKUP ===\n');

assert(lookupParticipant(TEST_REGISTRY, 'agent-alpha') !== null, 'Lookup: registered agent found');
assertEq(lookupParticipant(TEST_REGISTRY, 'agent-alpha').type, 'agent', 'Lookup: agent type correct');
assertEq(lookupParticipant(TEST_REGISTRY, 'agent-alpha').stake, '1000000', 'Lookup: agent stake correct');
assertEq(lookupParticipant(TEST_REGISTRY, 'agent-alpha').did, 'did:lgm:aaa111', 'Lookup: agent DID correct');

assert(lookupParticipant(TEST_REGISTRY, 'yournameisai') !== null, 'Lookup: registered human found');
assertEq(lookupParticipant(TEST_REGISTRY, 'yournameisai').type, 'human', 'Lookup: human type correct');
assertEq(lookupParticipant(TEST_REGISTRY, 'yournameisai').role, 'manager', 'Lookup: human role correct');

assertEq(lookupParticipant(TEST_REGISTRY, 'random-stranger'), null, 'Lookup: unregistered user returns null');
assertEq(lookupParticipant(TEST_REGISTRY, 'hacker42'), null, 'Lookup: unknown user returns null');

// Case-insensitive
assert(lookupParticipant(TEST_REGISTRY, 'Agent-Alpha') !== null, 'Lookup: case insensitive match');
assert(lookupParticipant(TEST_REGISTRY, 'YOURNAMEISAI') !== null, 'Lookup: case insensitive human match');

// Zero-stake agent still found
assert(lookupParticipant(TEST_REGISTRY, 'agent-zero') !== null, 'Lookup: zero-stake agent still registered');

// No-stake human still found
assert(lookupParticipant(TEST_REGISTRY, 'contributor-1') !== null, 'Lookup: no-stake human still registered');

// ============================================================
// 4. Account Age Tests
// ============================================================

console.log('\n=== 4. ACCOUNT AGE ===\n');

const now = new Date();
const thirtyDaysAgo = new Date(now - 30 * 24 * 60 * 60 * 1000).toISOString();
const twoDaysAgo = new Date(now - 2 * 24 * 60 * 60 * 1000).toISOString();
const exactlySevenDays = new Date(now - 7 * 24 * 60 * 60 * 1000).toISOString();
const sixDays = new Date(now - 6 * 24 * 60 * 60 * 1000).toISOString();

assertEq(isAccountOldEnough(thirtyDaysAgo), true, 'Age: 30 days old = old enough');
assertEq(isAccountOldEnough(twoDaysAgo), false, 'Age: 2 days old = too new');
assertEq(isAccountOldEnough(exactlySevenDays), true, 'Age: exactly 7 days = old enough');
assertEq(isAccountOldEnough(sixDays), false, 'Age: 6 days = too new');
assertEq(isAccountOldEnough('2020-01-01T00:00:00Z'), true, 'Age: 2020 account = old enough');

// ============================================================
// 5. Anti-Spam Gates (Full Tally)
// ============================================================

console.log('\n=== 5. ANTI-SPAM GATES ===\n');

// 5a. Unregistered user rejected
{
  const votes = {
    'random-stranger': {
      direction: 'support', justification: 'This is a great proposal!',
      accountOldEnough: true, agentDid: null,
    }
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  assertEq(result.rejectedVotes.length, 1, 'Gate: unregistered user rejected');
  assert(result.rejectedVotes[0].reason.includes('Not a registered'), 'Gate: rejection reason mentions registration');
  assertEq(result.validVotes.length, 0, 'Gate: no valid votes from unregistered');
}

// 5b. Self-vote rejected
{
  const votes = {
    'yournameisai': {
      direction: 'support', justification: 'My own proposal is great!',
      accountOldEnough: true, agentDid: null,
    }
  };
  const result = buildTallyComment(votes, 'yournameisai', TEST_REGISTRY);
  assertEq(result.rejectedVotes.length, 1, 'Gate: self-vote rejected');
  assert(result.rejectedVotes[0].reason.includes('Self-voting'), 'Gate: self-vote reason correct');
}

// 5c. Short justification rejected
{
  const votes = {
    'agent-alpha': {
      direction: 'support', justification: 'yes',
      accountOldEnough: true, agentDid: null,
    }
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  assertEq(result.rejectedVotes.length, 1, 'Gate: short justification rejected');
  assert(result.rejectedVotes[0].reason.includes('too short'), 'Gate: justification reason correct');
}

// 5d. Missing justification rejected
{
  const votes = {
    'agent-alpha': {
      direction: 'support', justification: null,
      accountOldEnough: true, agentDid: null,
    }
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  assertEq(result.rejectedVotes.length, 1, 'Gate: null justification rejected');
}

// 5e. New account rejected
{
  const votes = {
    'agent-alpha': {
      direction: 'support', justification: 'Valid justification here!',
      accountOldEnough: false, agentDid: null,
    }
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  assertEq(result.rejectedVotes.length, 1, 'Gate: new account rejected');
  assert(result.rejectedVotes[0].reason.includes('younger'), 'Gate: account age reason correct');
}

// ============================================================
// 6. Stake-Weighted Quadratic Tally
// ============================================================

console.log('\n=== 6. STAKE-WEIGHTED TALLY ===\n');

// 6a. Single agent vote — weight = sqrt(1M) = 1000
{
  const votes = {
    'agent-alpha': {
      direction: 'support', justification: 'Improves verification throughput significantly',
      accountOldEnough: true, agentDid: 'did:lgm:aaa111',
    }
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  assertEq(result.validVotes.length, 1, 'Tally: 1 valid vote');
  assertEq(result.validVotes[0].weight, 1000, 'Tally: agent-alpha weight = sqrt(1M) = 1000');
  assertEq(result.supportWeight, 1000, 'Tally: support weight = 1000');
  assertEq(result.quorumMet, false, 'Tally: quorum not met with 1 vote');
}

// 6b. Three agents — quorum met, all support
{
  const votes = {
    'agent-alpha': {
      direction: 'support', justification: 'Strong improvement to the protocol',
      accountOldEnough: true, agentDid: 'did:lgm:aaa111',
    },
    'agent-beta': {
      direction: 'support', justification: 'Aligns with protocol roadmap nicely',
      accountOldEnough: true, agentDid: 'did:lgm:bbb222',
    },
    'agent-gamma': {
      direction: 'support', justification: 'Good for ecosystem growth overall',
      accountOldEnough: true, agentDid: 'did:lgm:ccc333',
    },
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  assertEq(result.validVotes.length, 3, 'Tally: 3 valid votes');
  assertEq(result.quorumMet, true, 'Tally: quorum met with 3 votes');
  // alpha=1000, beta=500, gamma=2000 → total support=3500
  assertEq(result.supportWeight, 3500, 'Tally: combined support weight = 3500');
  assertEq(result.supportPct, 10000, 'Tally: 100% support');
  assertEq(result.status, 'Support Majority', 'Tally: status = Support Majority');
  assert(result.labels.includes('vote:support-majority'), 'Tally: support-majority label set');
  assert(result.labels.includes('vote:quorum-met'), 'Tally: quorum-met label set');
}

// 6c. Mixed votes — contested scenario
{
  const votes = {
    'agent-alpha': {
      direction: 'support', justification: 'Good change for knowledge verification',
      accountOldEnough: true,
    },
    'agent-beta': {
      direction: 'oppose', justification: 'This breaks backward compatibility badly',
      accountOldEnough: true,
    },
    'agent-gamma': {
      direction: 'support', justification: 'Net positive for the ecosystem overall',
      accountOldEnough: true,
    },
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  // support: alpha(1000) + gamma(2000) = 3000, oppose: beta(500) = 500
  // pct = 3000/(3000+500) = 85.7% = 8571 bps → Support Majority
  assertEq(result.supportWeight, 3000, 'Tally: support weight = 3000');
  assertEq(result.opposeWeight, 500, 'Tally: oppose weight = 500');
  assertEq(result.status, 'Support Majority', 'Tally: 85.7% → Support Majority');
}

// 6d. Majority oppose — below 40%
{
  const votes = {
    'agent-alpha': {
      direction: 'oppose', justification: 'Fundamentally flawed proposal design',
      accountOldEnough: true,
    },
    'agent-beta': {
      direction: 'oppose', justification: 'Agree this needs major rethinking first',
      accountOldEnough: true,
    },
    'agent-gamma': {
      direction: 'support', justification: 'I still think it has some merit though',
      accountOldEnough: true,
    },
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  // support: gamma(2000) = 2000, oppose: alpha(1000) + beta(500) = 1500
  // pct = 2000/(2000+1500) = 57.1% = 5714 bps → Contested
  assertEq(result.status, 'Contested', 'Tally: 57.1% support → Contested');
  assert(result.labels.includes('vote:contested'), 'Tally: contested label set');
}

// 6e. Heavily opposed — below 40%
{
  const votes = {
    'agent-alpha': {
      direction: 'oppose', justification: 'This would break consensus guarantees',
      accountOldEnough: true,
    },
    'agent-gamma': {
      direction: 'oppose', justification: 'Too risky for the current network state',
      accountOldEnough: true,
    },
    'yournameisai': {
      direction: 'support', justification: 'I think the risk is manageable here',
      accountOldEnough: true,
    },
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  // support: manager(sqrt(500000)=707) = 707, oppose: alpha(1000)+gamma(2000)=3000
  // pct = 707/(707+3000) = 19.1% = 1907 bps → Opposed
  assertEq(result.status, 'Opposed', 'Tally: 19% support → Opposed');
  assert(result.labels.includes('vote:opposed'), 'Tally: opposed label set');
}

// 6f. Abstentions count toward quorum but not ratio
{
  const votes = {
    'agent-alpha': {
      direction: 'support', justification: 'Solid proposal with clear benefits',
      accountOldEnough: true,
    },
    'agent-beta': {
      direction: 'abstain', justification: 'Need more data before taking a position',
      accountOldEnough: true,
    },
    'agent-gamma': {
      direction: 'abstain', justification: 'Waiting for security audit results first',
      accountOldEnough: true,
    },
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  assertEq(result.quorumMet, true, 'Tally: abstentions count toward quorum');
  assertEq(result.supportPct, 10000, 'Tally: abstentions excluded from support ratio → 100%');
  assertEq(result.abstainWeight, 2500, 'Tally: abstain weight = 500 + 2000 = 2500');
}

// 6g. Zero-stake agent gets weight 1
{
  const votes = {
    'agent-zero': {
      direction: 'support', justification: 'Even with no stake, I support this',
      accountOldEnough: true,
    },
    'agent-alpha': {
      direction: 'support', justification: 'Strong backing from a staked agent',
      accountOldEnough: true,
    },
    'contributor-1': {
      direction: 'support', justification: 'Human contributor also supports this',
      accountOldEnough: true,
    },
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  const zeroVote = result.validVotes.find(v => v.user === 'agent-zero');
  assertEq(zeroVote.weight, 1, 'Tally: zero-stake agent gets weight 1');
  const humanVote = result.validVotes.find(v => v.user === 'contributor-1');
  assertEq(humanVote.weight, 1, 'Tally: no-stake human gets weight 1');
  assertEq(humanVote.participantType, 'human', 'Tally: human type preserved');
  // total support: 1 + 1000 + 1 = 1002
  assertEq(result.supportWeight, 1002, 'Tally: combined weight = 1 + 1000 + 1 = 1002');
}

// ============================================================
// 7. Deduplication (last vote wins)
// ============================================================

console.log('\n=== 7. DEDUPLICATION ===\n');

// Simulating last-vote-wins at the comment-parsing level
{
  const comments = [
    '/vote support\nJustification: Initially in favor of this change',
    '/vote oppose\nJustification: Changed my mind after more review',
  ];

  // Parse both — last one should be the one used
  const first = parseVote(comments[0]);
  const second = parseVote(comments[1]);
  assertEq(first.direction, 'support', 'Dedup: first vote is support');
  assertEq(second.direction, 'oppose', 'Dedup: second vote is oppose');

  // In the bot, last vote per user wins (object key overwrite)
  const votes = {};
  for (const body of comments) {
    const parsed = parseVote(body);
    if (parsed) {
      votes['agent-alpha'] = {
        direction: parsed.direction,
        justification: parsed.justification,
        accountOldEnough: true,
      };
    }
  }
  assertEq(votes['agent-alpha'].direction, 'oppose', 'Dedup: last vote wins (oppose)');
}

// ============================================================
// 8. Edge Cases
// ============================================================

console.log('\n=== 8. EDGE CASES ===\n');

// 8a. Empty votes → no quorum
{
  const result = buildTallyComment({}, 'someone', TEST_REGISTRY);
  assertEq(result.quorumMet, false, 'Edge: empty votes → no quorum');
  assertEq(result.validVotes.length, 0, 'Edge: empty votes → 0 valid');
  assertEq(result.status, 'Voting Open — Quorum Not Met', 'Edge: empty → voting open');
}

// 8b. All votes rejected → no quorum
{
  const votes = {
    'stranger-1': { direction: 'support', justification: 'test test test test', accountOldEnough: true },
    'stranger-2': { direction: 'support', justification: 'test test test test', accountOldEnough: true },
    'stranger-3': { direction: 'support', justification: 'test test test test', accountOldEnough: true },
  };
  const result = buildTallyComment(votes, 'someone', TEST_REGISTRY);
  assertEq(result.validVotes.length, 0, 'Edge: all unregistered → 0 valid');
  assertEq(result.rejectedVotes.length, 3, 'Edge: all 3 rejected');
  assertEq(result.quorumMet, false, 'Edge: rejected votes don\'t count toward quorum');
}

// 8c. Mixed valid + rejected
{
  const votes = {
    'agent-alpha': { direction: 'support', justification: 'Legitimate support from registered agent', accountOldEnough: true },
    'random-hacker': { direction: 'support', justification: 'Trying to game the vote!', accountOldEnough: true },
    'agent-beta': { direction: 'support', justification: 'Another registered agent supports', accountOldEnough: true },
  };
  const result = buildTallyComment(votes, 'someone', TEST_REGISTRY);
  assertEq(result.validVotes.length, 2, 'Edge: 2 valid, 1 rejected');
  assertEq(result.rejectedVotes.length, 1, 'Edge: 1 rejected (unregistered)');
  assertEq(result.quorumMet, false, 'Edge: 2 valid < 3 quorum');
}

// 8d. Participant type preserved in votes
{
  const votes = {
    'agent-alpha': { direction: 'support', justification: 'Agent voting on protocol', accountOldEnough: true },
    'yournameisai': { direction: 'support', justification: 'Human manager also supports', accountOldEnough: true },
    'agent-beta': { direction: 'support', justification: 'Another agent in support', accountOldEnough: true },
  };
  const result = buildTallyComment(votes, 'someone-else', TEST_REGISTRY);
  const agentVote = result.validVotes.find(v => v.user === 'agent-alpha');
  const humanVote = result.validVotes.find(v => v.user === 'yournameisai');
  assertEq(agentVote.participantType, 'agent', 'Edge: agent type preserved');
  assertEq(humanVote.participantType, 'human', 'Edge: human type preserved');
}

// ============================================================
// 9. Real Registry Test (uses actual known-agents.json)
// ============================================================

console.log('\n=== 9. REAL REGISTRY ===\n');

{
  const realRegistry = loadKnownAgents();
  assert(realRegistry.humans.yournameisai !== undefined, 'Registry: manager exists');
  assertEq(realRegistry.humans.yournameisai.role, 'manager', 'Registry: manager role correct');

  // Unregistered user blocked against real registry
  const participant = lookupParticipant(realRegistry, 'some-random-github-user');
  assertEq(participant, null, 'Registry: unregistered user blocked in real registry');

  // Manager found
  const manager = lookupParticipant(realRegistry, 'yournameisai');
  assert(manager !== null, 'Registry: manager found in real registry');
  assertEq(manager.type, 'human', 'Registry: manager is human type');
}

// ============================================================
// Summary
// ============================================================

console.log('\n' + '='.repeat(50));
console.log(`\n  ${passed + failed} tests | ${passed} passed | ${failed} failed\n`);

if (failed > 0) {
  console.log('  SOME TESTS FAILED\n');
  process.exit(1);
} else {
  console.log('  ALL TESTS PASSED\n');
  process.exit(0);
}
