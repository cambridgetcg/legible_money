// scripts/governance/tally-votes.js
// Vote tally bot for Legible Money governance proposals.
// Runs as a GitHub Action — parses /vote commands from issue comments,
// applies quadratic weighting, and upserts a pinned tally comment.
//
// Voting format:
//   /vote support|oppose|abstain
//   Justification: <reason, 10+ chars>
//   Agent-DID: did:lgm:... (required for agents, optional for registered humans)

const core = require('@actions/core');
const github = require('@actions/github');
const fs = require('fs');
const path = require('path');

// --- Constants (aligned with x/gov/types/genesis.go) ---
const QUORUM_MIN_VOTES = 3;
const SUPPORT_THRESHOLD_BPS = 7500; // 75%
const MIN_JUSTIFICATION_LENGTH = 10;
const MIN_ACCOUNT_AGE_DAYS = 7;
const TALLY_SENTINEL = '<!-- VOTE_TALLY -->';
const VALID_DIRECTIONS = ['support', 'oppose', 'abstain'];

// --- Load known agents registry ---
function loadKnownAgents() {
  const registryPath = path.join(__dirname, 'known-agents.json');
  try {
    return JSON.parse(fs.readFileSync(registryPath, 'utf8'));
  } catch {
    return { agents: {}, humans: {} };
  }
}

// --- Check if user is a registered ecosystem participant ---
// Only registered agents and humans in known-agents.json can vote.
function lookupParticipant(registry, username) {
  const lower = username.toLowerCase();
  // Check agents
  for (const [name, entry] of Object.entries(registry.agents || {})) {
    if (name.toLowerCase() === lower) {
      return { type: 'agent', stake: entry.stake || '0', did: entry.did, tier: entry.tier };
    }
  }
  // Check humans
  for (const [name, entry] of Object.entries(registry.humans || {})) {
    if (name.toLowerCase() === lower) {
      return { type: 'human', stake: entry.stake || '0', role: entry.role };
    }
  }
  return null; // Not registered — cannot vote
}

// --- Quadratic weight: sqrt(stake), mirrors x/gov/keeper/msg_server.go:214 ---
// Minimum weight is 1 for registered participants with zero or no stake.
function quadraticWeight(stakeStr) {
  if (!stakeStr || stakeStr === '0') return 1;
  const stake = BigInt(stakeStr);
  if (stake <= 0n) return 1;
  // Integer square root via Newton's method
  let x = stake;
  let y = (x + 1n) / 2n;
  while (y < x) {
    x = y;
    y = (x + stake / x) / 2n;
  }
  return Number(x) || 1;
}

// --- Parse a single comment for a /vote command ---
function parseVote(commentBody) {
  const lines = commentBody.split('\n').map(l => l.trim());

  let direction = null;
  let justification = null;
  let agentDid = null;

  for (const line of lines) {
    const voteMatch = line.match(/^\/vote\s+(support|oppose|abstain)\s*$/i);
    if (voteMatch) {
      direction = voteMatch[1].toLowerCase();
    }

    const justMatch = line.match(/^Justification:\s*(.+)$/i);
    if (justMatch) {
      justification = justMatch[1].trim();
    }

    const didMatch = line.match(/^Agent-DID:\s*(did:lgm:\S+)\s*$/i);
    if (didMatch) {
      agentDid = didMatch[1];
    }
  }

  if (!direction) return null;

  return { direction, justification, agentDid };
}

// --- Check account age ---
function isAccountOldEnough(userCreatedAt) {
  const created = new Date(userCreatedAt);
  const now = new Date();
  const diffDays = (now - created) / (1000 * 60 * 60 * 24);
  return diffDays >= MIN_ACCOUNT_AGE_DAYS;
}

// --- Build tally markdown ---
function buildTallyComment(votes, issueAuthor) {
  const registry = loadKnownAgents();

  const validVotes = [];
  const rejectedVotes = [];

  for (const [user, vote] of Object.entries(votes)) {
    // Registry check — only ecosystem participants can vote
    const participant = lookupParticipant(registry, user);
    if (!participant) {
      rejectedVotes.push({ user, reason: 'Not a registered ecosystem participant (see scripts/governance/known-agents.json)' });
      continue;
    }

    // Self-vote check
    if (user.toLowerCase() === issueAuthor.toLowerCase()) {
      rejectedVotes.push({ user, reason: 'Self-voting not allowed' });
      continue;
    }

    // Justification check
    if (!vote.justification || vote.justification.length < MIN_JUSTIFICATION_LENGTH) {
      rejectedVotes.push({ user, reason: `Justification too short (need ${MIN_JUSTIFICATION_LENGTH}+ chars)` });
      continue;
    }

    // Account age check
    if (!vote.accountOldEnough) {
      rejectedVotes.push({ user, reason: `Account younger than ${MIN_ACCOUNT_AGE_DAYS} days` });
      continue;
    }

    // Stake-weighted quadratic vote — weight = sqrt(stake)
    const stakeWeight = quadraticWeight(participant.stake);

    validVotes.push({
      user,
      direction: vote.direction,
      justification: vote.justification,
      agentDid: vote.agentDid || participant.did || null,
      weight: stakeWeight,
      participantType: participant.type,
    });
  }

  // Compute tallies
  let supportWeight = 0;
  let opposeWeight = 0;
  let abstainWeight = 0;

  for (const v of validVotes) {
    if (v.direction === 'support') supportWeight += v.weight;
    else if (v.direction === 'oppose') opposeWeight += v.weight;
    else if (v.direction === 'abstain') abstainWeight += v.weight;
  }

  const totalWeight = supportWeight + opposeWeight + abstainWeight;
  const totalVotes = validVotes.length;
  const quorumMet = totalVotes >= QUORUM_MIN_VOTES;
  const supportPct = totalWeight > 0 ? Math.round((supportWeight / (supportWeight + opposeWeight)) * 10000) : 0;
  const supportPctDisplay = totalWeight > 0 ? (supportPct / 100).toFixed(1) : '0.0';

  // Determine status
  let status;
  if (!quorumMet) {
    status = 'Voting Open — Quorum Not Met';
  } else if (supportPct >= SUPPORT_THRESHOLD_BPS) {
    status = 'Support Majority';
  } else if (supportPct >= 4000) {
    status = 'Contested';
  } else {
    status = 'Opposed';
  }

  // Build markdown
  let md = `${TALLY_SENTINEL}\n`;
  md += `## Vote Tally\n\n`;
  md += `| Metric | Value |\n|--------|-------|\n`;
  md += `| Status | **${status}** |\n`;
  md += `| Total votes | ${totalVotes} / ${QUORUM_MIN_VOTES} quorum |\n`;
  md += `| Support | ${supportWeight} weight (${supportPctDisplay}%) |\n`;
  md += `| Oppose | ${opposeWeight} weight |\n`;
  md += `| Abstain | ${abstainWeight} weight |\n\n`;

  if (validVotes.length > 0) {
    md += `### Votes\n\n`;
    md += `| Voter | Type | Direction | Weight | Justification |\n`;
    md += `|-------|------|-----------|--------|----------------|\n`;
    for (const v of validVotes) {
      const did = v.agentDid ? ` (${v.agentDid.slice(0, 20)}...)` : '';
      const justShort = v.justification.length > 80
        ? v.justification.slice(0, 77) + '...'
        : v.justification;
      md += `| @${v.user}${did} | ${v.participantType} | ${v.direction} | ${v.weight} | ${justShort} |\n`;
    }
    md += '\n';
  }

  if (rejectedVotes.length > 0) {
    md += `<details><summary>Rejected votes (${rejectedVotes.length})</summary>\n\n`;
    for (const r of rejectedVotes) {
      md += `- @${r.user}: ${r.reason}\n`;
    }
    md += `\n</details>\n\n`;
  }

  md += `---\n`;
  md += `*Only registered ecosystem participants can vote. Quadratic voting: weight = sqrt(stake).*\n`;
  md += `*Thresholds: ${SUPPORT_THRESHOLD_BPS / 100}% support, ${QUORUM_MIN_VOTES} vote quorum. Abstentions count toward quorum but not support/oppose ratio.*\n`;
  md += `*Register in \`scripts/governance/known-agents.json\` to participate.*\n`;

  // Labels to set
  const labels = [];
  if (quorumMet) labels.push('vote:quorum-met');
  if (quorumMet && supportPct >= SUPPORT_THRESHOLD_BPS) labels.push('vote:support-majority');
  else if (quorumMet && supportPct >= 4000) labels.push('vote:contested');
  else if (quorumMet) labels.push('vote:opposed');

  return { markdown: md, labels, status, quorumMet, supportPct };
}

// --- Main ---
async function run() {
  try {
    const token = process.env.GITHUB_TOKEN;
    if (!token) throw new Error('GITHUB_TOKEN not set');

    const octokit = github.getOctokit(token);
    const { owner, repo } = github.context.repo;
    const issueNumber = github.context.issue.number;

    if (!issueNumber) {
      core.info('No issue number in context, skipping');
      return;
    }

    // Get issue details
    const { data: issue } = await octokit.rest.issues.get({
      owner, repo, issue_number: issueNumber,
    });

    // Only process issues with 'proposal' label
    const hasProposalLabel = issue.labels.some(l =>
      (typeof l === 'string' ? l : l.name) === 'proposal'
    );
    if (!hasProposalLabel) {
      core.info('Issue does not have "proposal" label, skipping');
      return;
    }

    // Get all comments
    const comments = await octokit.paginate(octokit.rest.issues.listComments, {
      owner, repo, issue_number: issueNumber, per_page: 100,
    });

    // Parse votes — last vote per user wins (deduplication)
    const votes = {};
    for (const comment of comments) {
      // Skip the tally bot's own comment
      if (comment.body && comment.body.includes(TALLY_SENTINEL)) continue;

      const parsed = parseVote(comment.body || '');
      if (!parsed) continue;

      const user = comment.user.login;

      // Check account age
      const { data: userData } = await octokit.rest.users.getByUsername({ username: user });
      const accountOldEnough = isAccountOldEnough(userData.created_at);

      votes[user] = {
        direction: parsed.direction,
        justification: parsed.justification,
        agentDid: parsed.agentDid,
        accountOldEnough,
        commentId: comment.id,
      };
    }

    // Build tally
    const issueAuthor = issue.user.login;
    const tally = buildTallyComment(votes, issueAuthor);

    // Find existing tally comment
    let tallyCommentId = null;
    for (const comment of comments) {
      if (comment.body && comment.body.startsWith(TALLY_SENTINEL)) {
        tallyCommentId = comment.id;
        break;
      }
    }

    // Upsert tally comment
    if (tallyCommentId) {
      await octokit.rest.issues.updateComment({
        owner, repo, comment_id: tallyCommentId, body: tally.markdown,
      });
    } else {
      await octokit.rest.issues.createComment({
        owner, repo, issue_number: issueNumber, body: tally.markdown,
      });
    }

    // Update labels
    const voteLabels = ['vote:quorum-met', 'vote:support-majority', 'vote:contested', 'vote:opposed'];
    const currentLabels = issue.labels.map(l => typeof l === 'string' ? l : l.name);

    // Remove old vote labels
    for (const label of voteLabels) {
      if (currentLabels.includes(label) && !tally.labels.includes(label)) {
        try {
          await octokit.rest.issues.removeLabel({
            owner, repo, issue_number: issueNumber, name: label,
          });
        } catch {
          // Label might not exist yet
        }
      }
    }

    // Add new vote labels
    for (const label of tally.labels) {
      if (!currentLabels.includes(label)) {
        await octokit.rest.issues.addLabels({
          owner, repo, issue_number: issueNumber, labels: [label],
        });
      }
    }

    // Check if ready for review (quorum + support + 48h elapsed)
    const createdAt = new Date(issue.created_at);
    const hoursElapsed = (Date.now() - createdAt.getTime()) / (1000 * 60 * 60);
    const discussionPeriodMet = hoursElapsed >= 48;

    if (tally.quorumMet && tally.supportPct >= SUPPORT_THRESHOLD_BPS && discussionPeriodMet) {
      if (!currentLabels.includes('ready-for-review')) {
        await octokit.rest.issues.addLabels({
          owner, repo, issue_number: issueNumber, labels: ['ready-for-review'],
        });
      }
    }

    core.setOutput('status', tally.status);
    core.setOutput('quorum_met', tally.quorumMet);
    core.setOutput('support_pct', tally.supportPct);
    core.info(`Tally complete: ${tally.status}`);
  } catch (error) {
    core.setFailed(error.message);
  }
}

run();
